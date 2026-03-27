package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// ============================================
// CLUSTER NODE - Multi-Node Horizontal Scaling
// ============================================

type ClusterNode struct {
	id           string
	address      string
	port         int
	isPrimary    bool
	peers        map[string]*Peer
	merkleTree   *MerkleTree
	txQueue      chan Transaction
	batchQueue  chan []Transaction
	mu           sync.RWMutex
	stats        ClusterStats
}

type Peer struct {
	id      string
	address string
	status  string // "connected", "disconnected"
	lastPing time.Time
}

type ClusterStats struct {
	NodeID        string    `json:"node_id"`
	IsPrimary     bool      `json:"is_primary"`
	PeerCount     int       `json:"peer_count"`
	TotalTPS      float64   `json:"total_tps"`
	Transactions   int64     `json:"transactions"`
	Proofs        int64     `json:"proofs"`
	Uptime        float64   `json:"uptime_seconds"`
	StartTime     time.Time `json:"start_time"`
}

type ClusterMessage struct {
	Type      string      `json:"type"` // "tx", "batch", "sync", "ping"
	From      string      `json:"from"`
	To        string      `json:"to"`
	Payload   interface{} `json:"payload"`
	Timestamp int64       `json:"timestamp"`
}

// NewClusterNode creates a new cluster node
func NewClusterNode(id string, port int, isPrimary bool) *ClusterNode {
	node := &ClusterNode{
		id:         id,
		address:    fmt.Sprintf("localhost:%d", port),
		port:       port,
		isPrimary:  isPrimary,
		peers:      make(map[string]*Peer),
		merkleTree: NewMerkleTree(),
		txQueue:    make(chan Transaction, 100000),
		batchQueue: make(chan []Transaction, 1000),
		stats: ClusterStats{
			NodeID:    id,
			IsPrimary: isPrimary,
			StartTime: time.Now(),
		},
	}

	// Start workers
	go node.processTransactions()
	go node.processBatches()
	go node.healthCheck()

	return node
}

// AddPeer adds a peer to the cluster
func (n *ClusterNode) AddPeer(id, address string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.peers[id] = &Peer{
		id:        id,
		address:   address,
		status:    "connected",
		lastPing:  time.Now(),
	}

	log.Printf("Node %s: Added peer %s at %s", n.id, id, address)
}

// RemovePeer removes a peer from the cluster
func (n *ClusterNode) RemovePeer(id string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	delete(n.peers, id)
	log.Printf("Node %s: Removed peer %s", n.id, id)
}

// Broadcast sends a message to all peers
func (n *ClusterNode) Broadcast(msg ClusterMessage) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	msg.From = n.id
	msg.Timestamp = time.Now().UnixMilli()

	for id, peer := range n.peers {
		if peer.status == "connected" {
			go n.sendToPeer(peer.address, msg)
		}
	}
}

// sendToPeer sends a message to a specific peer
func (n *ClusterNode) sendToPeer(address string, msg ClusterMessage) {
	// In production, use HTTP/gRPC to send to peer
	// Simplified: just log for now
	log.Printf("Node %s: Would send to %s: %s", n.id, address, msg.Type)
}

// processTransactions processes transactions from the queue
func (n *ClusterNode) processTransactions() {
	for {
		select {
		case tx := <-n.txQueue:
			n.merkleTree.Insert(tx.Hash)
			n.stats.Transactions++

			// If primary, broadcast to other nodes
			if n.isPrimary {
				msg := ClusterMessage{
					Type:    "tx",
					Payload: tx,
				}
				n.Broadcast(msg)
			}

		case batch := <-n.batchQueue:
			// Generate proof for batch
			root := n.merkleTree.Root()
			proof := n.merkleTree.Proof(0)

			// Broadcast proof to peers
			if n.isPrimary {
				msg := ClusterMessage{
					Type: "batch",
					Payload: map[string]interface{}{
						"root":  root,
						"proof": proof,
						"count": len(batch),
					},
				}
				n.Broadcast(msg)
			}

			n.stats.Proofs++
		}
	}
}

// processBatches creates batches from transactions
func (n *ClusterNode) processBatches() {
	batch := make([]Transaction, 0, 1000)
	ticker := time.NewTicker(1000 * time.Millisecond)

	for {
		select {
		case tx := <-n.txQueue:
			batch = append(batch, tx)
			if len(batch) >= 1000 {
				n.batchQueue <- batch
				batch = make([]Transaction, 0, 1000)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				n.batchQueue <- batch
				batch = make([]Transaction, 0, 1000)
			}
		}
	}
}

// healthCheck monitors peer health
func (n *ClusterNode) healthCheck() {
	ticker := time.NewTicker(10 * time.Second)

	for range ticker.C {
		n.mu.Lock()
		for id, peer := range n.peers {
			if time.Since(peer.lastPing) > 30*time.Second {
				peer.status = "disconnected"
				log.Printf("Node %s: Peer %s is unreachable", n.id, id)
			}
		}
		n.mu.Unlock()

		// Update stats
		n.stats.PeerCount = len(n.peers)
		n.stats.Uptime = time.Since(n.stats.StartTime).Seconds()

		// Calculate TPS
		if n.stats.Uptime > 0 {
			n.stats.TotalTPS = float64(n.stats.Transactions) / n.stats.Uptime
		}
	}
}

// HandleHTTP handles HTTP requests for cluster operations
func (n *ClusterNode) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Return cluster status
		n.mu.RLock()
		stats := n.stats
		n.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)

	case "POST":
		// Add peer
		var req struct {
			ID      string `json:"id"`
			Address string `json:"address"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		n.AddPeer(req.ID, req.Address)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "peer added"})

	case "DELETE":
		// Remove peer
		vars := mux.Vars(r)
		n.RemovePeer(vars["id"])
		w.WriteHeader(http.StatusOK)
		json.Marshal(map[string]string{"status": "peer removed"})
	}
}

// ============================================
// CLUSTER SERVER - HTTP API for Cluster Management
// ============================================

type ClusterServer struct {
	nodes    map[string]*ClusterNode
	primary  *ClusterNode
	mu       sync.RWMutex
	router   *mux.Router
}

func NewClusterServer() *ClusterServer {
	cs := &ClusterServer{
		nodes:   make(map[string]*ClusterNode),
		router:  mux.NewRouter(),
	}

	// Register routes
	cs.router.HandleFunc("/cluster/nodes", cs.listNodes).Methods("GET")
	cs.router.HandleFunc("/cluster/nodes", cs.addNode).Methods("POST")
	cs.router.HandleFunc("/cluster/nodes/{id}", cs.getNode).Methods("GET")
	cs.router.HandleFunc("/cluster/nodes/{id}", cs.removeNode).Methods("DELETE")
	cs.router.HandleFunc("/cluster/submit", cs.submitTransaction).Methods("POST")
	cs.router.HandleFunc("/cluster/batch", cs.getBatch).Methods("GET")
	cs.router.HandleFunc("/cluster/stats", cs.getStats).Methods("GET")

	return cs
}

func (cs *ClusterServer) listNodes(w http.ResponseWriter, r *http.Request) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	nodes := []map[string]interface{}{}
	for id, node := range cs.nodes {
		nodes = append(nodes, map[string]interface{}{
			"id":         id,
			"address":    node.address,
			"is_primary": node.isPrimary,
			"peer_count": len(node.peers),
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": nodes,
		"count": len(nodes),
	})
}

func (cs *ClusterServer) addNode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string `json:"id"`
		Port     int    `json:"port"`
		IsPrimary bool  `json:"is_primary"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.ID == "" {
		req.ID = fmt.Sprintf("node-%d", len(cs.nodes)+1)
	}
	if req.Port == 0 {
		req.Port = 9000 + len(cs.nodes)
	}

	node := NewClusterNode(req.ID, req.Port, req.IsPrimary)

	cs.mu.Lock()
	cs.nodes[req.ID] = node

	if req.IsPrimary {
		cs.primary = node
	}
	cs.mu.Unlock()

	log.Printf("Added cluster node: %s at localhost:%d (primary: %v)", req.ID, req.Port, req.IsPrimary)

	json.NewEncoder(w).Encode(map[string]string{"status": "node added", "id": req.ID})
}

func (cs *ClusterServer) getNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	cs.mu.RLock()
	node, ok := cs.nodes[id]
	cs.mu.RUnlock()

	if !ok {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	node.mu.RLock()
	stats := node.stats
	node.mu.RUnlock()

	json.NewEncoder(w).Encode(stats)
}

func (cs *ClusterServer) removeNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	cs.mu.Lock()
	delete(cs.nodes, id)
	cs.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "node removed"})
}

func (cs *ClusterServer) submitTransaction(w http.ResponseWriter, r *http.Request) {
	var tx Transaction
	json.NewDecoder(r.Body).Decode(&tx)

	cs.mu.RLock()
	primary := cs.primary
	cs.mu.RUnlock()

	if primary != nil {
		primary.txQueue <- tx
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

func (cs *ClusterServer) getBatch(w http.ResponseWriter, r *http.Request) {
	cs.mu.RLock()
	primary := cs.primary
	cs.mu.RUnlock()

	if primary == nil {
		http.Error(w, "No primary node", http.StatusBadRequest)
		return
	}

	select {
	case batch := <-primary.batchQueue:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"batch": batch,
			"count": len(batch),
		})
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"batch": []Transaction{},
			"count": 0,
		})
	}
}

func (cs *ClusterServer) getStats(w http.ResponseWriter, r *http.Request) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var totalTPS float64
	var totalTx int64
	var totalProofs int64

	for _, node := range cs.nodes {
		node.mu.RLock()
		totalTPS += node.stats.TotalTPS
		totalTx += node.stats.Transactions
		totalProofs += node.stats.Proofs
		node.mu.RUnlock()
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes":       len(cs.nodes),
		"total_tps":   totalTPS,
		"transactions": totalTx,
		"proofs":      totalProofs,
	})
}

func (cs *ClusterServer) Start(port int) {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Cluster server starting on %s", addr)

	cors := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}),
		handlers.AllowedHeaders([]string{"Content-Type"}),
	)

	log.Fatal(http.ListenAndServe(addr, cors(cs.router)))
}

// ============================================
// MAIN - Run Cluster Server
// ============================================

func main() {
	log.Println("═══════════════════════════════════════════════════════════")
	log.Println("    BrixaScaler Cluster Server - Multi-Node Deployment")
	log.Println("═══════════════════════════════════════════════════════════")

	// Parse flags
	port := 9000
	id := "primary"
	isPrimary := true

	// Create cluster server
	cs := NewClusterServer()

	// Add primary node
	cs.addNode(nil, nil) // Simplified - in production use proper method

	// Add secondary nodes
	for i := 1; i <= 9; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		nodePort := 9000 + i
		log.Printf("Adding node %s at localhost:%d", nodeID, nodePort)
		// In production: properly add nodes
	}

	// Handle shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down cluster server...")
		os.Exit(0)
	}()

	// Start server
	log.Printf("Starting cluster server on port %d...", port)
	cs.Start(port)
}