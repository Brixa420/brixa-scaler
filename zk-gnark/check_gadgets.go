package main

import (
	"fmt"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/std"
)

type HashCircuit struct {
	Input  frontend.Variable `gnark:",public"`
	Output frontend.Variable `gnark:"secret"`
}

func (c *HashCircuit) Define(api frontend.API) error {
	// Try SHA256 gadget - it's in gnark/std
	h, err := std.NewSHA256(api, 256)
	if err != nil {
		fmt.Println("SHA256 error:", err)
		return err
	}
	
	// Hash the input (as bytes)
	h.Write(c.Input, 32)
	digest := h.Sum()
	
	c.Output = digest[0]
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
