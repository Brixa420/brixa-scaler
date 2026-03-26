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
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

// ═══════════════════════════════════════════════════════════════
// CONFIGURATION
// ═══════════════════════════════════════════════════════════════

type Config struct {
	MaxBatchSize    int  `env:"MAX_BATCH_SIZE"`
	BatchTimeoutMs int  `env:"BATCH_TIMEOUT_MS"`
	RPCPort         int  `env:"RPC_PORT"`
	MetricsPort     int  `env:"METRICS_PORT"`
	MetricsEnabled  bool
	DemoMode        bool
}

type SecurityConfig struct {
	APIKey             string
	RateLimitPerSecond int
	RateLimitBurst     int
	MaxRequestSize     int64
	MaxGasPriceGwei    uint64
	CORSOrigins       []string
}

func LoadConfig() Config {
	return Config{
		MaxBatchSize:    getEnvInt("MAX_BATCH_SIZE", 1000),
		BatchTimeoutMs: getEnvInt("BATCH_TIMEOUT_MS", 1000),
		RPCPort:         getEnvInt("RPC_PORT", 8080),
		MetricsPort:     getEnvInt("METRICS_PORT", 9090),
		MetricsEnabled:  getEnvBool("METRICS_ENABLED", true),
		DemoMode:        getEnvBool("DEMO_MODE", true),
	}
}

func LoadSecurityConfig() SecurityConfig {
	return SecurityConfig{
		APIKey:             os.Getenv("API_KEY"),
		RateLimitPerSecond: getEnvInt("RATE_LIMIT_PER_SECOND", 10),
		RateLimitBurst:     getEnvInt("RATE_LIMIT_BURST", 20),
		MaxRequestSize:     getEnvInt64("MAX_REQUEST_SIZE", 1024*1024),
		MaxGasPriceGwei:    uint64(getEnvInt("MAX_GAS_PRICE_GWEI", 100)),
		CORSOrigins:        strings.Split(os.Getenv("CORS_ORIGINS"), ","),
	}
}

func getEnvInt(key string, def int) int {
	var val int
	fmt.Sscanf(os.Getenv(key), "%d", &val)
	return val
}

func getEnvInt64(key string, def int64) int64 {
	var val int64
	fmt.Sscanf(os.Getenv(key), "%d", &val)
	if val == 0 {
		return def
	}
	return val
}

func getEnvBool(key string, def bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val == "true" || val == "1"
}

// ═══════════════════════════════════════════════════════════════
// LOGGING
// ═══════════════════════════════════════════════════════════════

type Logger struct{}

func (l Logger) Info(msg string, fields map[string]interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"level":         "info",
		"message":       msg,
		"timestamp":     time.Now().Format(time.RFC3339),
		"correlationId": "",
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

// ═══════════════════════════════════════════════════════════════
// CORE DATA STRUCTURES
// ═══════════════════════════════════════════════════════════════

type Transaction struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value uint64 `json:"value"`
	Nonce uint64 `json:"nonce"`
	Data  string `json:"data"`
}

type BatchRequest struct {
	Transactions []Transaction `json:"transactions"`
	ShardID     int           `json:"shard_id"`
}

type BatchResponse struct {
	Root      string `json:"root"`
	BatchID   string `json:"batch_id"`
	Timestamp int64  `json:"timestamp"`
}

type Stats struct {
	totalBatches    uint64
	totalTxs        uint64
	pendingBatches uint64
	pendingProofs  uint64
	startTime      time.Time
	lock           sync.Mutex
}

// ═══════════════════════════════════════════════════════════════
// GLOBAL STATE
// ═══════════════════════════════════════════════════════════════

var (
	logger     Logger
	stats      = Stats{startTime: time.Now()}
	config     = LoadConfig()
	secConfig  = LoadSecurityConfig()
	rateLimiter = rate.NewLimiter(rate.Limit(secConfig.RateLimitPerSecond), secConfig.RateLimitBurst)
	nonceTracker = NewNonceTracker()
)

// ═══════════════════════════════════════════════════════════════
// MERKLE TREE
// ═══════════════════════════════════════════════════════════════

func buildMerkleRoot(hashes [][]byte, numShards int) []byte {
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

// ═══════════════════════════════════════════════════════════════
// BATCH PROCESSING
// ═══════════════════════════════════════════════════════════════

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

// ═══════════════════════════════════════════════════════════════
// HTTP HANDLERS
// ═══════════════════════════════════════════════════════════════

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
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
		"root":       root,
		"elapsed_ms": elapsed,
		"tps":        tps,
	})
}

func handleBatch(w http.ResponseWriter, r *http.Request) {
	var req BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Transactions) == 0 {
		http.Error(w, "No transactions in batch", http.StatusBadRequest)
		return
	}

	if len(req.Transactions) > config.MaxBatchSize {
		http.Error(w, fmt.Sprintf("Batch too large (max %d)", config.MaxBatchSize), http.StatusBadRequest)
		return
	}

	root, elapsed := ProcessBatch(req.Transactions, 4)

	logger.Info("batch processed", map[string]interface{}{
		"tx_count":   len(req.Transactions),
		"elapsed_ms": elapsed,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BatchResponse{
		Root:      root,
		BatchID:   fmt.Sprintf("batch_%d", time.Now().UnixNano()),
		Timestamp: time.Now().Unix(),
	})
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	stats.lock.Lock()
	uptime := int64(time.Since(stats.startTime).Seconds())
	stats.lock.Unlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP brixascaler_total_batches Total batches processed\n")
	fmt.Fprintf(w, "# TYPE brixascaler_total_batches counter\n")
	fmt.Fprintf(w, "brixascaler_total_batches %d\n", stats.totalBatches)
	fmt.Fprintf(w, "# HELP brixascaler_total_txs Total transactions processed\n")
	fmt.Fprintf(w, "# TYPE brixascaler_total_txs counter\n")
	fmt.Fprintf(w, "brixascaler_total_txs %d\n", stats.totalTxs)
	fmt.Fprintf(w, "# HELP brixascaler_uptime_seconds Server uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE brixascaler_uptime_seconds gauge\n")
	fmt.Fprintf(w, "brixascaler_uptime_seconds %d\n", uptime)
}

// ═══════════════════════════════════════════════════════════════
// NONCE TRACKING
// ═══════════════════════════════════════════════════════════════

type NonceTracker struct {
	nonces map[string]uint64
	mu     sync.Mutex
}

func NewNonceTracker() *NonceTracker {
	return &NonceTracker{
		nonces: make(map[string]uint64),
	}
}

func (nt *NonceTracker) GetNonce(address string) uint64 {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	return nt.nonces[address]
}

func (nt *NonceTracker) SetNonce(address string, nonce uint64) {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.nonces[address] = nonce
}

func (nt *NonceTracker) IncrementNonce(address string) uint64 {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.nonces[address]++
	return nt.nonces[address]
}

// ═══════════════════════════════════════════════════════════════
// SERVER SETUP WITH SECURITY MIDDLEWARE
// ═══════════════════════════════════════════════════════════════

func SetupServer() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/batch", handleBatch)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/benchmark", handleBenchmark)
	if config.MetricsEnabled {
		mux.HandleFunc("/metrics", handleMetrics)
	}
	return mux
}

// Secure middleware chain
func secureMiddleware(next http.Handler) http.Handler {
	// Rate limiting
	handler := rateLimitMiddleware(next)
	// Request size limit
	handler = requestSizeMiddleware(secConfig.MaxRequestSize)(handler)
	// CORS
	handler = corsMiddleware(secConfig.CORSOrigins)(handler)
	// API Key (if configured)
	if secConfig.APIKey != "" {
		handler = apiKeyMiddleware(secConfig.APIKey)(handler)
	}
	return handler
}

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rateLimiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestSizeMiddleware(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxSize {
				http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			next.ServeHTTP(w, r)
		})
	}
}

func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(origins) > 0 {
				origin := r.Header.Get("Origin")
				for _, allowed := range origins {
					if origin == allowed || allowed == "*" {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
						w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
						break
					}
				}
			}
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func apiKeyMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := r.Header.Get("X-API-Key")
			if provided == "" {
				provided = r.URL.Query().Get("api_key")
			}
			if provided != apiKey {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ═══════════════════════════════════════════════════════════════
// SERVER CONTROL
// ═══════════════════════════════════════════════════════════════

func PrintRoutes() {
	fmt.Println("  GET  /batch    - Submit transaction batch (POST JSON)")
	fmt.Println("  GET  /health   - Server health & stats")
	fmt.Println("  GET  /benchmark - Quick TPS benchmark")
	fmt.Println("  GET  /metrics  - Prometheus metrics")
}

func StartServer(port int) error {
	mux := SetupServer()
	handler := secureMiddleware(mux)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
}

func RunMain() {
	logger.Info("brixa-scaler starting", nil)

	if config.DemoMode {
		logger.Warn("DEMO_MODE enabled - no real transactions", map[string]interface{}{
			"warning": "DEMO_MODE is enabled - no real transactions",
		})
		fmt.Printf("\n⚠️  WARNING: DEMO_MODE is enabled!\n")
		fmt.Printf("   No real transactions will be processed.\n")
		fmt.Printf("   Set DEMO_MODE=false to enable real transactions\n\n")
	}

	PrintRoutes()
	fmt.Printf("🚀 BrixaScaler running on http://localhost:%d\n", config.RPCPort)

	if config.MetricsEnabled {
		go func() {
			logger.Info("metrics server started", nil)
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.MetricsPort), http.HandlerFunc(handleMetrics)))
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}

func main() {
	RunMain()
}
