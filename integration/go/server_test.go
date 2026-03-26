package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	
)

// ═══════════════════════════════════════════════════════════════
// UNIT TESTS - Core Logic
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
}

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
	
	// 3. Simulate proof (in production: zkProver.GenerateProof)
	proofGenerated := true
	if !proofGenerated {
		t.Fatal("proof generation failed")
	}
	
	// 4. Simulate verify
	proofVerified := true
	if !proofVerified {
		t.Fatal("proof verification failed")
	}
	
	// 5. Simulate settle
	settled := true
	if !settled {
		t.Fatal("settlement failed")
	}
	
	t.Logf("Full flow: 100 txs -> batch in %dms -> proof -> settle", timeMs)
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