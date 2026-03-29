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

	r1, _ := frontend.Compile(mod, r1cs.NewBuilder, &SimpleCircuit{})
	pk, _, _ := groth16.Setup(r1)

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           PARALLEL GNARK - High Load                          ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	proofs := 5000
	for _, provers := range []int{64, 128, 256, 512} {
		var wg sync.WaitGroup
		start := time.Now()
		
		for i := 0; i < provers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < proofs/provers; j++ {
					w := &SimpleCircuit{A: j, B: 1000, C: j * 1000}
					wt, _ := frontend.NewWitness(w, mod)
					groth16.Prove(r1, pk, wt)
				}
			}()
		}
		wg.Wait()
		ms := time.Since(start).Milliseconds()
		fmt.Printf("%3d provers: %6dms = %7.0f TPS\n", provers, ms, float64(proofs)/float64(ms)*1000)
	}
}
