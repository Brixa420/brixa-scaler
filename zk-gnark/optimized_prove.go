package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

type SimpleCircuit struct {
	A, B, C frontend.Variable
}

func (c *SimpleCircuit) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(c.A, c.B), c.C)
	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	curve := ecc.BN254
	mod := curve.ScalarField()

	// OPTIMIZATION 1: Compile ONCE, reuse for all proofs
	fmt.Println("OPTIMIZATION 1: Compile circuit once...")
	ccs, _ := frontend.Compile(mod, r1cs.NewBuilder, &SimpleCircuit{})
	
	// OPTIMIZATION 2: Setup ONCE, reuse key
	fmt.Println("OPTIMIZATION 2: Setup once...")
	pk, vk, _ := groth16.Setup(ccs)
	_ = vk
	
	fmt.Println("Ready! Now proving...")
	fmt.Println()

	// Test throughput
	for _, proofs := range []int{100, 500, 1000, 2000, 5000} {
		var wg sync.WaitGroup
		start := time.Now()
		
		for i := 0; i < proofs; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				w := &SimpleCircuit{A: idx, B: 1000, C: idx * 1000}
				wt, _ := frontend.NewWitness(w, mod)
				groth16.Prove(ccs, pk, wt)
			}(i)
		}
		wg.Wait()
		ms := time.Since(start).Milliseconds()
		
		fmt.Printf("%5d proofs: %5dms = %7.0f TPS\n", proofs, ms, float64(proofs)*1000/float64(ms))
	}
	
	fmt.Println("\n✅ This is optimized groth16 - good baseline for comparison")
	fmt.Println("   For real recursion: need Nova/Sangria or wrap C++/Rust")
}
