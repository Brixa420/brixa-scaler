package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/plonk"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/plonk"
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

	fmt.Println("=== PLONK BENCHMARK ===")
	
	ccs, _ := frontend.Compile(mod, plonk.NewBuilder, &SimpleCircuit{})
	pk, vk, _ := plonk.Setup(ccs)
	fmt.Println("Setup complete")

	// Benchmark
	for _, n := range []int{10, 50, 100, 500, 1000} {
		start := time.Now()
		for i := 0; i < n; i++ {
			w := &SimpleCircuit{A: i, B: 1000, C: i * 1000}
			wt, _ := frontend.NewWitness(w, mod)
			plonk.Prove(ccs, pk, wt)
		}
		ms := time.Since(start).Milliseconds()
		fmt.Printf("%4d proofs: %5dms = %6.0f/sec\n", n, ms, float64(n)/float64(ms)*1000)
	}
	_ = vk
}
