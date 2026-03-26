package main

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"sync"
	"time"
)

func HashSHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

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

func ProcessChunk(start, end int, result chan []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	
	hashes := make([][]byte, end-start)
	for i := start; i < end; i++ {
		data := []byte(fmt.Sprintf("tx%d", i))
		hashes[i-start] = HashSHA256(data)
	}
	
	root := MerkleRoot(hashes)
	result <- root
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	sizes := []int{10000000, 20000000, 50000000}
	numWorkers := runtime.NumCPU()
	
	fmt.Printf("CPUs: %d
", numWorkers)
	
	fmt.Println("| Size       | Time     | TPS        |")
	fmt.Println("|------------|----------|------------|")
	
	for _, size := range sizes {
		chunkSize := size / numWorkers
		if chunkSize < 1000 {
			chunkSize = 1000
		}
		
		start := time.Now()
		
		var wg sync.WaitGroup
		result := make(chan []byte, numWorkers)
		
		for w := 0; w < numWorkers; w++ {
			chunkStart := w * chunkSize
			chunkEnd := chunkStart + chunkSize
			if chunkEnd > size {
				chunkEnd = size
			}
			if chunkStart >= size {
				break
			}
			
			wg.Add(1)
			go ProcessChunk(chunkStart, chunkEnd, result, &wg)
		}
		
		wg.Wait()
		close(result)
		
		var roots [][]byte
		for r := range result {
			roots = append(roots, r)
		}
		
		_ = MerkleRoot(roots)
		elapsed := time.Since(start)
		
		tps := float64(size) / elapsed.Seconds()
		
		fmt.Printf("| %d | %7s | %10.0f |
", size, elapsed.Round(time.Millisecond), tps)
	}
}
