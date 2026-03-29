package main

import (
	"fmt"

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
	
	cs, _ := frontend.Compile(fieldMod, r1cs.NewBuilder, &Circuit{})
	pk, vk, _ := groth16.Setup(cs)
	
	// Create full witness
	w := &Circuit{Res: 100, A: 10, B: 10}
	wt, _ := frontend.NewWitness(w, fieldMod)
	
	proof, _ := groth16.Prove(cs, pk, wt)
	
	fmt.Println("=== Inspecting API ===")
	
	// Just verify with same witness we proved with
	err := groth16.Verify(proof, vk, wt)
	fmt.Printf("Verify with full witness: %v\n", err)
	
	_ = vk
	_ = fieldMod
}
