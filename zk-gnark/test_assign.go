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
	
	// Use assignment map - this is the correct way in gnark
	fullAssignment := map[string]interface{}{
		"Res": 100,
		"A": 10,
		"B": 10,
	}
	
	wt, err := frontend.NewWitness(fullAssignment, fieldMod)
	fmt.Printf("Full witness: %v\n", err)
	
	proof, err := groth16.Prove(cs, pk, wt)
	fmt.Printf("Prove: %v\n", err)
	
	// For verification, use public-only assignment
	publicAssignment := map[string]interface{}{
		"Res": 100,
	}
	
	publicWt, err := frontend.NewWitness(publicAssignment, fieldMod)
	fmt.Printf("Public witness: %v\n", err)
	
	err = groth16.Verify(proof, vk, publicWt)
	fmt.Printf("Verify: %v\n", err)
}
