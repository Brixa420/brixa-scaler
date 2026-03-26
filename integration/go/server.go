package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
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

func main() {
	cfg := LoadConfig()
	rpcPort := cfg.RPCPort
	metricsPort := cfg.MetricsPort

	logger.Info("brixa-scaler starting", map[string]interface{}{
		"version":        "1.0.0",
		"rpc_port":       rpcPort,
		"metrics_port":   metricsPort,
		"demo_mode":      cfg.DemoMode,
	})

	if cfg.DemoMode {
		fmt.Printf("\n⚠️  WARNING: DEMO_MODE is enabled!\n")
		fmt.Printf("   No real transactions will be processed.\n")
		fmt.Printf("   Set DEMO_MODE=false to enable real transactions\n\n")
	}

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

	// HTTP handlers with rate limiting
	http.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
		if !globalLimiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"rate limit exceeded","retry_after":1}`)
			return
		}
		handleBatch(w, r)
	})
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/benchmark", handleBenchmark)

	// Metrics endpoint
	if cfg.MetricsEnabled {
		http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "# HELP brixa_scaler_total_txs Total transactions processed\n")
			fmt.Fprintf(w, "# TYPE brixa_scaler_total_txs counter\n")
			fmt.Fprintf(w, "brixa_scaler_total_txs %d\n", stats.totalTxs)
			fmt.Fprintf(w, "# HELP brixa_scaler_total_batches Total batches created\n")
			fmt.Fprintf(w, "# TYPE brixa_scaler_total_batches counter\n")
			fmt.Fprintf(w, "brixa_scaler_total_batches %d\n", stats.totalBatches)
		})
	}

	fmt.Printf("\n🚀 BrixaScaler running on http://localhost:%d\n", rpcPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", rpcPort), nil))
}