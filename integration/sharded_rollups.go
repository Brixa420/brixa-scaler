package main

import (
	"sync"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	BATCHING_TPS = 5000000
	ZK_PROVE_MS  = 385
	SETTLE_MS    = 12000
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           SHARDED ROLLUPS - Parallel Settlement                   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
	
	configs := []struct{ shards, provers int }{
		{1, 100}, {10, 100}, {50, 500}, {100, 1000}, {500, 2000}, {1000, 5000},
	}
	
	fmt.Println("\n┌──────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ FULL PIPELINE: Batching(5M) → ZK → Sharded Settlement              │")
	fmt.Println("├────────────┬────────────┬────────────┬────────────┬────────────────┤")
	fmt.Printf("│ Shards     │ Provers    │ ZK TPS     │ Settle TPS │ Effective TPS  │\n")
	fmt.Println("├────────────┼────────────┼────────────┼────────────┼────────────────┤")
	
	for _, c := range configs {
		zkTPS := float64(c.provers) * (1000.0 / ZK_PROVE_MS)
		settleTPS := float64(c.shards) * (1000*1000) / SETTLE_MS
		effective := zkTPS
		if settleTPS < zkTPS { effective = settleTPS }
		
		bar := ""
		for i := 0; i < int(effective/1000); i++ {
			bar += "█"
		}
		
		fmt.Printf("│ %8d │ %8d │ %10.0f │ %10.0f │ %7.0f %s │\n",
			c.shards, c.provers, zkTPS, settleTPS, effective, bar)
	}
	fmt.Println("└────────────┴────────────┴────────────┴────────────┴────────────────┘")
	
	// Simple live test - prove throughput
	fmt.Println("\n┌──────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ Live: ZK Proving Only (3s)                                         │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
	
	for _, c := range configs[:3] {
		testProving(c.provers, 3*time.Second)
	}
	
	// Summary
	fmt.Println("\n╔════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      KEY INSIGHTS                                ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  • 1 shard = 83 TPS (Ethereum L2 baseline)                        ║")
	fmt.Println("║  • 10 shards = 833 TPS                                           ║")
	fmt.Println("║  • 100 shards = 8,333 TPS (80x faster!)                          ║")
	fmt.Println("║  • 1000 shards = 83,333 TPS (1000x!)                            ║")
	fmt.Println("║                                                                    ║")
	fmt.Println("║  TO REACH 100K TPS: Need ~1200 shards OR faster finality         ║")
	fmt.Println("║  Alternative: Solana/Avalanche finality (faster than 12s)       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
}

func testProving(provers int, dur time.Duration) {
	queue := make(chan struct{}, provers)
	var proven int64
	
	var wg sync.WaitGroup
	for i := 0; i < provers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case queue <- struct{}{}:
					time.Sleep(ZK_PROVE_MS * time.Millisecond)
					atomic.AddInt64(&proven, 1)
				default:
					return
				}
			}
		}()
	}
	
	time.Sleep(dur)
	wg.Wait()
	
	fmt.Printf("  %4d provers → %,.0f proofs/sec\n", provers, float64(proven)/dur.Seconds())
}
