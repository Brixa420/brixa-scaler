package main

import (
	"fmt"
	"runtime"
	"sync"
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

	ccs, _ := frontend.Compile(mod, plonk.NewBuilder, &SimpleCircuit{})
	pk, vk, _ := plonk.Setup(ccs)

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           PLONK vs GROTH16 Benchmark                          ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	proofs := 500
	for _, provers := range []int{1, 8, 64} {
		var wg sync.WaitGroup
		start := time.Now()
		
		for i := 0; i < provers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < proofs/provers; j++ {
					w := &SimpleCircuit{A: j, B: 1000, C: j * 1000}
					wt, _ := frontend.NewWitness(w, mod)
					plonk.Prove(ccs, pk, wt)
				}
			}()
		}
		wg.Wait()
		ms := time.Since(start).Milliseconds()
		fmt.Printf("%3d provers: %5dms = %6.0f TPS\n", provers, ms, float64(proofs)/float64(ms)*1000)
	}
	_ = vk
}
