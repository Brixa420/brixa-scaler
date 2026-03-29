package main

import (
	"fmt"
	"math/big"
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
	
	fmt.Println("Testing gnark verification...")
	
	cs, _ := frontend.Compile(fieldMod, r1cs.NewBuilder, &Circuit{})
	pk, vk, _ := groth16.Setup(cs)
	
	// Create witness
	w := &Circuit{Res: 100, A: 10, B: 10}
	wt, _ := frontend.NewWitness(w, fieldMod)
	
	// Prove
	proof, err := groth16.Prove(cs, pk, wt)
	if err != nil { panic(err) }
	
	// Verify with public witness
	pubW := &Circuit{Res: 100}
	publicWt, _ := frontend.NewWitness(pubW, fieldMod)
	
	start := time.Now()
	err = groth16.Verify(proof, vk, publicWt)
	ms := time.Since(start).Milliseconds()
	
	fmt.Printf("Verify: %dms, err=%v\n", ms, err)
}
