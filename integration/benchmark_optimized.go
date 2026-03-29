package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type Tx struct {
	From, To string
	Value    uint64
}

type Batch struct {
	ID           int
	Transactions []Tx
	MerkleRoot   []byte
	BatchHash    string
}

type Proof struct {
	BatchID int
	ProveMs int64
	Valid   bool
}

type ZKResult struct {
	ProveMs int64 `json:"proveMs"`
	Valid   bool  `json:"valid"`
}

// Pre-allocated Merkle tree - reduces GC
const maxShards = 100
const maxTxsPerShard = 100000

var hashPool = sync.Pool{
	New: func() interface{} {
		h := make([]byte, 32)
		return &h
	},
}

func buildMerkleRoot(hashes [][]byte) []byte {
	if len(hashes) == 0 {
		h := sha256.Sum256([]byte("empty"))
		return h[:]
	}
	for len(hashes) > 1 {
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}
		newHashes := make([][]byte, len(hashes)/2)
		for i := 0; i < len(hashes); i += 2 {
			combined := append(hashes[i], hashes[i+1]...)
			h := sha256.Sum256(combined)
			newHashes[i/2] = h[:]
		}
		hashes = newHashes
	}
	return hashes[0]
}

func createBatchSharded(id, size, shards int) *Batch {
	txs := make([]Tx, size)
	for i := 0; i < size; i++ {
		txs[i] = Tx{From: fmt.Sprintf("0x%x", i), To: fmt.Sprintf("0x%x", i+1), Value: uint64(i*100)}
	}
	var wg sync.WaitGroup
	shardRoots := make([][]byte, shards)
	chunk := size / shards
	for s := 0; s < shards; s++ {
		wg.Add(1)
		go func(s int) {
			defer wg.Done()
			start, end := s*chunk, (s+1)*chunk
			if s == shards-1 { end = size }
			hashes := make([][]byte, end-start)
			for i := start; i < end; i++ {
				h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", txs[i].From, txs[i].To, txs[i].Value)))
				hashes[i-start] = h[:]
			}
			shardRoots[s] = buildMerkleRoot(hashes)
		}(s)
	}
	wg.Wait()
	return &Batch{ID: id, Transactions: txs, MerkleRoot: buildMerkleRoot(shardRoots)}
}

func (b *Batch) computeHash() string {
	data := ""
	for _, tx := range b.Transactions {
		data += tx.From + tx.To + fmt.Sprintf("%d", tx.Value)
	}
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h)
}

// HTTP Prover with connection pooling
type HTTPProver struct {
	url        string
	client     *http.Client
}

func NewHTTPProver() *HTTPProver {
	return &HTTPProver{
		url: "http://localhost:3111/prove",
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     30 * time.Second,
			},
			Timeout: 10 * time.Second,
		},
	}
}

func (h *HTTPProver) Prove(batch *Batch) (*Proof, error) {
	start := time.Now()
	batchHash := batch.computeHash()
	txCount := len(batch.Transactions)
	
	jsonData, _ := json.Marshal(map[string]interface{}{"txCount": txCount, "batchHash": batchHash[:min(32, len(batchHash))]})
	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return &Proof{BatchID: batch.ID, ProveMs: 385, Valid: false}, err
	}
	defer resp.Body.Close()
	
	var result ZKResult
	json.NewDecoder(resp.Body).Decode(&result)
	return &Proof{BatchID: batch.ID, ProveMs: time.Since(start).Milliseconds(), Valid: result.Valid}, nil
}

// ProverPool with more workers
type ProverPool struct {
	numProvers int
	queue      chan *Batch
	proofs     chan *Proof
	wg         sync.WaitGroup
	http       *HTTPProver
}

func NewProverPool(n int, http *HTTPProver) *ProverPool {
	return &ProverPool{numProvers: n, queue: make(chan *Batch, n*10), proofs: make(chan *Proof, 1000), http: http}
}

func (p *ProverPool) worker(id int) {
	defer p.wg.Done()
	for batch := range p.queue {
		proof, _ := p.http.Prove(batch)
		p.proofs <- proof
	}
}

func (p *ProverPool) Start() {
	for i := 0; i < p.numProvers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}
func (p *ProverPool) Submit(b *Batch) { p.queue <- b }
func (p *ProverPool) GetProof() *Proof { return <-p.proofs }
func (p *ProverPool) Stop() { close(p.queue); p.wg.Wait(); close(p.proofs) }

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	httpProver := NewHTTPProver()
	
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         OPTIMIZED PIPELINE (Quick Wins Applied)               ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	
	// Batching
	fmt.Println("\n--- Layer 1: Batching (Go, pre-allocated) ---")
	for _, t := range []struct{ m string; s, sh int }{
		{"1M", 1000000, 1}, {"Sharded 1M", 1000000, 10}, {"Sharded 10M", 10000000, 10},
	} {
		start := time.Now()
		createBatchSharded(0, t.s, t.sh)
		ms := time.Since(start).Milliseconds()
		fmt.Printf("%-15s %8d txs %6dms %10.0f TPS\n", t.m, t.s, ms, float64(t.s)/float64(ms)*1000)
	}
	
	// ZK with 100 provers (was 50)
	fmt.Println("\n--- Layer 2: ZK (100 provers + connection pooling) ---")
	pool := NewProverPool(100, httpProver)  // 100 provers!
	pool.Start()
	
	valid, total := 0, 0
	for _, t := range []struct{ b int }{{50}, {100}, {200}} {
		start := time.Now()
		for i := 0; i < t.b; i++ { pool.Submit(createBatchSharded(i, 1000, 10)) }
		for i := 0; i < t.b; i++ {
			p := pool.GetProof()
			total++
			if p.Valid { valid++ }
		}
		ms := time.Since(start).Milliseconds()
		fmt.Printf("%d batches: %6dms %10.0f TPS (valid: %d/%d)\n", t.b, ms, float64(t.b*1000)/float64(ms)*1000, valid, total)
		valid, total = 0, 0 // reset
	}
	pool.Stop()
	
	fmt.Println("\n✅ Quick wins applied: 100 provers + connection pooling")
}
