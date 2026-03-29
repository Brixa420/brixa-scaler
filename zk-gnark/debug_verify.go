package main

import (
	"fmt"
	"reflect"

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
	
	// Create witness with full assignment
	w := &Circuit{Res: 100, A: 10, B: 10}
	wt, _ := frontend.NewWitness(w, fieldMod)
	
	fmt.Printf("Full witness: len=%d\n", len(wt.Public()))
	
	// Prove
	proof, err := groth16.Prove(cs, pk, wt)
	if err != nil { panic(err) }
	
	// Inspect the proof
	fmt.Printf("Proof type: %s\n", reflect.TypeOf(proof))
	fmt.Printf("Proof value: %v\n", proof)
	
	// Create PUBLIC-ONLY witness (this is what we're doing wrong?)
	// Try with just the public variable
	pubW := map[string]interface{}{"Res": 100}
	publicWt, _ := frontend.NewWitness(pubW, fieldMod)
	
	fmt.Printf("Public witness: len=%d\n", len(publicWt.Public()))
	
	// Try verifying with different witness types
	fmt.Println("\nTrying verification...")
	
	// Try 1: Full witness
	err = groth16.Verify(proof, vk, wt)
	fmt.Printf("Full witness: %v\n", err)
	
	// Try 2: Just public (what we tried)
	err = groth16.Verify(proof, vk, publicWt)
	fmt.Printf("Public only: %v\n", err)
}
