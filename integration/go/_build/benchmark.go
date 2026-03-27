package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// Transaction represents a simple transaction
type Transaction struct {
	From  string
	To    string
	Value uint64
	Nonce uint64
}

// HashSHA256 computes SHA256 hash
func HashSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// MerkleRoot computes merkle root from transactions (binary tree)
func MerkleRoot(txs []Transaction) string {
	if len(txs) == 0 {
		return HashSHA256([]byte("empty"))
	}
	if len(txs) == 1 {
		return HashSHA256([]byte(txs[0].From + txs[0].To))
	}

	// Build merkle tree
	level := make([]string, len(txs))
	for i, tx := range txs {
		level[i] = HashSHA256([]byte(fmt.Sprintf("%s:%s:%d:%d", tx.From, tx.To, tx.Value, tx.Nonce)))
	}

	for len(level) > 1 {
		nextLevel := make([]string, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				combined := level[i] + level[i+1]
				nextLevel = append(nextLevel, HashSHA256([]byte(combined)))
			} else {
				// Odd one out - hash with itself
				nextLevel = append(nextLevel, HashSHA256([]byte(level[i]+level[i])))
			}
		}
		level = nextLevel
	}

	return level[0]
}

// ProcessChunk processes a chunk of transactions
func ProcessChunk(txs []Transaction, start, end int) string {
	chunk := txs[start:end]
	if len(chunk) == 0 {
		return ""
	}
	return MerkleRoot(chunk)
}

// ProcessBatch processes transactions in parallel
func ProcessBatch(txs []Transaction, numWorkers int) string {
	if len(txs) == 0 {
		return ""
	}

	if len(txs) < 1000 || numWorkers <= 1 {
		return MerkleRoot(txs)
	}

	chunkSize := len(txs) / numWorkers
	if chunkSize < 100 {
		chunkSize = 100
	}

	var wg sync.WaitGroup
	results := make([]string, 0, numWorkers)

	for i := 0; i < len(txs); i += chunkSize {
		end := i + chunkSize
		if end > len(txs) {
			end = len(txs)
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			root := ProcessChunk(txs, start, end)
			if root != "" {
				results = append(results, root)
			}
		}(i, end)
	}

	wg.Wait()

	return MerkleRoot(txs)
}

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
	b.ResetTimer()
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
	b.ResetTimer()
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
	b.ResetTimer()
	for b.N > 0 {
		ProcessBatch(txs, 8)
	}
}

func BenchmarkMerkle_1000000(b *testing.B) {
	txs := make([]Transaction, 1000000)
	for i := 0; i < 1000000; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 1000000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	b.ResetTimer()
	for b.N > 0 {
		ProcessBatch(txs, 8)
	}
}

func main() {
	// Run benchmarks
	fmt.Println("=== BrixaScaler Merkle Layer Benchmark ===")
	fmt.Printf("CPUs: %d\n", runtime.NumCPU())
	fmt.Println()

	// Quick test
	txs := make([]Transaction, 1000)
	for i := 0; i < 1000; i++ {
		txs[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 1000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}

	start := time.Now()
	root := ProcessBatch(txs, 4)
	elapsed := time.Since(start)

	fmt.Printf("1000 txs: %v -> root: %s...\n", elapsed, root[:16])
	
	// 10K
	txs10k := make([]Transaction, 10000)
	for i := 0; i < 10000; i++ {
		txs10k[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 10000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	start = time.Now()
	root = ProcessBatch(txs10k, 4)
	elapsed = time.Since(start)
	fmt.Printf("10K txs: %v -> root: %s...\n", elapsed, root[:16])
	
	// 100K
	txs100k := make([]Transaction, 100000)
	for i := 0; i < 100000; i++ {
		txs100k[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 100000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	start = time.Now()
	root = ProcessBatch(txs100k, 8)
	elapsed = time.Since(start)
	fmt.Printf("100K txs: %v -> root: %s...\n", elapsed, root[:16])
	
	// 1M
	txs1m := make([]Transaction, 1000000)
	for i := 0; i < 1000000; i++ {
		txs1m[i] = Transaction{
			From:  fmt.Sprintf("0x%x", i),
			To:    fmt.Sprintf("0x%x", 1000000-i),
			Value: uint64(i),
			Nonce: uint64(i),
		}
	}
	start = time.Now()
	root = ProcessBatch(txs1m, 8)
	elapsed = time.Since(start)
	
	tps := float64(1000000) / elapsed.Seconds()
	fmt.Printf("1M txs: %v -> root: %s...\n", elapsed, root[:16])
	fmt.Printf("TPS: %.0f\n", tps)
}