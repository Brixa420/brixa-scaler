package main

import (
	"fmt"
	"runtime"
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
	fmt.Println("║           DIRECT GNARK (No HTTP, Same Process)                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	// 100 proofs
	start := time.Now()
	for i := 0; i < 100; i++ {
		w := &SimpleCircuit{A: i, B: 1000, C: i * 1000}
		wt, _ := frontend.NewWitness(w, mod)
		groth16.Prove(r1, pk, wt)
	}
	ms := time.Since(start).Milliseconds()
	
	fmt.Printf("\n100 proofs in %dms = %.0f TPS\n", ms, float64(100)/float64(ms)*1000)
	fmt.Println("\n✅ This is the max we can get without parallelism!")
}
