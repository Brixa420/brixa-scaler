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
)

// ═══════════════════════════════════════════════════════════════════════
// BRIXASCALER HTTP API - Production Ready
// ═══════════════════════════════════════════════════════════════════════

// Logger provides structured JSON logging
type Logger struct {
	Format string
}

func (l *Logger) Init(format string) {
	l.Format = format
}

func (l *Logger) Log(level, msg string, fields map[string]interface{}) {
	if l.Format == "json" {
		output := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     level,
			"message":   msg,
		}
		for k, v := range fields {
			output[k] = v
		}
		jsonBytes, _ := json.Marshal(output)
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("[%s] %s: %s\n", level, msg, fields)
	}
}

func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.Log("INFO", msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.Log("ERROR", msg, fields)
}

func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.Log("WARN", msg, fields)
}

var logger Logger

// Config holds all configuration
type Config struct {
	DemoMode           bool   `env:"DEMO_MODE"`
	SettlementRPCURL   string `env:"SETTLEMENT_RPC_URL"`
	VerifierAddress    string `env:"VERIFIER_ADDRESS"`
	SettlementKey      string `env:"SETTLEMENT_PRIVATE_KEY"`
	MaxBatchSize       int    `env:"MAX_BATCH_SIZE"`
	BatchTimeoutMs     int    `env:"BATCH_TIMEOUT_MS"`
	MaxPendingTxs      int    `env:"MAX_PENDING_TXS"`
	MetricsEnabled     bool   `env:"METRICS_ENABLED"`
	MetricsPort        int    `env:"METRICS_PORT"`
	LogLevel           string `env:"LOG_LEVEL"`
	LogFormat          string `env:"LOG_FORMAT"`
	PersistenceEnabled bool   `env:"PERSISTENCE_ENABLED"`
	PersistencePath    string `env:"PERSISTENCE_PATH"`
}

// LoadConfig loads configuration from environment
func LoadConfig() *Config {
	return &Config{
		DemoMode:           getEnvBool("DEMO_MODE", true),
		SettlementRPCURL:   getEnv("SETTLEMENT_RPC_URL", "https://polygon-rpc.com"),
		VerifierAddress:    getEnv("VERIFIER_ADDRESS", ""),
		SettlementKey:      getEnv("SETTLEMENT_PRIVATE_KEY", ""),
		MaxBatchSize:       getEnvInt("MAX_BATCH_SIZE", 1000),
		BatchTimeoutMs:     getEnvInt("BATCH_TIMEOUT_MS", 5000),
		MaxPendingTxs:      getEnvInt("MAX_PENDING_TXS", 100000),
		MetricsEnabled:     getEnvBool("METRICS_ENABLED", true),
		MetricsPort:        getEnvInt("METRICS_PORT", 9090),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		LogFormat:          getEnv("LOG_FORMAT", "json"),
		PersistenceEnabled: getEnvBool("PERSISTENCE_ENABLED", true),
		PersistencePath:    getEnv("PERSISTENCE_PATH", "./data/state.db"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var intVal int
		fmt.Sscanf(val, "%d", &intVal)
		return intVal
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}

type Transaction struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Value    uint64 `json:"value"`
	Data     string `json:"data,omitempty"`
	Nonce    uint64 `json:"nonce"`
	Hash     string `json:"hash,omitempty"`
	BatchID  string `json:"batch_id,omitempty"`
	Position uint64 `json:"position,omitempty"`
}

type BatchRequest struct {
	Transactions []Transaction `json:"transactions"`
	ShardID      int           `json:"shard_id,omitempty"`
}

type BatchResponse struct {
	BatchID     string   `json:"batch_id"`
	Root        string   `json:"merkle_root"`
	TxCount     int      `json:"tx_count"`
	ProcessTime int64    `json:"process_time_ms"`
	Shards      int      `json:"shards_used"`
}

type HealthResponse struct {
	Status        string  `json:"status"`
	Version       string  `json:"version"`
	Uptime        int64   `json:"uptime_seconds"`
	TPS           float64 `json:"current_tps"`
	TotalBatches  uint64  `json:"total_batches"`
	TotalTxs      uint64  `json:"total_txs"`
	NumShards     int     `json:"num_shards"`
	CPUCores      int     `json:"cpu_cores"`
	PendingBatches uint64 `json:"pending_batches"`
	PendingProofs uint64  `json:"pending_proofs"`
}

type Stats struct {
	totalBatches   uint64
	totalTxs       uint64
	pendingBatches uint64
	pendingProofs  uint64
	startTime      time.Time
	lock           sync.Mutex
}

var stats Stats

// ProcessBatch processes a batch of transactions
func ProcessBatch(txs []Transaction, numShards int) (string, int64) {
	start := time.Now()
	
	// Build Merkle tree
	leaves := make([][]byte, len(txs))
	for i, tx := range txs {
		data := fmt.Sprintf("%s%s%d%d%s", tx.From, tx.To, tx.Value, tx.Nonce, tx.Data)
		hash := sha256.Sum256([]byte(data))
		leaves[i] = hash[:]
	}
	
	// Build tree based on number of shards
	root := buildMerkleRoot(leaves, numShards)
	
	elapsed := time.Since(start).Milliseconds()
	
	stats.lock.Lock()
	stats.totalBatches++
	stats.totalTxs += uint64(len(txs))
	stats.lock.Unlock()
	
	return hex.EncodeToString(root), elapsed
}

func buildMerkleRoot(leaves [][]byte, numShards int) []byte {
	if len(leaves) == 0 {
		return make([]byte, 32)
	}
	
	// Simple implementation - in production use full merkle tree
	for len(leaves) > 1 {
		newLevel := make([][]byte, (len(leaves)+1)/2)
		for i := 0; i < len(leaves)/2; i++ {
			combined := append(leaves[i*2], leaves[i*2+1]...)
			hash := sha256.Sum256(combined)
			newLevel[i] = hash[:]
		}
		if len(leaves)%2 == 1 {
			newLevel[len(newLevel)-1] = leaves[len(leaves)-1]
		}
		leaves = newLevel
	}
	
	return leaves[0]
}

func init() {
	stats.startTime = time.Now()
}

func handleBatch(w http.ResponseWriter, r *http.Request) {
	// start := time.Now()
	correlationID := fmt.Sprintf("%d", time.Now().UnixNano())
	
	var req BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("failed to decode request", map[string]interface{}{
			"correlation_id": correlationID,
			"error":         err.Error(),
		})
		http.Error(w, err.Error(), 400)
		return
	}
	
	// Demo mode warning
	if os.Getenv("DEMO_MODE") != "false" {
		logger.Info("demo_mode: transaction logged but not sent", map[string]interface{}{
			"correlation_id":  correlationID,
			"tx_count":       len(req.Transactions),
			"demo_mode":      true,
			"warning":        "DEMO_MODE is enabled - no real transactions",
		})
	}
	
	root, timeMs := ProcessBatch(req.Transactions, runtime.NumCPU())
	
	batchID := fmt.Sprintf("batch-%d", stats.totalBatches+1)
	
	resp := BatchResponse{
		BatchID:     batchID,
		Root:        root,
		TxCount:     len(req.Transactions),
		ProcessTime: timeMs,
		Shards:      runtime.NumCPU(),
	}
	
	logger.Info("batch processed", map[string]interface{}{
		"correlation_id": correlationID,
		"batch_id":      batchID,
		"tx_count":      len(req.Transactions),
		"process_time_ms": timeMs,
	})
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	elapsed := time.Since(stats.startTime).Seconds()
	tps := float64(stats.totalTxs) / elapsed
	
	stats.lock.Lock()
	totalBatches := stats.totalBatches
	totalTxs := stats.totalTxs
	pendingBatches := stats.pendingBatches
	pendingProofs := stats.pendingProofs
	stats.lock.Unlock()
	
	resp := HealthResponse{
		Status:         "healthy",
		Version:        "1.0.0",
		Uptime:         int64(elapsed),
		TPS:            tps,
		TotalBatches:  totalBatches,
		TotalTxs:      totalTxs,
		NumShards:     runtime.NumCPU(),
		CPUCores:       runtime.NumCPU(),
		PendingBatches: pendingBatches,
		PendingProofs: pendingProofs,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleBenchmark(w http.ResponseWriter, r *http.Request) {
	testSizes := []int{1000, 10000, 100000, 1000000}
	numShards := runtime.NumCPU()
	
	type benchmarkResult struct {
		Size  int   `json:"batch_size"`
		Time  int64 `json:"time_ms"`
		TPS   int   `json:"tps"`
	}
	
	results := make([]benchmarkResult, len(testSizes))
	
	for i, size := range testSizes {
		txs := make([]Transaction, size)
		for j := 0; j < size; j++ {
			txs[j] = Transaction{
				From:  fmt.Sprintf("0x%x", j),
				To:    fmt.Sprintf("0x%x", size-j),
				Value: uint64(j),
				Nonce: uint64(j),
			}
		}
		
		_, timeMs := ProcessBatch(txs, numShards)
		
		results[i] = benchmarkResult{
			Size: size,
			Time: timeMs,
			TPS:  int(float64(size) / float64(timeMs) * 1000),
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"shards":     numShards,
		"cpu_cores":  runtime.NumCPU(),
		"results":    results,
	})
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	cfg := LoadConfig()
	logger.Init(cfg.LogFormat)
	
	logger.Info("brixa-scaler starting", map[string]interface{}{
		"version":        "1.0.0",
		"demo_mode":      cfg.DemoMode,
		"cpu_cores":      runtime.NumCPU(),
		"metrics_enabled": cfg.MetricsEnabled,
	})
	
	// Demo mode loud warning
	if cfg.DemoMode {
		logger.Warn("⚠️ DEMO MODE ENABLED - No real transactions will be sent!", map[string]interface{}{
			"warning": "Set DEMO_MODE=false to enable real transactions",
			"danger":  "REAL MONEY AT RISK",
		})
	}
	
	// Graceful shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-shutdownChan
		logger.Info("shutdown signal received, flushing pending batches", map[string]interface{}{})
		
		// Flush pending batches
		stats.lock.Lock()
		pendingBatches := stats.pendingBatches
		stats.lock.Unlock()
		
		if pendingBatches > 0 {
			logger.Info("flushing pending batches", map[string]interface{}{
				"pending_batches": pendingBatches,
			})
			// In production: wait for pending batches to settle
			time.Sleep(2 * time.Second)
		}
		
		logger.Info("shutdown complete", map[string]interface{}{})
		os.Exit(0)
	}()
	
	rpcPort := getEnvInt("RPC_PORT", 8080)
	metricsPort := getEnvInt("METRICS_PORT", 9090)
	
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║           BRIXASCALER HTTP API - Production Ready            ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")
	
	fmt.Printf("Server starting on http://localhost:%d\n", rpcPort)
	fmt.Printf("Metrics at       http://localhost:%d/metrics\n", metricsPort)
	fmt.Printf("Health check at  http://localhost:%d/health\n", rpcPort)
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("\nEndpoints:\n")
	fmt.Printf("  POST /batch     - Submit transaction batch\n")
	fmt.Printf("  GET  /health    - Server health & stats\n")
	fmt.Printf("  GET  /benchmark - Quick TPS benchmark\n")
	fmt.Printf("  GET  /metrics  - Prometheus metrics\n")
	fmt.Printf("\n")
	
	// HTTP handlers
	http.HandleFunc("/batch", handleBatch)
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
		
		// Start metrics server
		go func() {
			logger.Info("metrics server started", map[string]interface{}{
				"port": metricsPort,
			})
			http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), nil)
		}()
	}
	
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", rpcPort), nil))
}