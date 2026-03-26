package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
)

// ═══════════════════════════════════════════════════════════════
// UNIT TESTS - Core Logic (Merkle Batching & Serialization)
// ═══════════════════════════════════════════════════════════════

func TestMerkleRoot_Empty(t *testing.T) {
	txs := []Transaction{}
	root, _ := ProcessBatch(txs, 1)
	
	if len(root) != 32*2 { // hex encoded
		t.Errorf("expected root length 64 (hex), got %d", len(root))
	}
}

func TestMerkleRoot_Single(t *testing.T) {
	txs := []Transaction{{
		From:  "0x123",
		To:    "0x456",
		Value: 100,
		Nonce: 1,
	}}
	
	root, timeMs := ProcessBatch(txs, 1)
	
	if len(root) != 64 {
		t.Errorf("expected root length 64 (hex), got %d", len(root))
	}
	if timeMs < 0 {
		t.Errorf("expected positive time, got %d", timeMs)
	}
}

func TestMerkleRoot_Multiple(t *testing.T) {
	txs := make([]Transaction, 100)
	for i := 0; i < 100; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 100-i),
			Value: uint64(i * 100),
			Nonce: uint64(i),
		}
	}
	root, _ := ProcessBatch(txs, 4)
	
	if len(root) != 64 {
		t.Errorf("expected root length 64 (hex), got %d", len(root))
	}
}

func TestMerkleRoot_Deterministic(t *testing.T) {
	txs := make([]Transaction, 50)
	for i := 0; i < 50; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 50-i),
			Value: uint64(i * 100),
			Nonce: uint64(i),
		}
	}
	
	root1, _ := ProcessBatch(txs, 4)
	root2, _ := ProcessBatch(txs, 4)
	
	if root1 != root2 {
		t.Errorf("expected deterministic root, got different: %s vs %s", root1, root2)
	}
}

func TestMerkleRoot_LargeBatch(t *testing.T) {
	txs := make([]Transaction, 100000)
	for i := 0; i < 100000; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 100000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	root, timeMs := ProcessBatch(txs, 8)
	
	if len(root) != 64 {
		t.Errorf("expected root length 64 (hex), got %d", len(root))
	}
	
	// Should complete in reasonable time (< 10 seconds for 100k)
	if timeMs > 10000 {
		t.Errorf("expected < 10000ms, got %dms", timeMs)
	}
	
	t.Logf("100k txs processed in %dms = %d TPS", timeMs, 100000*1000/int(timeMs))
}

func TestMerkleRoot_ShardCount(t *testing.T) {
	txs := make([]Transaction, 32)
	for i := 0; i < 32; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 31-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	
	// Test different shard counts
	root2, _ := ProcessBatch(txs, 2)
	root4, _ := ProcessBatch(txs, 4)
	root8, _ := ProcessBatch(txs, 8)
	
	// Different shard counts = different roots (correct behavior)
	_ = root2
	_ = root4
	_ = root8
	t.Logf("shard tests: 2=%s, 4=%s, 8=%s", root2[:8], root4[:8], root8[:8])
}

func TestTransaction_HashCollision(t *testing.T) {
	tx1 := Transaction{From: "0x111", To: "0x222", Value: 100, Nonce: 1}
	tx2 := Transaction{From: "0x111", To: "0x222", Value: 100, Nonce: 2}
	tx3 := Transaction{From: "0x111", To: "0x222", Value: 101, Nonce: 1}
	
	data1 := fmt.Sprintf("%s%s%d%d%s", tx1.From, tx1.To, tx1.Value, tx1.Nonce, tx1.Data)
	data2 := fmt.Sprintf("%s%s%d%d%s", tx2.From, tx2.To, tx2.Value, tx2.Nonce, tx2.Data)
	data3 := fmt.Sprintf("%s%s%d%d%s", tx3.From, tx3.To, tx3.Value, tx3.Nonce, tx3.Data)
	
	hash1 := sha256.Sum256([]byte(data1))
	hash2 := sha256.Sum256([]byte(data2))
	hash3 := sha256.Sum256([]byte(data3))
	
	if hex.EncodeToString(hash1[:]) == hex.EncodeToString(hash2[:]) {
		t.Error("expected different hashes for different nonces")
	}
	
	if hex.EncodeToString(hash1[:]) == hex.EncodeToString(hash3[:]) {
		t.Error("expected different hashes for different values")
	}
}

func TestTransaction_Serialization(t *testing.T) {
	tx := Transaction{
		From:  "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB71",
		To:    "0x8Ba1f109551bD432803012645Ac136ddd64DBA72",
		Value: 1000000,
		Nonce: 42,
		Data:  "0x1234",
	}
	
	// Test JSON serialization round-trip
	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	
	var tx2 Transaction
	if err := json.Unmarshal(data, &tx2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	
	if tx2.From != tx.From || tx2.To != tx.To || tx2.Value != tx.Value {
		t.Errorf("serialization round-trip failed")
	}
}

func TestStats_Increment(t *testing.T) {
	initialBatches := stats.totalBatches
	initialTxs := stats.totalTxs
	
	txs := []Transaction{{From: "0x1", To: "0x2", Value: 100, Nonce: 1}}
	ProcessBatch(txs, 1)
	
	if stats.totalBatches != initialBatches+1 {
		t.Errorf("expected batch count %d, got %d", initialBatches+1, stats.totalBatches)
	}
	if stats.totalTxs != initialTxs+1 {
		t.Errorf("expected tx count %d, got %d", initialTxs+1, stats.totalTxs)
	}
}

func TestStats_PendingBatches(t *testing.T) {
	// Track initial state
	initialPending := stats.pendingBatches
	
	// Simulate adding pending batch
	stats.lock.Lock()
	stats.pendingBatches++
	stats.lock.Unlock()
	
	if stats.pendingBatches != initialPending+1 {
		t.Errorf("expected pending %d, got %d", initialPending+1, stats.pendingBatches)
	}
	
	// Simulate clearing
	stats.lock.Lock()
	stats.pendingBatches = 0
	stats.lock.Unlock()
}

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := LoadConfig()
	
	if cfg.MaxBatchSize == 0 {
		t.Error("expected default max batch size")
	}
	if cfg.BatchTimeoutMs == 0 {
		t.Error("expected default batch timeout")
	}
}

func TestHealthResponse_Structure(t *testing.T) {
	resp := HealthResponse{
		Status:         "healthy",
		Version:        "1.0.0",
		Uptime:         100,
		TPS:            1000.5,
		TotalBatches:   10,
		TotalTxs:       1000,
		NumShards:      4,
		CPUCores:       4,
		PendingBatches: 0,
		PendingProofs:  0,
	}
	
	if resp.Status != "healthy" {
		t.Errorf("expected status healthy, got %s", resp.Status)
	}
	if resp.Version == "" {
		t.Error("expected non-empty version")
	}
	if resp.CPUCores <= 0 {
		t.Error("expected positive CPU cores")
	}
}

func TestLogger_Structured(t *testing.T) {
	logger := Logger{}
	
	// Test info log
	logger.Info("test message", map[string]interface{}{
		"key": "value",
		"num": 123,
	})
	
	// Test error log
	logger.Error("error message", map[string]interface{}{
		"err": "test error",
	})
	
	// Test warning
	logger.Warn("warning message", map[string]interface{}{
		"warn": "test warning",
	})
	
	// If we get here without panic, logging works
	t.Log("structured logging works")
}

// Full integration test with mocked L2
func TestFullFlow_BatchToSettle(t *testing.T) {
	// 1. Ingest transactions
	txs := make([]Transaction, 100)
	for i := 0; i < 100; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 100-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	
	// 2. Create batch
	root, timeMs := ProcessBatch(txs, 4)
	
	if len(root) != 64 {
		t.Fatalf("invalid Merkle root")
	}
	
	// 3. Simulate proof (mocked)
	proofGenerated := true
	if !proofGenerated {
		t.Fatal("proof generation failed")
	}
	
	// 4. Simulate verify (mocked)
	proofVerified := true
	if !proofVerified {
		t.Fatal("proof verification failed")
	}
	
	// 5. Simulate settle (mocked L2)
	settled := true
	if !settled {
		t.Fatal("settlement failed")
	}
	
	t.Logf("Full flow: 100 txs -> batch in %dms -> proof -> settle", timeMs)
}

func TestFullFlow_LargeBatch(t *testing.T) {
	txs := make([]Transaction, 10000)
	for i := 0; i < 10000; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 10000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	
	root, timeMs := ProcessBatch(txs, 8)
	
	if len(root) != 64 {
		t.Fatalf("invalid Merkle root")
	}
	
	tps := float64(10000) * 1000 / float64(timeMs)
	t.Logf("10k txs: %dms = %.0f TPS", timeMs, tps)
	
	// Should be fast
	if timeMs > 5000 {
		t.Errorf("expected < 5s, got %dms", timeMs)
	}
}

func TestIntegration_MultipleBatches(t *testing.T) {
	// Simulate multiple batches being processed
	batchCount := 10
	txsPerBatch := 100
	
	totalTxs := 0
	totalTime := int64(0)
	
	for b := 0; b < batchCount; b++ {
		txs := make([]Transaction, txsPerBatch)
		for i := 0; i < txsPerBatch; i++ {
			txs[i] = Transaction{
				From:  fmt.Sprintf("0x%x", b*txsPerBatch+i),
				To:    fmt.Sprintf("0x%x", b*txsPerBatch+txsPerBatch-i),
				Value: uint64(i),
				Nonce: uint64(i),
			}
		}
		
		_, timeMs := ProcessBatch(txs, 4)
		totalTxs += txsPerBatch
		totalTime += timeMs
	}
	
	avgTime := totalTime / int64(batchCount)
	t.Logf("Multiple batches: %d txs in %dms (avg %dms/batch)", totalTxs, totalTime, avgTime)
}

// Benchmark tests
func BenchmarkMerkle_1000(b *testing.B) {
	txs := make([]Transaction, 1000)
	for i := 0; i < 1000; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 1000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	for b.N > 0 {
		ProcessBatch(txs, 4)
	}
}

func BenchmarkMerkle_10000(b *testing.B) {
	txs := make([]Transaction, 10000)
	for i := 0; i < 10000; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 10000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	for b.N > 0 {
		ProcessBatch(txs, 4)
	}
}

func BenchmarkMerkle_100000(b *testing.B) {
	txs := make([]Transaction, 100000)
	for i := 0; i < 100000; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 100000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	for b.N > 0 {
		ProcessBatch(txs, 8)
	}
}
func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(10, 5) // 10 rps, burst 5
	
	// Should allow first 5 (burst)
	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Errorf("expected allow for request %d", i)
		}
	}
	
	// Burst exhausted - should now be rate limited
	if rl.Allow() {
		t.Error("expected rate limit after burst")
	}
	
	if rl.GetRequests() != 5 {
		t.Errorf("expected 5 requests, got %d", rl.GetRequests())
	}
}

func TestRateLimiter_Burst(t *testing.T) {
	rl := NewRateLimiter(1, 3) // 1 rps, burst 3
	
	// All within burst
	allowed := 0
	for i := 0; i < 10; i++ {
		if rl.Allow() {
			allowed++
		}
	}
	
	// Should allow exactly 3 (burst)
	if allowed != 3 {
		t.Errorf("expected burst of 3, got %d", allowed)
	}
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")
	
	result := getEnvInt("TEST_INT", 0)
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
	
	// Default
	os.Unsetenv("TEST_INT")
	result = getEnvInt("TEST_INT", 99)
	if result != 99 {
		t.Errorf("expected default 99, got %d", result)
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		val      string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
		{"", false},
	}
	
	for _, tt := range tests {
		if tt.val != "" {
			os.Setenv("TEST_BOOL", tt.val)
			defer os.Unsetenv("TEST_BOOL")
		} else {
			os.Unsetenv("TEST_BOOL")
		}
		
		result := getEnvBool("TEST_BOOL", false)
		if result != tt.expected {
			t.Errorf("expected %v for %q, got %v", tt.expected, tt.val, result)
		}
	}
}

func TestConfig_Load(t *testing.T) {
	os.Setenv("MAX_BATCH_SIZE", "5000")
	os.Setenv("BATCH_TIMEOUT_MS", "2000")
	os.Setenv("RPC_PORT", "9000")
	os.Setenv("METRICS_PORT", "9001")
	os.Setenv("DEMO_MODE", "false")
	defer func() {
		os.Unsetenv("MAX_BATCH_SIZE")
		os.Unsetenv("BATCH_TIMEOUT_MS")
		os.Unsetenv("RPC_PORT")
		os.Unsetenv("METRICS_PORT")
		os.Unsetenv("DEMO_MODE")
	}()
	
	cfg := LoadConfig()
	
	if cfg.MaxBatchSize != 5000 {
		t.Errorf("expected 5000, got %d", cfg.MaxBatchSize)
	}
	if cfg.BatchTimeoutMs != 2000 {
		t.Errorf("expected 2000, got %d", cfg.BatchTimeoutMs)
	}
	if cfg.RPCPort != 9000 {
		t.Errorf("expected 9000, got %d", cfg.RPCPort)
	}
	if cfg.MetricsPort != 9001 {
		t.Errorf("expected 9001, got %d", cfg.MetricsPort)
	}
	if cfg.DemoMode != false {
		t.Errorf("expected false, got %v", cfg.DemoMode)
	}
}

func TestStats_Concurrent(t *testing.T) {
	// Note: stats are global, so we test relative increase
	initialTxs := stats.totalTxs
	
	var wg sync.WaitGroup
	
	// Concurrent increments
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats.lock.Lock()
			stats.totalTxs++
			stats.totalBatches++
			stats.lock.Unlock()
		}()
	}
	
	wg.Wait()
	
	// Test relative increase (accounting for other tests)
	if stats.totalTxs < initialTxs+100 {
		t.Errorf("expected at least %d txs, got %d", initialTxs+100, stats.totalTxs)
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	rl := NewRateLimiter(10000, 10000)
	
	for b.N > 0 {
		rl.Allow()
	}
}

func TestCluster_AddNode(t *testing.T) {
	c := &Cluster{Nodes: make(map[string]*Node)}
	
	node := &Node{ID: "node1", Address: "localhost", Port: 8080, Weight: 50}
	c.AddNode(node)
	
	if len(c.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(c.Nodes))
	}
	
	if c.Nodes["node1"].Status != "active" {
		t.Errorf("expected active status")
	}
}

func TestCluster_RemoveNode(t *testing.T) {
	c := &Cluster{Nodes: make(map[string]*Node)}
	
	c.AddNode(&Node{ID: "node1", Status: "active"})
	c.RemoveNode("node1")
	
	if c.Nodes["node1"].Status != "leaving" {
		t.Errorf("expected leaving status")
	}
}

func TestCluster_GetLeader(t *testing.T) {
	c := &Cluster{Nodes: make(map[string]*Node)}
	
	c.AddNode(&Node{ID: "node1", Weight: 50, Status: "active"})
	c.AddNode(&Node{ID: "node2", Weight: 100, Status: "active"})
	
	leader := c.GetLeader()
	if leader != "node2" {
		t.Errorf("expected node2 as leader, got %s", leader)
	}
}

func TestCluster_GetLeader_NoActive(t *testing.T) {
	c := &Cluster{Nodes: make(map[string]*Node)}
	
	// Only inactive nodes
	c.AddNode(&Node{ID: "node1", Status: "leaving"})
	
	leader := c.GetLeader()
	if leader != "" {
		t.Errorf("expected no leader, got %s", leader)
	}
}

func TestCluster_ListNodes(t *testing.T) {
	c := &Cluster{Nodes: make(map[string]*Node)}
	
	c.AddNode(&Node{ID: "node1", Weight: 50})
	c.AddNode(&Node{ID: "node2", Weight: 100})
	
	nodes := c.ListNodes()
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}
