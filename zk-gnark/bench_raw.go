package main

import (
	"fmt"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

type Circuit struct {
	Res frontend.Variable `gnark:",public"`
	A, B frontend.Variable `gnark:"secret"`
}

func (c *Circuit) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(c.A, c.B), c.Res)
	return nil
}

func main() {
	fieldMod := ecc.BN254.ScalarField()
	
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           RAW PROVING TIME (gnark groth16)                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	
	cs, _ := frontend.Compile(fieldMod, r1cs.NewBuilder, &Circuit{})
	pk, _, _ := groth16.Setup(cs)
	fmt.Printf("Circuit: %d constraints (A*B=Res)\n\n", cs.GetNbConstraints())
	
	// Fixed witness that should work
	w := &Circuit{Res: 100, A: 10, B: 10}
	wt, _ := frontend.NewWitness(w, fieldMod)
	
	fmt.Println("=== PROVING TIME ===")
	
	for _, count := range []int{10, 50, 100, 500, 1000} {
		start := time.Now()
		var lastProof interface{}
		for i := 0; i < count; i++ {
			lastProof, _ = groth16.Prove(cs, pk, wt)
		}
		ms := time.Since(start).Milliseconds()
		rate := count * 1000 / int(ms)
		fmt.Printf("%d proofs: %dms = %d proofs/sec\n", count, ms, rate)
		_ = lastProof
	}
	
	fmt.Println("\n=== SINGLE PROOF ===")
	start := time.Now()
	p, _ := groth16.Prove(cs, pk, wt)
	ms := time.Since(start).Microseconds()
	fmt.Printf("Time: %.2fms\n", float64(ms)/1000)
	
	_ = p // Use the proof
}
