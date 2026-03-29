package main

import "fmt"

func main() {
    baseConstraints := 62044
    baseTimeMs := 109.0
    
    fmt.Printf("%6s | %10s | %10s | %s\n", "Txs", "Constraints", "Time/ms", "TPS")
    fmt.Printf("%6s-+-%10s-+-%10s-+-%s\n", "------", "----------", "----------", "-----")
    
    for _, n := range []int{32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072} {
        // Constraints scale ~n * 1938
        constraints := n * 1938
        timeMs := baseTimeMs * float64(constraints) / float64(baseConstraints)
        tps := float64(n) / (timeMs / 1000)
        fmt.Printf("%6d | %10d | %10.0f | %.0f\n", n, constraints, timeMs, tps)
    }
}
