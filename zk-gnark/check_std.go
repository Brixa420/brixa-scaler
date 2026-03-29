package main

import (
	"fmt"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/std/sha256"
)

type HashCircuit struct {
	Input frontend.Variable `gnark:",public"`
}

func (c *HashCircuit) Define(api frontend.API) error {
	// Try SHA256 from std/sha256
	h := sha256.New(api)
	h.Write([]byte("hello"))
	_ = h.Sum()
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
