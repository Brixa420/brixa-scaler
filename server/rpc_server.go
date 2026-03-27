package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

/*
 ╔══════════════════════════════════════════════════════════════════════╗
 ║                     BRIXA SCALER RPC SERVER                          ║
 ║            Full Pipeline: RPC → Batching → ZK → Settlement          ║
 ╚══════════════════════════════════════════════════════════════════════╝
*/

type Transaction struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Value uint64 `json:"value"`
	Nonce uint64 `json:"nonce"`
	TS    int64  `json:"timestamp"`
}

type Batch struct {
	ID           int
	Transactions []Transaction
}

type Proof struct {
	BatchID int    `json:"batch_id"`
	Proof   string `json:"proof"`
	Valid   bool   `json:"valid"`
}

type Block struct {
	BlockNum uint64   `json:"block_num"`
	Proofs   []Proof `json:"proofs"`
	TS       int64   `json:"timestamp"`
}

const (
	BATCH_SIZE    = 1000
	BATCH_TIMEOUT = 100 * time.Millisecond
	ZK_PROVE_MS   = 385
	MAX_PROVERS   = 100
	SHARDS        = 10
)

type Server struct {
	mu          sync.Mutex
	pendingTx   []Transaction
	proverChan  chan *Batch
	proofChan   chan *Proof
	shards      []*Shard
	txReceived  uint64
	txBatched   uint64
	proofsMade  uint64
	blocksSettled uint64
}

type Shard struct {
	id        int
	proofChan chan *Proof
	settled   uint64
}

func NewServer() *Server {
	s := &Server{
		proverChan: make(chan *Batch, MAX_PROVERS*10),
		proofChan:  make(chan *Proof, 10000),
		shards:     make([]*Shard, SHARDS),
	}
	for i := 0; i < SHARDS; i++ {
		s.shards[i] = &Shard{id: i, proofChan: make(chan *Proof, 1000)}
	}
	return s
}

type SubmitRequest struct {
	Transactions []Transaction `json:"transactions"`
}

type SubmitResponse struct {
	BatchID   string `json:"batch_id"`
	Timestamp int64  `json:"timestamp"`
}

type StatusResponse struct {
	TxReceived    uint64 `json:"tx_received"`
	TxBatched     uint64 `json:"tx_batched"`
	ProofsMade    uint64 `json:"proofs_made"`
	BlocksSettled uint64 `json:"blocks_settled"`
	PendingTx     int    `json:"pending_tx"`
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	s.mu.Lock()
	s.txReceived += uint64(len(req.Transactions))
	s.pendingTx = append(s.pendingTx, req.Transactions...)
	pending := len(s.pendingTx)
	s.mu.Unlock()
	
	if pending >= BATCH_SIZE {
		go s.createBatch()
	}
	
	json.NewEncoder(w).Encode(SubmitResponse{
		BatchID:   fmt.Sprintf("batch_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UnixMilli(),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	json.NewEncoder(w).Encode(StatusResponse{
		TxReceived:    s.txReceived,
		TxBatched:     s.txBatched,
		ProofsMade:    s.proofsMade,
		BlocksSettled: s.blocksSettled,
		PendingTx:     len(s.pendingTx),
	})
	s.mu.Unlock()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) createBatch() {
	s.mu.Lock()
	if len(s.pendingTx) < BATCH_SIZE {
		s.mu.Unlock()
		return
	}
	txs := s.pendingTx[:BATCH_SIZE]
	s.pendingTx = s.pendingTx[BATCH_SIZE:]
	s.txBatched += BATCH_SIZE
	s.mu.Unlock()
	
	select {
	case s.proverChan <- &Batch{Transactions: txs}:
	default:
	}
}

func (s *Server) runBatcher() {
	ticker := time.NewTicker(BATCH_TIMEOUT)
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			if len(s.pendingTx) >= 100 {
				s.mu.Unlock()
				s.createBatch()
			} else {
				s.mu.Unlock()
			}
		}
	}
}

func (s *Server) runProverPool() {
	var wg sync.WaitGroup
	for i := 0; i < MAX_PROVERS; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range s.proverChan {
				time.Sleep(ZK_PROVE_MS * time.Millisecond)
				s.proofChan <- &Proof{BatchID: batch.ID, Valid: true}
				s.mu.Lock()
				s.proofsMade++
				s.mu.Unlock()
			}
		}()
	}
	wg.Wait()
}

func (s *Server) runSettlement() {
	shardIdx := 0
	for proof := range s.proofChan {
		s.shards[shardIdx].proofChan <- proof
		shardIdx = (shardIdx + 1) % SHARDS
		s.mu.Lock()
		if s.proofsMade%100 == 0 {
			s.blocksSettled++
		}
		s.mu.Unlock()
	}
}

func (s *Server) runShards() {
	var wg sync.WaitGroup
	for _, shard := range s.shards {
		wg.Add(1)
		go func(shard *Shard) {
			defer wg.Done()
			for range shard.proofChan {
				time.Sleep(10 * time.Millisecond)
				shard.settled++
			}
		}(shard)
	}
	wg.Wait()
}

func main() {
	fmt.Println(`
╔══════════════════════════════════════════════════════════════════════╗
║                     BRIXA SCALER RPC SERVER                          ║
║            Full Pipeline: RPC → Batching → ZK → Settlement          ║
╚══════════════════════════════════════════════════════════════════════╝
`)
	
	srv := NewServer()
	go srv.runBatcher()
	go srv.runProverPool()
	go srv.runSettlement()
	go srv.runShards()
	
	http.HandleFunc("/health", srv.handleHealth)
	http.HandleFunc("/submit", srv.handleSubmit)
	http.HandleFunc("/status", srv.handleStatus)
	
	go func() {
		for {
			time.Sleep(5 * time.Second)
			log.Printf("[STATS] Rcv:%d | Batch:%d | Prove:%d | Settle:%d",
				srv.txReceived, srv.txBatched, srv.proofsMade, srv.blocksSettled)
		}
	}()
	
	log.Println("🚀 Server on :8080 | /submit /status /health")
	http.ListenAndServe(":8080", nil)
}
