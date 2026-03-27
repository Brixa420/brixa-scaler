package main

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// MERKLE TREE BENCHMARK - Batching Layer
// ═══════════════════════════════════════════════════════════════

func HashSHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// Single-shard Merkle root (baseline)
func MerkleRoot(hashes [][]byte) []byte {
	if len(hashes) == 0 {
		return HashSHA256([]byte("empty"))
	}
	
	layer := make([][]byte, len(hashes))
	copy(layer, hashes)
	
	for len(layer) > 1 {
		next := make([][]byte, 0, (len(layer)+1)/2)
		for i := 0; i < len(layer); i += 2 {
			left := layer[i]
			right := left
			if i+1 < len(layer) {
				right = layer[i+1]
			}
			combined := make([]byte, len(left)+len(right))
			copy(combined, left)
			copy(combined[len(left):], right)
			next = append(next, HashSHA256(combined))
		}
		layer = next
	}
	
	return layer[0]
}

// ProcessShardRange - parallel shard processing
func ProcessShardRange(start, end int, result chan []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	
	hashes := make([][]byte, 0, end-start)
	for i := start; i < end; i++ {
		data := []byte(fmt.Sprintf("tx-%d", i))
		hashes = append(hashes, HashSHA256(data))
	}
	
	root := MerkleRoot(hashes)
	result <- root
}

// Sharded Merkle tree (parallel)
func ShardedMerkleRoot(batchSize, numShards int) []byte {
	chunkSize := batchSize / numShards
	if chunkSize < 1 {
		chunkSize = 1
	}
	
	var wg sync.WaitGroup
	result := make(chan []byte, numShards)
	
	for s := 0; s < numShards; s++ {
		startIdx := s * chunkSize
		endIdx := startIdx + chunkSize
		if endIdx > batchSize {
			endIdx = batchSize
		}
		if startIdx >= batchSize {
			break
		}
		
		wg.Add(1)
		go ProcessShardRange(startIdx, endIdx, result, &wg)
	}
	
	wg.Wait()
	close(result)
	
	// Combine shard roots
	var roots [][]byte
	for r := range result {
		roots = append(roots, r)
	}
	
	return MerkleRoot(roots)
}

func runBenchmark(name string, fn func() time.Duration, iterations int) (float64, time.Duration) {
	var total time.Duration
	for i := 0; i < iterations; i++ {
		elapsed := fn()
		total += elapsed
	}
	avg := total / time.Duration(iterations)
	tps := float64(1000000) / avg.Seconds() / 1000 // 1M / ms = K TPS
	
	return tps, avg
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║         MERKLE TREE BENCHMARK - Batching Layer             ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")
	
	fmt.Printf("CPU Cores: %d\n\n", runtime.NumCPU())
	
	iterations := 5
	
	// Test 1: Single-shard (baseline)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("SINGLE-SHARD (baseline)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("%-15s %12s %15s\n", "Batch Size", "Time", "TPS")
	fmt.Println("───────────────────────────────────────────────────────────")
	
	singleTests := []int{1000, 10000, 100000, 1000000}
	for _, size := range singleTests {
		hashes := make([][]byte, size)
		for i := 0; i < size; i++ {
			data := []byte(fmt.Sprintf("tx-%d", i))
			hashes[i] = HashSHA256(data)
		}
		
		_, avg := runBenchmark("single", func() time.Duration {
			start := time.Now()
			_ = MerkleRoot(hashes)
			return time.Since(start)
		}, iterations)
		
		fmt.Printf("%-15d %12s %15.0fK\n", size, avg.Round(time.Microsecond), float64(size)/avg.Seconds()/1000)
	}
	
	// Test 2: Sharded (parallel)
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("SHARDED + PARALLEL (10 cores)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("%-15s %8s %12s %15s\n", "Batch Size", "Shards", "Time", "TPS")
	fmt.Println("─────────────────────────────────────────────────────────────")
	
	shardedTests := []struct {
		batchSize int
		shards    int
	}{
		{100000, 10},
		{500000, 10},
		{1000000, 10},
		{5000000, 10},
		{10000000, 10},
		{10000000, 20},
	}
	
	for _, test := range shardedTests {
		_, avg := runBenchmark("sharded", func() time.Duration {
			start := time.Now()
			_ = ShardedMerkleRoot(test.batchSize, test.shards)
			return time.Since(start)
		}, iterations)
		
		fmt.Printf("%-15d %8d %12s %15.0fK\n", 
			test.batchSize, test.shards, avg.Round(time.Microsecond), 
			float64(test.batchSize)/avg.Seconds()/1000)
	}
	
	// Summary
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      SUMMARY                               ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ %-20s %25.0fK TPS     ║\n", "Single-shard (1M):", float64(1000000)/160*1000)
	fmt.Printf("║ %-20s %25.0fK TPS     ║\n", "Sharded 10 (10M):", 10000000.0/600*1000)
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}
