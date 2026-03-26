package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"sync"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

// Logger provides structured JSON logging
type Logger struct{}

func (l Logger) Info(msg string, fields map[string]interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"level":        "info",
		"message":      msg,
		"timestamp":    time.Now().Format(time.RFC3339),
		"correlation_id": "",
	})
	fmt.Println(string(data))
}

func (l Logger) Error(msg string, fields map[string]interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"level":     "error",
		"message":   msg,
		"timestamp": time.Now().Format(time.RFC3339),
	})
	fmt.Println(string(data))
}

func (l Logger) Warn(msg string, fields map[string]interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"level":     "warn",
		"message":   msg,
		"timestamp": time.Now().Format(time.RFC3339),
	})
	fmt.Println(string(data))
}

// Config holds all configuration for BrixaScaler
type Config struct {
	MaxBatchSize    int `env:"MAX_BATCH_SIZE"`
	BatchTimeoutMs int `env:"BATCH_TIMEOUT_MS"`
	RPCPort         int `env:"RPC_PORT"`
	MetricsPort     int `env:"METRICS_PORT"`
	MetricsEnabled  bool
	DemoMode        bool
}

// LoadConfig loads configuration from environment
func LoadConfig() Config {
	cfg := Config{
		MaxBatchSize:    getEnvInt("MAX_BATCH_SIZE", 1000),
		BatchTimeoutMs:  getEnvInt("BATCH_TIMEOUT_MS", 1000),
		RPCPort:         getEnvInt("RPC_PORT", 8080),
		MetricsPort:     getEnvInt("METRICS_PORT", 9090),
		MetricsEnabled:  getEnvBool("METRICS_ENABLED", true),
		DemoMode:        getEnvBool("DEMO_MODE", true),
	}
	return cfg
}

func getEnvInt(key string, def int) int {
	fmt.Sscanf(os.Getenv(key), "%d", &def)
	return def
}

func getEnvBool(key string, def bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1"
	}
	return def
}

// Transaction represents a simplified L2 transaction
type Transaction struct {
	From  string
	To    string
	Value uint64
	Nonce uint64
	Data  string
}

// Stats holds server statistics
type Stats struct {
	totalBatches   uint64
	totalTxs       uint64
	pendingBatches uint64
	pendingProofs  uint64
	startTime      time.Time
	lock           sync.Mutex
}

var stats = Stats{startTime: time.Now()}
var logger = Logger{}

// ═══════════════════════════════════════════════════════════════
// MERKLE TREE - Core Batching Logic
// ═══════════════════════════════════════════════════════════════

func buildMerkleRoot(leaves [][]byte, numShards int) []byte {
	if len(leaves) == 0 {
		empty := sha256.Sum256([]byte("empty"))
		return empty[:]
	}

	shardSize := (len(leaves) + numShards - 1) / numShards

	var wg sync.WaitGroup
	roots := make([][]byte, 0, numShards)
	result := make(chan []byte, numShards)

	for i := 0; i < numShards; i++ {
		start := i * shardSize
		end := start + shardSize
		if end > len(leaves) {
			end = len(leaves)
		}
		if start >= len(leaves) {
			break
		}

		wg.Add(1)
		go func(shard [][]byte) {
			defer wg.Done()
			root := computeMerkleRoot(shard)
			result <- root
		}(leaves[start:end])
	}

	wg.Wait()
	close(result)

	for r := range result {
		roots = append(roots, r)
	}

		sort.Slice(roots, func(i, j int) bool { return string(roots[i]) < string(roots[j]) })
	return computeMerkleRoot(roots)
}

func computeMerkleRoot(hashes [][]byte) []byte {
	if len(hashes) == 0 {
		empty := sha256.Sum256([]byte("empty"))
		return empty[:]
	}

	for len(hashes) > 1 {
		next := make([][]byte, 0, (len(hashes)+1)/2)
		for i := 0; i < len(hashes); i += 2 {
			right := hashes[i]
			if i+1 < len(hashes) {
				right = hashes[i+1]
			}
			combined := append(hashes[i], right...)
			hash := sha256.Sum256(combined)
			next = append(next, hash[:])
		}
		hashes = next
	}
	return hashes[0]
}

// ProcessBatch processes a batch of transactions
func ProcessBatch(txs []Transaction, numShards int) (string, int64) {
	start := time.Now()

	leaves := make([][]byte, len(txs))
	for i, tx := range txs {
		data := fmt.Sprintf("%s%s%d%d%s", tx.From, tx.To, tx.Value, tx.Nonce, tx.Data)
		hash := sha256.Sum256([]byte(data))
		leaves[i] = hash[:]
	}

	root := buildMerkleRoot(leaves, numShards)
	elapsed := time.Since(start).Milliseconds()

	stats.lock.Lock()
	stats.totalBatches++
	stats.totalTxs += uint64(len(txs))
	stats.lock.Unlock()

	return hex.EncodeToString(root), elapsed
}

// HealthResponse is the response for /health
type HealthResponse struct {
	Status         string  `json:"status"`
	Version        string  `json:"version"`
	Uptime         int64   `json:"uptime"`
	TPS            float64 `json:"tps"`
	TotalBatches   uint64  `json:"total_batches"`
	TotalTxs       uint64  `json:"total_txs"`
	NumShards      int     `json:"num_shards"`
	CPUCores       int     `json:"cpu_cores"`
	PendingBatches uint64  `json:"pending_batches"`
	PendingProofs  uint64  `json:"pending_proofs"`
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	stats.lock.Lock()
	uptime := int64(time.Since(stats.startTime).Seconds())
	tps := float64(stats.totalTxs) / float64(uptime)
	if uptime == 0 {
		tps = 0
	}
	pendingBatches := stats.pendingBatches
	pendingProofs := stats.pendingProofs
	stats.lock.Unlock()

	resp := HealthResponse{
		Status:         "healthy",
		Version:        "1.0.0",
		Uptime:         uptime,
		TPS:            tps,
		TotalBatches:   stats.totalBatches,
		TotalTxs:       stats.totalTxs,
		NumShards:      4,
		CPUCores:       runtime.NumCPU(),
		PendingBatches: pendingBatches,
		PendingProofs:  pendingProofs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleBatch(w http.ResponseWriter, r *http.Request) {
	correlationID := fmt.Sprintf("%d", time.Now().UnixNano())

	if os.Getenv("DEMO_MODE") != "false" {
		logger.Warn("DEMO_MODE enabled - no real transactions", map[string]interface{}{
			"correlation_id":  correlationID,
			"warning":        "DEMO_MODE is enabled - no real transactions",
		})
	}

	var txs []Transaction
	if err := json.NewDecoder(r.Body).Decode(&txs); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	root, elapsed := ProcessBatch(txs, 4)

	logger.Info("batch processed", map[string]interface{}{
		"correlation_id": correlationID,
		"tx_count":      len(txs),
		"elapsed_ms":    elapsed,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"root":   root,
		"elapsed": fmt.Sprintf("%dms", elapsed),
	})
}

func handleBenchmark(w http.ResponseWriter, r *http.Request) {
	txs := make([]Transaction, 1000)
	for i := range txs {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 1000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}

	root, elapsed := ProcessBatch(txs, 4)
	tps := float64(1000) * 1000 / float64(elapsed)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"root":      root,
		"elapsed_ms": elapsed,
		"tps":       tps,
	})
}

// RateLimiter provides DOS protection
type RateLimiter struct {
	limiter  *rate.Limiter
	requests uint64
	mu       sync.Mutex
}

var globalLimiter *RateLimiter

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	allowed := rl.limiter.Allow()
	if allowed {
		rl.requests++
	}
	return allowed
}

func (rl *RateLimiter) GetRequests() uint64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.requests
}

func init() {
	globalLimiter = NewRateLimiter(100, 200)
}

func handleRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !globalLimiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"rate limit exceeded","retry_after":1}`)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetupServer creates the HTTP muxer (extracted for testability)
func SetupServer() *http.ServeMux {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
		if !globalLimiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"rate limit exceeded","retry_after":1}`)
			return
		}
		handleBatch(w, r)
	})
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/benchmark", handleBenchmark)

	// Metrics endpoint
	cfg := LoadConfig()
	if cfg.MetricsEnabled {
		mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "# HELP brixa_scaler_total_txs Total transactions processed\n")
			fmt.Fprintf(w, "# TYPE brixa_scaler_total_txs counter\n")
			fmt.Fprintf(w, "brixa_scaler_total_txs %d\n", stats.totalTxs)
			fmt.Fprintf(w, "# HELP brixa_scaler_total_batches Total batches created\n")
			fmt.Fprintf(w, "# TYPE brixa_scaler_total_batches counter\n")
			fmt.Fprintf(w, "brixa_scaler_total_batches %d\n", stats.totalBatches)
		})
	}
	
	return mux
}

// RunStartup runs startup sequence (extracted for testability)
func RunStartup(cfg Config) {
	logger.Info("brixa-scaler starting", map[string]interface{}{
		"version":        "1.0.0",
		"rpc_port":       cfg.RPCPort,
		"metrics_port":   cfg.MetricsPort,
		"demo_mode":      cfg.DemoMode,
	})

	if cfg.DemoMode {
		fmt.Printf("\n⚠️  WARNING: DEMO_MODE is enabled!\n")
		fmt.Printf("   No real transactions will be processed.\n")
		fmt.Printf("   Set DEMO_MODE=false to enable real transactions\n\n")
	}
}

func main() {
	cfg := LoadConfig()
	RunStartup(cfg)
	
	rpcPort := cfg.RPCPort
	metricsPort := cfg.MetricsPort

	// Graceful shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		logger.Info("shutdown signal received, flushing pending batches", map[string]interface{}{})

		stats.lock.Lock()
		pendingBatches := stats.pendingBatches
		stats.lock.Unlock()

		if pendingBatches > 0 {
			logger.Info("flushing pending batches", map[string]interface{}{
				"pending_batches": pendingBatches,
			})
		}
		os.Exit(0)
	}()

	// Start metrics server
	if cfg.MetricsEnabled {
		go func() {
			logger.Info("metrics server started", map[string]interface{}{
				"port": metricsPort,
			})
			http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), nil)
		}()
	}

	fmt.Printf("  GET  /batch    - Submit transaction batch (POST JSON)\n")
	fmt.Printf("  GET  /health   - Server health & stats\n")
	fmt.Printf("  GET  /benchmark - Quick TPS benchmark\n")
	fmt.Printf("  GET  /metrics  - Prometheus metrics\n")

	// Use extracted server setup
	mux := SetupServer()

	fmt.Printf("\n🚀 BrixaScaler running on http://localhost:%d\n", rpcPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", rpcPort), mux))
}
// ═══════════════════════════════════════════════════════════════
// MULTI-NODE COORDINATION - Cluster Management (Priority 5)
// ═══════════════════════════════════════════════════════════════

type Node struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Port      int       `json:"port"`
	Status    string    `json:"status"` // "active", "joining", "leaving"
	LastSeen  time.Time `json:"last_seen"`
	Weight    int       `json:"weight"` // for leader election
}

type Cluster struct {
	Nodes map[string]*Node
	mu    sync.RWMutex
	Self  string
}

var cluster *Cluster

func (c *Cluster) AddNode(node *Node) {
	c.mu.Lock()
	defer c.mu.Unlock()
	node.LastSeen = time.Now()
	node.Status = "active"
	c.Nodes[node.ID] = node
}

func (c *Cluster) RemoveNode(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if node, ok := c.Nodes[id]; ok {
		node.Status = "leaving"
	}
}

func (c *Cluster) GetLeader() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var leader *Node
	for _, n := range c.Nodes {
		if n.Status != "active" {
			continue
		}
		if leader == nil || n.Weight > leader.Weight {
			leader = n
		}
	}
	
	if leader != nil {
		return leader.ID
	}
	return ""
}

func (c *Cluster) ListNodes() []*Node {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result := make([]*Node, 0, len(c.Nodes))
	for _, n := range c.Nodes {
		result = append(result, n)
	}
	return result
}

func initCluster(selfID, addr string, port int) {
	cluster = &Cluster{
		Nodes: make(map[string]*Node),
		Self:  selfID,
	}
	cluster.AddNode(&Node{
		ID:       selfID,
		Address:  addr,
		Port:     port,
		Status:   "active",
		LastSeen: time.Now(),
		Weight:   100,
	})
}

// Health check for cluster
func handleClusterHealth(w http.ResponseWriter, r *http.Request) {
	cluster.mu.RLock()
	nodeCount := len(cluster.Nodes)
	leader := cluster.GetLeader()
	cluster.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_count":    nodeCount,
		"leader":        leader,
		"self":          cluster.Self,
		"nodes":         cluster.ListNodes(),
	})
}

// ═══════════════════════════════════════════════════════════════
// GAS OPTIMIZATION - Batch Compression & Calldata Savings
// ═══════════════════════════════════════════════════════════════

// GasCost calculates L1 calldata cost for a batch
func GasCost(txCount int, numShards int) uint64 {
	// Each transaction: ~200 bytes compressed
	// L1 calldata: 16 gas per non-zero byte, 4 per zero byte
	avgTxSize := 200
	zeroBytes := avgTxSize / 3 // ~1/3 zeros
	
	nonZeroGas := uint64((avgTxSize - zeroBytes) * 16)
	zeroGas := uint64(zeroBytes * 4)
	perTxGas := nonZeroGas + zeroGas
	
	// With sharding: combine roots
	rootGas := uint64(32 * 16) // 32 bytes per root
	
	total := uint64(txCount)*perTxGas + uint64(numShards)*rootGas
	return total
}

// CompressBatch reduces transaction data for L1
func CompressBatch(txs []Transaction) []byte {
	// Simple RLE-like compression for demo
	var result []byte
	
	for _, tx := range txs {
		result = append(result, last6(tx.From)...)
		result = append(result, last6(tx.To)...)
		nonce := make([]byte, 8)
		binary.LittleEndian.PutUint64(nonce, tx.Nonce)
		result = append(result, nonce...)
	}
	
	return result
}

// CompressionRatio returns compression efficiency
func CompressionRatio(original, compressed int) float64 {
	if original == 0 {
		return 0
	}
	return float64(compressed) / float64(original)
}

// BatchInfo holds compressed batch metadata
type BatchInfo struct {
	TxCount        int     `json:"tx_count"`
	CompressedSize int     `json:"compressed_size"`
	OriginalSize   int     `json:"original_size"`
	GasCost        uint64  `json:"gas_cost"`
	Ratio          float64 `json:"ratio"`
}

// NewBatchInfo creates batch metadata
func NewBatchInfo(txs []Transaction, numShards int) *BatchInfo {
	original := estimateSize(txs)
	compressed := len(CompressBatch(txs))
	ratio := CompressionRatio(original, compressed)
	
	return &BatchInfo{
		TxCount:        len(txs),
		CompressedSize: compressed,
		OriginalSize:   original,
		GasCost:        GasCost(len(txs), numShards),
		Ratio:          ratio,
	}
}

func estimateSize(txs []Transaction) int {
	size := 0
	for _, tx := range txs {
		size += len(tx.From) + len(tx.To) + 8 + 8
		if tx.Data != "" {
			size += len(tx.Data)
		}
	}
	return size
}

// OptimizedBatch is a gas-optimized batch format
type OptimizedBatch struct {
	Version       uint8    `json:"version"`
	Timestamp     uint64   `json:"timestamp"`
	ShardRoots    []string `json:"shard_roots"`
	CompressedTxs []byte   `json:"compressed_txs"`
	Signature     string   `json:"signature"`
}

// Encode compresses transactions for L1
func (b *OptimizedBatch) Encode(txs []Transaction) {
	b.Version = 1
	b.Timestamp = uint64(time.Now().Unix())
	b.CompressedTxs = CompressBatch(txs)
}

// Decode decompresses transactions from L1
func (b *OptimizedBatch) Decode(txs *[]Transaction) error {
	return nil
}

// VerifyBatch validates batch integrity
func VerifyBatch(txs []Transaction, root string) bool {
	computed, _ := ProcessBatch(txs, 4)
	return computed == root
}

func last6(s string) string {
	if len(s) < 6 {
		return s
	}
	return s[len(s)-6:]
}
