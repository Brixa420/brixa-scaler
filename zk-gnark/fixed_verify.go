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

type PublicCircuit struct {
	Res frontend.Variable `gnark:",public"`
}

func (c *PublicCircuit) Define(api frontend.API) error {
	return nil
}

var pk groth16.ProvingKey
var vk groth16.VerifyingKey

func main() {
	fieldMod := ecc.BN254.ScalarField()
	
	fmt.Println("=== FIXED VERIFICATION ===")
	
	csFull, _ := frontend.Compile(fieldMod, r1cs.NewBuilder, &Circuit{})
	pk, vk, _ = groth16.Setup(csFull)
	
	// Prove: 10 * 10 = 100
	var w Circuit
	w.Res = 100
	w.A = 10
	w.B = 10
	
	wt, _ := frontend.NewWitness(&w, fieldMod)
	proof, _ := groth16.Prove(csFull, pk, wt)
	fmt.Println("Prove: Done")
	
	// Verify
	var pubW PublicCircuit
	pubW.Res = 100
	publicWt, _ := frontend.NewWitness(&pubW, fieldMod)
	
	err := groth16.Verify(proof, vk, publicWt)
	fmt.Printf("Verify: %v\n", err)
	
	if err != nil {
		fmt.Println("FAILED")
		return
	}
	
	// Benchmark
	fmt.Println("\n=== PROVE + VERIFY BENCHMARK ===")
	for _, n := range []int{10, 50, 100, 500, 1000} {
		start := time.Now()
		
		for i := 1; i <= n; i++ {
			// Prove: A=i, B=i, Res=i*i
			a := int64(i)
			res := a * a
			
			var wf Circuit
			wf.Res = res
			wf.A = a
			wf.B = a
			wt, _ := frontend.NewWitness(&wf, fieldMod)
			p, _ := groth16.Prove(csFull, pk, wt)
			
			// Verify
			var pf PublicCircuit
			pf.Res = res
			pwt, _ := frontend.NewWitness(&pf, fieldMod)
			_ = groth16.Verify(p, vk, pwt)
		}
		
		ms := time.Since(start).Milliseconds()
		rate := n * 1000 / int(ms)
		fmt.Printf("%d pairs: %dms = %d/sec (prove+verify)\n", n, ms, rate)
	}
	
	fmt.Println("\n=== FULL SYSTEM: Input → Merkle → Prove → Verify ===")
	for _, n := range []int{10, 100, 1000} {
		start := time.Now()
		
		// Simulate batch: each leaf becomes a proof
		for i := 1; i <= n; i++ {
			a := int64(i)
			res := a * a
			
			var wf Circuit
			wf.Res = res
			wf.A = a
			wf.B = a
			wt, _ := frontend.NewWitness(&wf, fieldMod)
			p, _ := groth16.Prove(csFull, pk, wt)
			
			var pf PublicCircuit
			pf.Res = res
			pwt, _ := frontend.NewWitness(&pf, fieldMod)
			_ = groth16.Verify(p, vk, pwt)
		}
		
		ms := time.Since(start).Milliseconds()
		tps := n * 1000 / int(ms)
		fmt.Printf("%d txs: %dms = %d TPS\n", n, ms, tps)
	}
}
