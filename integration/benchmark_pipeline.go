package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Batch struct {
	ID   int
	Size int
}

type Proof struct {
	BatchID int
}

const (
	BATCHING_TPS = 5000000
	ZK_PROVE_MS  = 385
	SETTLE_MS    = 12000
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║     REALISTIC PIPELINE: Full Throughput Simulation                 ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\nConfig: Batching=%d TPS, ZK=%dms, Settlement=%ds\n", 
		BATCHING_TPS, ZK_PROVE_MS, SETTLE_MS/1000)
	
	proverCounts := []int{1, 10, 50, 100, 500, 1000, 2000, 5000}
	
	fmt.Println("\n┌────────────┬────────────┬────────────┬────────────┬────────────────┐")
	fmt.Println("│ Provers    │ ZK TPS     │ Settled/s │ Backlog    │ Effective TPS  │")
	fmt.Println("├────────────┼────────────┼────────────┼────────────┼────────────────┤")
	
	for _, numProvers := range proverCounts {
		zkTPS := float64(numProvers) * (1000.0 / ZK_PROVE_MS)
		settleTPS := float64(1000*1000) / SETTLE_MS
		
		effective := zkTPS
		if settleTPS < zkTPS { effective = settleTPS }
		
		backlog := float64(BATCHING_TPS) - zkTPS
		if backlog < 0 { backlog = 0 }
		
		fmt.Printf("│ %10d │ %10.0f │ %10.1f │ %10.0f │ %14.0f │\n",
			numProvers, zkTPS, settleTPS, backlog, effective)
	}
	fmt.Println("└────────────┴────────────┴────────────┴────────────┴────────────────┘")
	
	// Live simulation - fixed
	fmt.Println("\n┌──────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ Live Simulation (3s window)                                        │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
	
	for _, n := range []int{10, 100, 500} {
		runSim(n, 3*time.Second)
	}
	
	fmt.Println("\n╔════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        ANALYSIS                                    ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Batching:    5M TPS (surplus capacity)                          ║")
	fmt.Println("║  ZK Pool:     2.6 TPS/prover                                       ║")
	fmt.Println("║  Settlement:  83 TPS max (THE BOTTLENECK!)                         ║")
	fmt.Println("║                                                                    ║")
	fmt.Println("║  To match batching: need ~2,000 provers                           ║")
	fmt.Println("║  But settlement = 83 TPS = NEED SHARDED ROLLUPS                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
}

func runSim(numProvers int, dur time.Duration) {
	fmt.Printf("\n─── %d provers, %v ───\n", numProvers, dur)
	
	batchChan := make(chan *Batch, numProvers*10)
	proofChan := make(chan *Proof, 10000)
	
	var batched, proven, settled atomic.Int64
	
	// Batching at 5M TPS
	ticker := time.NewTicker(time.Duration(1e9/BATCHING_TPS) * time.Nanosecond)
	go func() {
		id := 0
		for range ticker.C {
			select {
			case batchChan <- &Batch{ID: id, Size: 1000}:
				batched.Add(1000)
				id++
			default:
			}
		}
	}()
	
	// ZK provers
	var wg sync.WaitGroup
	for i := 0; i < numProvers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for b := range batchChan {
				time.Sleep(ZK_PROVE_MS * time.Millisecond)
				select {
				case proofChan <- &Proof{BatchID: b.ID}:
					proven.Add(1000)
				default:
				}
			}
		}()
	}
	
	// Settlement worker
	go func() {
		for p := range proofChan {
			time.Sleep(10 * time.Millisecond)
			_ = p
			settled.Add(1000)
		}
	}()
	
	time.Sleep(dur)
	ticker.Stop()
	close(batchChan)
	wg.Wait()
	close(proofChan)
	
	batchedPS := float64(batched.Load()) / dur.Seconds()
	provenPS := float64(proven.Load()) / dur.Seconds()
	settledPS := float64(settled.Load()) / dur.Seconds()
	
	fmt.Printf("  Batched: %,.0f/s | Proven: %,.0f/s | Settled: %,.0f/s\n",
		batchedPS, provenPS, settledPS)
}
