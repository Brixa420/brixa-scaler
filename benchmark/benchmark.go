// Package benchmark provides TPS benchmarks for the BrixaScaler batching layer.
// Run with: go run benchmark.go
//
// Hardware: Apple M4 (10-core) - Your results may vary based on CPU
package main

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// HashSHA256 performs a single SHA256 hash
func HashSHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// HashOnlyParallel hashes transactions in parallel using all CPU cores
// This is the "batching layer" - fast ingestion, no ZK, no settlement
func HashOnlyParallel(batchSize int) int64 {
	numWorkers := runtime.NumCPU()
	if numWorkers > batchSize {
		numWorkers = batchSize
	}

	var wg sync.WaitGroup
	result := make(chan []byte, numWorkers)

	baseSize := batchSize / numWorkers
	remainder := batchSize % numWorkers

	current := 0
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		end := current + baseSize
		if w < remainder {
			end++
		}
		if current >= batchSize {
			break
		}
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				data := []byte(fmt.Sprintf("tx%d", i))
				result <- HashSHA256(data)
			}
		}(current, end)
		current = end
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	count := int64(0)
	for range result {
		count++
	}

	return count
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     BrixaScaler Batching Layer Benchmark (Hash Only)          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("\nCPUs: %d\n\n", runtime.NumCPU())

	fmt.Println("| Size       | Time     | TPS        |")
	fmt.Println("|------------|----------|------------|")

	for _, size := range []int{100000, 1000000, 10000000} {
		start := time.Now()
		HashOnlyParallel(size)
		elapsed := time.Since(start)
		tps := float64(size) / elapsed.Seconds()
		fmt.Printf("| %d | %7s | %10.0f |\n", size, elapsed.Round(time.Millisecond), tps)
	}

	fmt.Println("\nNote: This is the BATCHING LAYER (hash only).")
	fmt.Println("ZK settlement layer adds merkle tree + proof generation (~17k TPS).")
	fmt.Println("See WHITEPAPER.md for architecture explanation.")
}