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
	MaxBatchSize    int                 `env:"MAX_BATCH_SIZE"`
	BatchTimeoutMs int                 `env:"BATCH_TIMEOUT_MS"`
	RPCPort         int                 `env:"RPC_PORT"`
	MetricsPort     int                 `env:"METRICS_PORT"`
	MetricsEnabled  bool                `env:"METRICS_ENABLED"`
	DemoMode        bool                `env:"DEMO_MODE"`
	Persistence     PersistenceConfig   `yaml:"persistence"`
}

type SecurityConfig struct {
	APIKey             string
	RateLimitPerSecond int
	RateLimitBurst     int
	MaxRequestSize     int64
	MaxGasPriceGwei    uint64
	CORSOrigins        []string
}

func LoadConfig() Config {
	return Config{
		MaxBatchSize:    getEnvInt("MAX_BATCH_SIZE", 1000),
		BatchTimeoutMs:  getEnvInt("BATCH_TIMEOUT_MS", 1000),
		RPCPort:         getEnvInt("RPC_PORT", 8080),
		MetricsPort:     getEnvInt("METRICS_PORT", 9090),
		MetricsEnabled:  getEnvBool("METRICS_ENABLED", true),
		DemoMode:        getEnvBool("DEMO_MODE", true),
		Persistence:     PersistenceConfig{
			Enabled:     getEnvBool("PERSISTENCE_ENABLED", true),
			Backend:     getEnv("PERSISTENCE_BACKEND", "leveldb"),
			Path:        getEnv("PERSISTENCE_PATH", "./data/state.db"),
			SyncWrites:  getEnvBool("PERSISTENCE_SYNC_WRITES", false),
			Compression: getEnvBool("PERSISTENCE_COMPRESSION", true),
		},
	}
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
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

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
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
	ShardID      int           `json:"shard_id"`
}

type BatchResponse struct {
	Root      string `json:"root"`
	BatchID   string `json:"batch_id"`
	Timestamp int64  `json:"timestamp"`
}

type Stats struct {
	totalBatches   uint64
	totalTxs       uint64
	pendingBatches uint64
	pendingProofs  uint64
	startTime      time.Time
	lock           sync.Mutex
}

// ═══════════════════════════════════════════════════════════════
// GLOBAL STATE
// ═══════════════════════════════════════════════════════════════

var (
	logger       Logger
	stats        = Stats{startTime: time.Now()}
	config       Config
	secConfig    SecurityConfig
	rateLimiter  *rate.Limiter
	nonceTracker = NewNonceTracker()
	stateStore   *StateStore
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

	batchID := fmt.Sprintf("batch_%d", time.Now().UnixNano())
	root, elapsed := ProcessBatch(req.Transactions, 4)

	// Persist to state store if enabled
	if stateStore != nil && stateStore.IsEnabled() {
		// Store each transaction
		for i, tx := range req.Transactions {
			// Generate hash from tx data
			txData := fmt.Sprintf("%s%s%d%s", tx.From, tx.To, tx.Value, tx.Data)
			txHash := fmt.Sprintf("%x", sha256.Sum256([]byte(txData)))
			record := &TxRecord{
				Hash:      txHash,
				BatchID:   batchID,
				Position:  uint64(i),
				Timestamp: time.Now().Unix(),
				Status:    "batched",
			}
			if err := stateStore.PutTx(record); err != nil {
				logger.Error("failed to store tx", map[string]interface{}{"error": err.Error()})
			}
		}

		// Store batch (store empty if no txs)
		txHashes := make([]string, len(req.Transactions))
		for i, tx := range req.Transactions {
			txData := fmt.Sprintf("%s%s%d%s", tx.From, tx.To, tx.Value, tx.Data)
			txHashes[i] = fmt.Sprintf("%x", sha256.Sum256([]byte(txData)))
		}
		batch := &Batch{
			BatchID:    batchID,
			TxHashes:   txHashes,
			MerkleRoot: root,
			Timestamp:  time.Now().Unix(),
			Settled:    false,
		}
		if err := stateStore.PutBatch(batch); err != nil {
			logger.Error("failed to store batch", map[string]interface{}{"error": err.Error()})
		}
	}

	logger.Info("batch processed", map[string]interface{}{
		"tx_count":   len(req.Transactions),
		"elapsed_ms": elapsed,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BatchResponse{
		Root:      root,
		BatchID:   batchID,
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

// handleStateStats returns state store statistics
func handleStateStats(w http.ResponseWriter, r *http.Request) {
	if stateStore == nil || !stateStore.IsEnabled() {
		json.NewEncoder(w).Encode(map[string]string{"error": "state store not enabled"})
		return
	}

	stats, err := stateStore.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleGetBatch retrieves a batch by ID
func handleGetBatch(w http.ResponseWriter, r *http.Request) {
	if stateStore == nil || !stateStore.IsEnabled() {
		json.NewEncoder(w).Encode(map[string]string{"error": "state store not enabled"})
		return
	}

	batchID := r.URL.Query().Get("id")
	if batchID == "" {
		http.Error(w, "missing batch id", http.StatusBadRequest)
		return
	}

	batch, err := stateStore.GetBatch(batchID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(batch)
}

// handleGetTx retrieves a transaction by hash
func handleGetTx(w http.ResponseWriter, r *http.Request) {
	if stateStore == nil || !stateStore.IsEnabled() {
		json.NewEncoder(w).Encode(map[string]string{"error": "state store not enabled"})
		return
	}

	txHash := r.URL.Query().Get("hash")
	if txHash == "" {
		http.Error(w, "missing tx hash", http.StatusBadRequest)
		return
	}

	tx, err := stateStore.GetTx(txHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tx)
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
	mux.HandleFunc("/audit", handleAuditLog)
	mux.HandleFunc("/settlement", HandleSettlement)
	// State store endpoints
	mux.HandleFunc("/state/stats", handleStateStats)
	mux.HandleFunc("/state/batch", handleGetBatch)
	mux.HandleFunc("/state/tx", handleGetTx)
	return mux
}

// Secure middleware chain
func secureMiddleware(next http.Handler) http.Handler {
	// Security headers first
	handler := securityHeaders(next)
	// HTTPS redirect
	handler = httpsRedirectMiddleware(handler)
	// Rate limiting
	handler = rateLimitMiddleware(handler)
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
		// Bypass rate limiting if RATE_LIMIT_BURST is 0
		if secConfig.RateLimitBurst > 0 && !rateLimiter.Allow() {
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
	fmt.Println("  GET  /batch       - Submit transaction batch (POST JSON)")
	fmt.Println("  GET  /health      - Server health & stats")
	fmt.Println("  GET  /benchmark   - Quick TPS benchmark")
	fmt.Println("  GET  /metrics     - Prometheus metrics")
	fmt.Println("  GET  /state/stats - State store statistics")
	fmt.Println("  GET  /state/batch - Get batch by ID (?id=...)")
	fmt.Println("  GET  /state/tx    - Get transaction by hash (?hash=...)")
}

func StartServer(port int) error {
	mux := SetupServer()
	handler := secureMiddleware(mux)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
}

func RunMain() {
	logger.Info("brixa-scaler starting", nil)

	// Initialize config (must be done at runtime to read env vars)
	config = LoadConfig()
	secConfig = LoadSecurityConfig()
	rateLimiter = rate.NewLimiter(rate.Limit(secConfig.RateLimitPerSecond), secConfig.RateLimitBurst)

	// Initialize state store
	var err error
	stateStore, err = NewStateStore(&config.Persistence)
	if err != nil {
		logger.Error("failed to initialize state store", map[string]interface{}{"error": err.Error()})
	} else {
		logger.Info("state store initialized", map[string]interface{}{
			"enabled": stateStore.IsEnabled(),
			"backend": config.Persistence.Backend,
			"path":    config.Persistence.Path,
		})
	}

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

	// Start the main HTTP server in a goroutine
	go func() {
		logger.Info("starting HTTP server", nil)
		if err := StartServer(config.RPCPort); err != nil {
			logger.Error("HTTP server error", map[string]interface{}{"error": err.Error()})
		}
	}()

	if config.MetricsEnabled {
		go func() {
			logger.Info("metrics server started", nil)
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.MetricsPort), http.HandlerFunc(handleMetrics)))
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	logger.Info("shutting down...", nil)
	if stateStore != nil {
		if err := stateStore.Close(); err != nil {
			logger.Error("failed to close state store", map[string]interface{}{"error": err.Error()})
		}
	}
}

func main() {
	RunMain()
}

// ═══════════════════════════════════════════════════════════════
// PRIVATE KEY SECURITY
// ═══════════════════════════════════════════════════════════════

// LoadPrivateKey loads and validates private key from environment
func LoadPrivateKey() (string, error) {
	key := os.Getenv("SETTLEMENT_PRIVATE_KEY")
	if key == "" {
		return "", nil // Not configured - demo mode
	}

	// Remove 0x prefix if present
	if strings.HasPrefix(key, "0x") {
		key = key[2:]
	}

	// Validate hex length (32 bytes = 64 hex chars)
	if len(key) != 64 {
		return "", fmt.Errorf("invalid private key length: %d (expected 64)", len(key))
	}

	_, err := hex.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("invalid private key format: %v", err)
	}

	return "0x" + key, nil
}

// ValidatePrivateKey validates private key format
func ValidatePrivateKey(key string) error {
	if key == "" {
		return fmt.Errorf("private key is empty")
	}

	// Remove 0x prefix if present
	if strings.HasPrefix(key, "0x") {
		key = key[2:]
	}

	// Validate hex length (32 bytes = 64 hex chars)
	if len(key) != 64 {
		return fmt.Errorf("invalid private key length: %d (expected 64)", len(key))
	}

	_, err := hex.DecodeString(key)
	if err != nil {
		return fmt.Errorf("invalid private key format: %v", err)
	}

	return nil
}

// ValidateAddress validates Ethereum address format
func ValidateAddress(addr string) error {
	if addr == "" {
		return fmt.Errorf("address is empty")
	}

	if strings.HasPrefix(addr, "0x") {
		addr = addr[2:]
	}

	if len(addr) != 40 {
		return fmt.Errorf("invalid address length: %d (expected 40)", len(addr))
	}

	_, err := hex.DecodeString(addr)
	if err != nil {
		return fmt.Errorf("invalid address format: %v", err)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════
// TRANSACTION VALIDATION
// ═══════════════════════════════════════════════════════════════

// ValidateTransaction validates a transaction before processing
func ValidateTransaction(tx Transaction) error {
	if err := ValidateAddress(tx.From); err != nil {
		return fmt.Errorf("invalid 'from' address: %v", err)
	}
	if err := ValidateAddress(tx.To); err != nil {
		return fmt.Errorf("invalid 'to' address: %v", err)
	}
	if tx.Value == 0 && tx.Data == "" {
		return fmt.Errorf("transaction has no value and no data")
	}
	return nil
}

// ValidateBatch validates a batch of transactions
func ValidateBatch(txs []Transaction) error {
	for i, tx := range txs {
		if err := ValidateTransaction(tx); err != nil {
			return fmt.Errorf("transaction %d: %v", i, err)
		}
	}
	return nil
}

// CheckGasPrice checks if gas price is within limits
func CheckGasPrice(gasPrice uint64, maxGwei uint64) error {
	gasPriceGwei := gasPrice / 1_000_000_000
	if gasPriceGwei > maxGwei {
		return fmt.Errorf("gas price %d Gwei exceeds max %d Gwei", gasPriceGwei, maxGwei)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════
// AUDIT LOGGING
// ═══════════════════════════════════════════════════════════════

type AuditEntry struct {
	Timestamp string      `json:"timestamp"`
	Action    string      `json:"action"`
	ClientIP  string      `json:"client_ip"`
	UserAgent string      `json:"user_agent"`
	APIKey    string      `json:"api_key_hash"`
	Success   bool        `json:"success"`
	Error     string      `json:"error,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}

var auditLog []AuditEntry
var auditLogMu sync.Mutex

func logAudit(action string, r *http.Request, success bool, err error, details interface{}) {
	auditLogMu.Lock()
	defer auditLogMu.Unlock()

	entry := AuditEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    action,
		ClientIP:  r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Success:   success,
		Details:   details,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Hash API key if present
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		hash := sha256.Sum256([]byte(apiKey))
		entry.APIKey = fmt.Sprintf("%x", hash[:8])
	}

	auditLog = append(auditLog, entry)

	// Keep only last 1000 entries
	if len(auditLog) > 1000 {
		auditLog = auditLog[len(auditLog)-1000:]
	}

	logger.Info("audit: "+action, map[string]interface{}{
		"success": success,
		"client":  r.RemoteAddr,
	})
}

func handleAuditLog(w http.ResponseWriter, r *http.Request) {
	auditLogMu.Lock()
	defer auditLogMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auditLog)
}

// ═══════════════════════════════════════════════════════════════
// ADDITIONAL SECURITY MIDDLEWARE
// ═══════════════════════════════════════════════════════════════

// SanitizeInput sanitizes user input to prevent injection
func SanitizeInput(s string) string {
	// Remove null bytes and control characters
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r > 31 && r != 127 {
			result = append(result, r)
		}
	}
	return string(result)
}

// SanitizeTransaction sanitizes transaction data
func SanitizeTransaction(tx Transaction) Transaction {
	tx.From = SanitizeInput(tx.From)
	tx.To = SanitizeInput(tx.To)
	tx.Data = SanitizeInput(tx.Data)
	return tx
}

// HTTPSRedirect middleware redirects HTTP to HTTPS
func httpsRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			// HTTP request - check if HTTPS redirect enabled
			if os.Getenv("REDIRECT_HTTP_TO_HTTPS") == "true" {
				httpsURL := "https://" + r.Host + r.URL.Path
				if r.URL.RawQuery != "" {
					httpsURL += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// NoSniff middleware prevents content type sniffing
func noSniffMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders adds security headers to all responses
func securityHeaders(next http.Handler) http.Handler {
	return noSniffMiddleware(next)
}

// ═══════════════════════════════════════════════════════════════
// SETTLEMENT SAFETY
// ═══════════════════════════════════════════════════════════════

// CircuitBreaker tracks settlement failures
type CircuitBreaker struct {
	failures    int
	maxFailures int
	opened      bool
	lastFailure time.Time
	resetAfter  time.Duration
	mu          sync.Mutex
}

func NewCircuitBreaker(maxFailures int, resetAfter time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		resetAfter:  resetAfter,
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.opened = false
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.maxFailures {
		cb.opened = true
	}
}

func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.opened {
		// Check if reset time has passed
		if time.Since(cb.lastFailure) > cb.resetAfter {
			cb.opened = false
			cb.failures = 0
			return false
		}
		return true
	}
	return false
}

// Global circuit breaker for settlement
var settlementBreaker = NewCircuitBreaker(5, 5*time.Minute)

// ═══════════════════════════════════════════════════════════════
// INPUT VALIDATION ENHANCEMENTS
// ═══════════════════════════════════════════════════════════════

// MaxValues holds maximum values for validation
type MaxValues struct {
	MaxValue     uint64 `env:"MAX_TX_VALUE"`
	MaxDataSize  int    `env:"MAX_TX_DATA_SIZE"`
	MaxBatchSize int    `env:"MAX_BATCH_SIZE"`
}

func LoadMaxValues() MaxValues {
	return MaxValues{
		MaxValue:     getEnvUint64("MAX_TX_VALUE", 1_000_000_000_000_000_000), // 1000 ETH
		MaxDataSize:  getEnvInt("MAX_TX_DATA_SIZE", 1024),
		MaxBatchSize: getEnvInt("MAX_BATCH_SIZE", 1000),
	}
}

func getEnvUint64(key string, def uint64) uint64 {
	var val uint64
	fmt.Sscanf(os.Getenv(key), "%d", &val)
	if val == 0 {
		return def
	}
	return val
}

// CheckTransactionValue checks if transaction value is within limits
func CheckTransactionValue(value uint64, maxValue uint64) error {
	if value > maxValue {
		return fmt.Errorf("transaction value %d exceeds max %d", value, maxValue)
	}
	return nil
}

// CheckTransactionDataSize checks data size
func CheckTransactionDataSize(data string, maxSize int) error {
	if len(data) > maxSize {
		return fmt.Errorf("transaction data size %d exceeds max %d", len(data), maxSize)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════
// ZK PROOF VERIFICATION
// ═══════════════════════════════════════════════════════════════

// ZKProof represents a zero-knowledge proof
type ZKProof struct {
	PublicInputs []string `json:"public_inputs"`
	ProofData    string   `json:"proof_data"`
	Timestamp    int64    `json:"timestamp"`
}

// VerifyZKProof verifies a ZK proof (placeholder - implement actual verification)
func VerifyZKProof(proof ZKProof) error {
	if proof.ProofData == "" {
		return fmt.Errorf("empty proof data")
	}
	if len(proof.PublicInputs) == 0 {
		return fmt.Errorf("no public inputs")
	}

	// Check proof isn't too old (max 1 hour)
	if time.Now().Unix()-proof.Timestamp > 3600 {
		return fmt.Errorf("proof expired")
	}

	// TODO: Implement actual groth16/plonk verification
	// This would use gnark or similar library

	return nil
}

// ValidatePublicInputs validates public inputs match expected constraints
func ValidatePublicInputs(publicInputs []string, expectedRoot string, expectedCount int) error {
	if len(publicInputs) < 2 {
		return fmt.Errorf("insufficient public inputs")
	}

	// First input should be Merkle root
	if publicInputs[0] != expectedRoot {
		return fmt.Errorf("root mismatch: expected %s, got %s", expectedRoot, publicInputs[0])
	}

	// Second input should be transaction count
	// (simplified - actual implementation would verify count matches batch)

	return nil
}

// ProofVerifier holds verification state
type ProofVerifier struct {
	timeout time.Duration
	metrics *ProofMetrics
	mu      sync.Mutex
}

type ProofMetrics struct {
	Verified  uint64
	Failed    uint64
	Expired   uint64
	AvgTimeUs uint64
}

func NewProofVerifier(timeout time.Duration) *ProofVerifier {
	return &ProofVerifier{
		timeout: timeout,
		metrics: &ProofMetrics{},
	}
}

func (pv *ProofVerifier) VerifyWithTimeout(proof ZKProof) error {
	done := make(chan error, 1)

	go func() {
		done <- VerifyZKProof(proof)
	}()

	select {
	case err := <-done:
		pv.mu.Lock()
		if err != nil {
			pv.metrics.Failed++
		} else {
			pv.metrics.Verified++
		}
		pv.mu.Unlock()
		return err
	case <-time.After(pv.timeout):
		pv.mu.Lock()
		pv.metrics.Expired++
		pv.mu.Unlock()
		return fmt.Errorf("proof verification timeout after %v", pv.timeout)
	}
}

// Global proof verifier with 30 second timeout
var proofVerifier = NewProofVerifier(30 * time.Second)

// ═══════════════════════════════════════════════════════════════
// TRANSACTION SIMULATION & CONFIRMATION MONITORING
// ═══════════════════════════════════════════════════════════════

// SettlementConfig holds settlement chain configuration
type SettlementConfig struct {
	RPCURL      string
	PrivateKey  string
	ChainID     uint64
	GasLimit    uint64
	MaxGasPrice uint64
}

// LoadSettlementConfig loads settlement configuration
func LoadSettlementConfig() SettlementConfig {
	return SettlementConfig{
		RPCURL:      os.Getenv("SETTLEMENT_RPC_URL"),
		PrivateKey:  os.Getenv("SETTLEMENT_PRIVATE_KEY"),
		ChainID:     uint64(getEnvInt("SETTLEMENT_CHAIN_ID", 1)),
		GasLimit:    uint64(getEnvInt("SETTLEMENT_GAS_LIMIT", 21000)),
		MaxGasPrice: uint64(getEnvInt("MAX_GAS_PRICE_GWEI", 100)) * 1_000_000_000,
	}
}

// SimulatedTransaction represents a simulated transaction result
type SimulatedTransaction struct {
	Success      bool   `json:"success"`
	GasUsed      uint64 `json:"gas_used"`
	RevertReason string `json:"revert_reason,omitempty"`
}

// SimulateTransaction simulates a transaction before broadcast
// In production, this would call eth_call on the RPC
func SimulateTransaction(tx Transaction, config SettlementConfig) (SimulatedTransaction, error) {
	// TODO: Implement actual RPC call to eth_call

	// Placeholder simulation
	if tx.Value > 0 && tx.To == "0x0000000000000000000000000000000000000000" {
		return SimulatedTransaction{
			Success:      false,
			RevertReason: "cannot send to zero address",
		}, nil
	}

	// Basic validation
	if err := ValidateTransaction(tx); err != nil {
		return SimulatedTransaction{
			Success:      false,
			RevertReason: err.Error(),
		}, nil
	}

	return SimulatedTransaction{
		Success: true,
		GasUsed: config.GasLimit,
	}, nil
}

// ConfirmationStatus tracks transaction confirmation
type ConfirmationStatus struct {
	TxHash      string    `json:"tx_hash"`
	Status      string    `json:"status"` // "pending", "confirmed", "failed"
	BlockNumber uint64    `json:"block_number"`
	ConfirmedAt time.Time `json:"confirmed_at"`
	Retries     int       `json:"retries"`
}

// TransactionMonitor monitors blockchain for confirmations
type TransactionMonitor struct {
	confirmations map[string]*ConfirmationStatus
	mu            sync.RWMutex
	rpcURL        string
	timeout       time.Duration
	maxRetries    int
}

func NewTransactionMonitor(rpcURL string) *TransactionMonitor {
	return &TransactionMonitor{
		confirmations: make(map[string]*ConfirmationStatus),
		rpcURL:        rpcURL,
		timeout:       5 * time.Minute,
		maxRetries:    3,
	}
}

// WaitForConfirmation waits for transaction to be confirmed
func (tm *TransactionMonitor) WaitForConfirmation(txHash string, requiredConfirmations uint64) (*ConfirmationStatus, error) {
	if tm.rpcURL == "" {
		return nil, fmt.Errorf("RPC URL not configured")
	}

	// TODO: Implement actual eth_getTransactionReceipt polling
	// This would poll eth_getTransactionReceipt until confirmed

	status := &ConfirmationStatus{
		TxHash:      txHash,
		Status:      "confirmed",
		BlockNumber: 12345678,
		ConfirmedAt: time.Now(),
	}

	tm.mu.Lock()
	tm.confirmations[txHash] = status
	tm.mu.Unlock()

	return status, nil
}

// BroadcastTransaction broadcasts a transaction to the blockchain
func BroadcastTransaction(tx Transaction, config SettlementConfig) (string, error) {
	if config.RPCURL == "" || config.PrivateKey == "" {
		return "", fmt.Errorf("settlement not configured")
	}

	// Check gas price
	if config.MaxGasPrice > 0 {
		// TODO: Get current gas price and verify
	}

	// Sign and broadcast
	// TODO: Implement actual signing and broadcast via RPC

	return "0x" + "simulated_hash_" + fmt.Sprintf("%d", time.Now().UnixNano()), nil
}

// ═══════════════════════════════════════════════════════════════
// MULTI-SIG FOR HIGH VALUE TRANSACTIONS
// ═══════════════════════════════════════════════════════════════

// MultiSigConfig holds multi-signature configuration
type MultiSigConfig struct {
	Enabled            bool     `env:"MULTISIG_ENABLED"`
	Threshold          int      `env:"MULTISIG_THRESHOLD"`            // Required approvals
	Approvers          []string `env:"MULTISIG_APPROVERS"`            // Comma-separated addresses
	HighValueThreshold uint64   `env:"MULTISIG_HIGH_VALUE_THRESHOLD"` // Value requiring multi-sig
}

// LoadMultiSigConfig loads multi-sig configuration
func LoadMultiSigConfig() MultiSigConfig {
	approvers := splitCSV(os.Getenv("MULTISIG_APPROVERS"))
	threshold := getEnvInt("MULTISIG_THRESHOLD", 2)
	if threshold > len(approvers) {
		threshold = len(approvers)
	}

	return MultiSigConfig{
		Enabled:            os.Getenv("MULTISIG_ENABLED") == "true",
		Threshold:          threshold,
		Approvers:          approvers,
		HighValueThreshold: getEnvUint64("MULTISIG_HIGH_VALUE_THRESHOLD", 10_000_000_000_000_000_000), // 10 ETH
	}
}

// ApprovalRequest represents a pending approval request
type ApprovalRequest struct {
	ID         string    `json:"id"`
	TxHash     string    `json:"tx_hash"`
	From       string    `json:"from"`
	Value      uint64    `json:"value"`
	Approvers  []string  `json:"approvers"`
	ApprovedBy []string  `json:"approved_by"`
	Status     string    `json:"status"` // "pending", "approved", "rejected"
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// MultiSigManager manages multi-sig approvals
type MultiSigManager struct {
	pendingApprovals map[string]*ApprovalRequest
	mu               sync.Mutex
	config           MultiSigConfig
}

func NewMultiSigManager(config MultiSigConfig) *MultiSigManager {
	return &MultiSigManager{
		pendingApprovals: make(map[string]*ApprovalRequest),
		config:           config,
	}
}

// RequiresMultiSig checks if transaction requires multi-sig approval
func (m *MultiSigManager) RequiresMultiSig(value uint64) bool {
	if !m.config.Enabled {
		return false
	}
	return value >= m.config.HighValueThreshold
}

// RequestApproval creates a new approval request
func (m *MultiSigManager) RequestApproval(txHash string, from string, value uint64) (*ApprovalRequest, error) {
	if !m.RequiresMultiSig(value) {
		return nil, nil // No multi-sig needed
	}

	req := &ApprovalRequest{
		ID:         fmt.Sprintf("approval_%d", time.Now().UnixNano()),
		TxHash:     txHash,
		From:       from,
		Value:      value,
		Approvers:  m.config.Approvers,
		ApprovedBy: []string{},
		Status:     "pending",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	m.mu.Lock()
	m.pendingApprovals[req.ID] = req
	m.mu.Unlock()

	return req, nil
}

// Approve registers an approval from an approver
func (m *MultiSigManager) Approve(requestID string, approver string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, ok := m.pendingApprovals[requestID]
	if !ok {
		return fmt.Errorf("approval request not found")
	}

	if time.Now().After(req.ExpiresAt) {
		return fmt.Errorf("approval request expired")
	}

	// Check if approver is valid
	validApprover := false
	for _, a := range m.config.Approvers {
		if strings.ToLower(a) == strings.ToLower(approver) {
			validApprover = true
			break
		}
	}
	if !validApprover {
		return fmt.Errorf("invalid approver")
	}

	// Check if already approved
	for _, a := range req.ApprovedBy {
		if strings.ToLower(a) == strings.ToLower(approver) {
			return fmt.Errorf("already approved")
		}
	}

	req.ApprovedBy = append(req.ApprovedBy, approver)

	// Check if threshold reached
	if len(req.ApprovedBy) >= m.config.Threshold {
		req.Status = "approved"
	}

	return nil
}

// IsApproved checks if request has enough approvals
func (m *MultiSigManager) IsApproved(requestID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, ok := m.pendingApprovals[requestID]
	if !ok {
		return false
	}

	return req.Status == "approved"
}

// Global multi-sig manager
var multiSigManager *MultiSigManager

// ═══════════════════════════════════════════════════════════════
// SETTLEMENT INTEGRATION
// ═══════════════════════════════════════════════════════════════

// SettlementState holds the current settlement state
type SettlementState struct {
	PendingSettlements   uint64    `json:"pending_settlements"`
	ConfirmedSettlements uint64    `json:"confirmed_settlements"`
	FailedSettlements    uint64    `json:"failed_settlements"`
	LastSettlementTime   time.Time `json:"last_settlement_time"`
	mu                   sync.Mutex
}

var settlementState = SettlementState{}

// ProcessSettlement processes a batch for settlement
func ProcessSettlement(batch BatchResponse, txs []Transaction) (string, error) {
	cfg := LoadSettlementConfig()

	// Check circuit breaker
	if settlementBreaker.IsOpen() {
		return "", fmt.Errorf("settlement circuit breaker open - too many failures")
	}

	// Check if demo mode
	if config.DemoMode || cfg.PrivateKey == "" {
		logger.Info("demo mode: skipping settlement", map[string]interface{}{
			"batch_id": batch.BatchID,
		})
		return "", nil
	}

	// Validate all transactions
	for _, tx := range txs {
		if err := ValidateTransaction(tx); err != nil {
			settlementBreaker.RecordFailure()
			return "", fmt.Errorf("transaction validation failed: %v", err)
		}

		// Check value limits
		if err := CheckTransactionValue(tx.Value, getEnvUint64("MAX_TX_VALUE", 1_000_000_000_000_000_000)); err != nil {
			settlementBreaker.RecordFailure()
			return "", err
		}

		// Check data size
		if err := CheckTransactionDataSize(tx.Data, getEnvInt("MAX_TX_DATA_SIZE", 1024)); err != nil {
			settlementBreaker.RecordFailure()
			return "", err
		}
	}

	// Check if multi-sig required
	totalValue := uint64(0)
	for _, tx := range txs {
		totalValue += tx.Value
	}

	if multiSigManager != nil && multiSigManager.RequiresMultiSig(totalValue) {
		req, err := multiSigManager.RequestApproval(batch.BatchID, "", totalValue)
		if err != nil {
			return "", err
		}
		if req != nil {
			return "", fmt.Errorf("high-value transaction requires multi-sig approval: %s", req.ID)
		}
	}

	// Simulate transaction before broadcast
	for _, tx := range txs {
		result, err := SimulateTransaction(tx, cfg)
		if err != nil {
			return "", err
		}
		if !result.Success {
			return "", fmt.Errorf("simulation failed: %s", result.RevertReason)
		}
	}

	// Broadcast
	txHash, err := BroadcastTransaction(txs[0], cfg)
	if err != nil {
		settlementBreaker.RecordFailure()
		settlementState.mu.Lock()
		settlementState.FailedSettlements++
		settlementState.mu.Unlock()
		return "", err
	}

	settlementBreaker.RecordSuccess()
	settlementState.mu.Lock()
	settlementState.PendingSettlements++
	settlementState.mu.Unlock()

	// Start confirmation monitoring
	go func() {
		monitor := NewTransactionMonitor(cfg.RPCURL)
		status, err := monitor.WaitForConfirmation(txHash, 12)
		if err != nil {
			logger.Error("confirmation monitoring failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
		if status != nil {
			settlementState.mu.Lock()
			settlementState.PendingSettlements--
			if status.Status == "confirmed" {
				settlementState.ConfirmedSettlements++
			} else {
				settlementState.FailedSettlements++
			}
			settlementState.LastSettlementTime = time.Now()
			settlementState.mu.Unlock()
		}
	}()

	return txHash, nil
}

// HandleSettlement returns settlement state
func HandleSettlement(w http.ResponseWriter, r *http.Request) {
	settlementState.mu.Lock()
	defer settlementState.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settlementState)
}

// ═══════════════════════════════════════════════════════════════
// HARDWARE WALLET SUPPORT
// ═══════════════════════════════════════════════════════════════

// WalletType represents the type of wallet
type WalletType string

const (
	WalletTypeSoftware WalletType = "software"
	WalletTypeTrezor   WalletType = "trezor"
	WalletTypeLedger   WalletType = "ledger"
)

// WalletConfig holds wallet configuration
type WalletConfig struct {
	Type         WalletType `env:"WALLET_TYPE"`            // "software", "trezor", "ledger"
	SoftwareKey  string     `env:"SETTLEMENT_PRIVATE_KEY"` // for software wallet
	HWWalletPath string     `env:"HW_WALLET_PATH"`         // e.g., "/dev/hidraw0" or IP:port
	HWChainID    uint64     `env:"HW_CHAIN_ID"`
}

// LoadWalletConfig loads wallet configuration
func LoadWalletConfig() WalletConfig {
	walletType := WalletType(os.Getenv("WALLET_TYPE"))
	if walletType == "" {
		walletType = WalletTypeSoftware
	}

	return WalletConfig{
		Type:         walletType,
		SoftwareKey:  os.Getenv("SETTLEMENT_PRIVATE_KEY"),
		HWWalletPath: os.Getenv("HW_WALLET_PATH"),
		HWChainID:    uint64(getEnvInt("HW_CHAIN_ID", 1)),
	}
}

// Signer interface for different wallet types
type Signer interface {
	SignTransaction(tx Transaction) (string, error)
	GetAddress() (string, error)
}

// SoftwareWallet implements software-based signing
type SoftwareWallet struct {
	privateKey string
	address    string
}

func NewSoftwareWallet(privateKey string) (*SoftwareWallet, error) {
	if privateKey == "" {
		return nil, fmt.Errorf("private key not provided")
	}

	// Validate key format
	if err := ValidatePrivateKey(privateKey); err != nil {
		return nil, err
	}

	// TODO: Derive address from private key
	// In production, use go-ethereum/crypto

	return &SoftwareWallet{
		privateKey: privateKey,
		address:    "0x" + "derived_address_here",
	}, nil
}

func (w *SoftwareWallet) SignTransaction(tx Transaction) (string, error) {
	// TODO: Implement actual ECDSA signing
	// In production, use go-ethereum

	return "0x" + "signed_tx_hash", nil
}

func (w *SoftwareWallet) GetAddress() (string, error) {
	return w.address, nil
}

// HardwareWallet implements hardware wallet signing (Trezor/Ledger)
type HardwareWallet struct {
	walletType WalletType
	devicePath string
	chainID    uint64
}

func NewHardwareWallet(wt WalletType, devicePath string, chainID uint64) (*HardwareWallet, error) {
	if wt != WalletTypeTrezor && wt != WalletTypeLedger {
		return nil, fmt.Errorf("unsupported wallet type: %s", wt)
	}

	return &HardwareWallet{
		walletType: wt,
		devicePath: devicePath,
		chainID:    chainID,
	}, nil
}

func (w *HardwareWallet) SignTransaction(tx Transaction) (string, error) {
	// TODO: Implement hardware wallet signing
	// - Trezor: use trezor-lib
	// - Ledger: use ledger-app-eth

	switch w.walletType {
	case WalletTypeTrezor:
		return "", fmt.Errorf("Trezor signing not implemented - use RPC")
	case WalletTypeLedger:
		return "", fmt.Errorf("Ledger signing not implemented - use RPC")
	}

	return "", fmt.Errorf("unsupported wallet type")
}

func (w *HardwareWallet) GetAddress() (string, error) {
	// TODO: Query hardware wallet for address
	return "", fmt.Errorf("not implemented")
}

// CreateSigner creates the appropriate signer based on config
func CreateSigner(cfg WalletConfig) (Signer, error) {
	switch cfg.Type {
	case WalletTypeSoftware:
		return NewSoftwareWallet(cfg.SoftwareKey)
	case WalletTypeTrezor, WalletTypeLedger:
		return NewHardwareWallet(cfg.Type, cfg.HWWalletPath, cfg.HWChainID)
	default:
		return nil, fmt.Errorf("unknown wallet type: %s", cfg.Type)
	}
}

// ═══════════════════════════════════════════════════════════════
// KEY ROTATION
// ═══════════════════════════════════════════════════════════════

// KeyRotationConfig holds key rotation settings
type KeyRotationConfig struct {
	Enabled          bool   `env:"KEY_ROTATION_ENABLED"`
	IntervalHours    int    `env:"KEY_ROTATION_INTERVAL_HOURS"`
	MinKeyVersion    int    `env:"KEY_ROTATION_MIN_KEY_VERSION"`
	NotifyWebhookURL string `env:"KEY_ROTATION_WEBHOOK_URL"` // Alert when rotation needed
}

// LoadKeyRotationConfig loads key rotation configuration
func LoadKeyRotationConfig() KeyRotationConfig {
	return KeyRotationConfig{
		Enabled:          os.Getenv("KEY_ROTATION_ENABLED") == "true",
		IntervalHours:    getEnvInt("KEY_ROTATION_INTERVAL_HOURS", 168), // 7 days default
		MinKeyVersion:    getEnvInt("KEY_ROTATION_MIN_KEY_VERSION", 1),
		NotifyWebhookURL: os.Getenv("KEY_ROTATION_WEBHOOK_URL"),
	}
}

// KeyRotationManager manages key rotation
type KeyRotationManager struct {
	config         KeyRotationConfig
	currentKey     string
	currentVersion int
	keyHistory     []KeyVersion
	lastRotatedAt  time.Time
	mu             sync.Mutex
}

// KeyVersion represents a key version in history
type KeyVersion struct {
	Version   int       `json:"version"`
	KeyHash   string    `json:"key_hash"` // Hash of the key, never the key itself
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active"`
}

func NewKeyRotationManager(cfg KeyRotationConfig, initialKey string) *KeyRotationManager {
	krm := &KeyRotationManager{
		config:         cfg,
		currentKey:     initialKey,
		currentVersion: 1,
		keyHistory:     []KeyVersion{},
		lastRotatedAt:  time.Now(),
	}

	// Record initial key version (only hash, not the key)
	if initialKey != "" {
		hash := sha256.Sum256([]byte(initialKey))
		krm.keyHistory = append(krm.keyHistory, KeyVersion{
			Version:   1,
			KeyHash:   fmt.Sprintf("%x", hash[:8]),
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Duration(cfg.IntervalHours) * time.Hour),
			Active:    true,
		})
	}

	return krm
}

// NeedsRotation checks if key rotation is needed
func (krm *KeyRotationManager) NeedsRotation() bool {
	if !krm.config.Enabled {
		return false
	}

	elapsed := time.Since(krm.lastRotatedAt)
	return elapsed >= time.Duration(krm.config.IntervalHours)*time.Hour
}

// RotateKey rotates to a new key
func (krm *KeyRotationManager) RotateKey(newKey string) error {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	// Validate new key
	if err := ValidatePrivateKey(newKey); err != nil {
		return fmt.Errorf("invalid key format: %v", err)
	}

	// Old key becomes inactive
	if len(krm.keyHistory) > 0 {
		krm.keyHistory[len(krm.keyHistory)-1].Active = false
	}

	// Add new key
	krm.currentVersion++
	hash := sha256.Sum256([]byte(newKey))
	krm.keyHistory = append(krm.keyHistory, KeyVersion{
		Version:   krm.currentVersion,
		KeyHash:   fmt.Sprintf("%x", hash[:8]),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(krm.config.IntervalHours) * time.Hour),
		Active:    true,
	})

	krm.currentKey = newKey
	krm.lastRotatedAt = time.Now()

	// Notify via webhook if configured
	if krm.config.NotifyWebhookURL != "" {
		go func() {
			// Fire and forget - don't block
			http.Get(krm.config.NotifyWebhookURL + "?version=" + fmt.Sprintf("%d", krm.currentVersion))
		}()
	}

	logger.Warn("key rotated", map[string]interface{}{
		"old_version": krm.currentVersion - 1,
		"new_version": krm.currentVersion,
	})

	return nil
}

// GetCurrentKeyVersion returns current key version
func (krm *KeyRotationManager) GetCurrentKeyVersion() int {
	krm.mu.Lock()
	defer krm.mu.Unlock()
	return krm.currentVersion
}

// GetKeyHistory returns key version history
func (krm *KeyRotationManager) GetKeyHistory() []KeyVersion {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	result := make([]KeyVersion, len(krm.keyHistory))
	copy(result, krm.keyHistory)
	return result
}

// StartKeyRotationMonitor starts background key rotation monitoring
func StartKeyRotationMonitor(manager *KeyRotationManager) {
	if !manager.config.Enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			if manager.NeedsRotation() {
				logger.Warn("key rotation recommended", map[string]interface{}{
					"hours_since_rotation": int(time.Since(manager.lastRotatedAt).Hours()),
				})
			}
		}
	}()
}

// Global key rotation manager
var keyRotationManager *KeyRotationManager
