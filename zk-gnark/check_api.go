package main

import (
	"fmt"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark-crypto/ecc"
)

// Simple hash-like circuit: x^3 + y for mixing
type HashCircuit struct {
	A frontend.Variable `gnark:",public"`
	B frontend.Variable `gnark:"secret"`
	Hash frontend.Variable `gnark:"secret"`
}

func (c *HashCircuit) Define(api frontend.API) error {
	// Simple collision-resistant-ish hash: (a * b)^3 + a + b
	m := api.Mul(c.A, c.B)
	squared := api.Mul(m, m)
	cubed := api.Mul(squared, m)
	
	result := api.Add(api.Add(cubed, c.A), c.B)
	
	api.AssertIsEqual(c.Hash, result)
	return nil
}

func main() {
	fieldMod := ecc.BN254.ScalarField()
	cs, err := frontend.Compile(fieldMod, r1cs.NewBuilder, &HashCircuit{})
	if err != nil {
		fmt.Println("Compile error:", err)
		return
	}
	fmt.Println("Constraints:", cs.GetNbConstraints())
}
