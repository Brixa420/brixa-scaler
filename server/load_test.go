package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

type Transaction struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Value uint64 `json:"value"`
}

type SubmitRequest struct {
	Transactions []Transaction `json:"transactions"`
}

func main() {
	fmt.Println(`
╔══════════════════════════════════════════════════════════════════════╗
║                    BRIXA SCALER LOAD TEST                              ║
╚══════════════════════════════════════════════════════════════════════╝
`)
	
	baseURL := "http://localhost:8080"
	concurrency := 10
	txPerRequest := 100
	duration := 10 * time.Second
	
	fmt.Printf("Config: %d concurrent workers, %d tx/request, %s duration\n",
		concurrency, txPerRequest, duration)
	fmt.Printf("Target: %s\n\n", baseURL)
	
	var (
		txSent     int64
		txSuccess  int64
		txFailed   int64
	)
	
	// Health check
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		fmt.Printf("❌ Server not running at %s\n", baseURL)
		fmt.Println("   Run: cd server && go run rpc_server.go")
		return
	}
	resp.Body.Close()
	fmt.Printf("✅ Server healthy at %s\n", baseURL)
	
	// Start load test
	start := time.Now()
	stop := make(chan struct{})
	
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			reqID := workerID * 1000000
			for {
				select {
				case <-stop:
					return
				default:
					txs := make([]Transaction, txPerRequest)
					for j := 0; j < txPerRequest; j++ {
						txs[j] = Transaction{
							ID:    fmt.Sprintf("tx_%d_%d", workerID, reqID+j),
							From:  fmt.Sprintf("0x%x", workerID),
							To:    fmt.Sprintf("0x%x", (workerID+1)%concurrency),
							Value: uint64(reqID + j),
						}
					}
					
					reqBody, _ := json.Marshal(SubmitRequest{Transactions: txs})
					resp, err := http.Post(baseURL+"/submit", "application/json", bytes.NewBuffer(reqBody))
					atomic.AddInt64(&txSent, int64(txPerRequest))
					if err == nil {
						resp.Body.Close()
						atomic.AddInt64(&txSuccess, int64(txPerRequest))
					} else {
						atomic.AddInt64(&txFailed, int64(txPerRequest))
					}
				}
			}
		}(i)
	}
	
	// Progress ticker
	go func() {
		for {
			time.Sleep(2 * time.Second)
			elapsed := time.Since(start)
			sent := atomic.LoadInt64(&txSent)
			success := atomic.LoadInt64(&txSuccess)
			fmt.Printf("\r⏱ %s | Sent: %d | Success: %d | TPS: %.0f", 
				elapsed.Round(time.Second), sent, success, float64(success)/elapsed.Seconds())
		}
	}()
	
	time.Sleep(duration)
	close(stop)
	time.Sleep(time.Second)
	
	elapsed := time.Since(start)
	total := atomic.LoadInt64(&txSent)
	success := atomic.LoadInt64(&txSuccess)
	failed := atomic.LoadInt64(&txFailed)
	
	fmt.Printf("\n\n╔══════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                         RESULTS                                     ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║  Duration:    %s                                                ║\n", elapsed.Round(time.Second))
	fmt.Printf("║  Total Sent:  %d                                               ║\n", total)
	fmt.Printf("║  Success:     %d                                               ║\n", success)
	fmt.Printf("║  Failed:      %d                                               ║\n", failed)
	fmt.Printf("║  TPS:         %.0f                                              ║\n", float64(success)/elapsed.Seconds())
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════╝\n")
}
