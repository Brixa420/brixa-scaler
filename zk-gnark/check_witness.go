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
	
	// Full witness
	w := &Circuit{Res: 100, A: 10, B: 10}
	wt, _ := frontend.NewWitness(w, fieldMod)
	
	proof, _ := groth16.Prove(cs, pk, wt)
	
	// What if we try to extract just the public part?
	// Check what the API expects
	fmt.Printf("Witness public: %v\n", wt.Public())
	fmt.Printf("Witness private: %v\n", wt.Secret())
	
	// Try with public witness created from full witness
	// Actually let's try just using public fields as a separate struct
	
	// Try using Assign with full assignment
	type FullAssign struct {
		Res int64
		A int64  
		B int64
	}
	
	fa := FullAssign{Res: 100, A: 10, B: 10}
	wt2, err := frontend.NewWitness(&fa, fieldMod)
	fmt.Printf("FullAssign witness: %v\n", err)
	
	proof2, _ := groth16.Prove(cs, pk, wt2)
	
	// Now verify - use same full witness
	err = groth16.Verify(proof2, vk, wt2)
	fmt.Printf("Verify with same witness: %v\n", err)
	
	// Try with different values
	fa2 := FullAssign{Res: 200, A: 20, B: 10}
	wt3, _ := frontend.NewWitness(&fa2, fieldMod)
	err = groth16.Verify(proof2, vk, wt3)
	fmt.Printf("Verify with wrong witness (should fail): %v\n", err)
}
