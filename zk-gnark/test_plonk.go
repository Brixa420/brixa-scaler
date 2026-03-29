package main

import (
	"fmt"
	"math/big"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/plonk"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/scs"
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
	
	fmt.Println("Testing PLONK verification...")
	
	cs, _ := frontend.Compile(fieldMod, scs.NewBuilder, &Circuit{})
	srs, err := plonk.NewSRS(cs) // Auto-generates SRS
	if err != nil { panic(err) }
	
	pk, vk, err := plonk.Setup(cs, srs)
	if err != nil { panic(err) }
	
	// Create witness
	w := &Circuit{Res: 100, A: 10, B: 10}
	wt, _ := frontend.NewWitness(w, fieldMod)
	
	// Prove
	proof, err := plonk.Prove(cs, pk, wt)
	if err != nil { panic(err) }
	
	// Verify
	pubW := &Circuit{Res: 100}
	publicWt, _ := frontend.NewWitness(pubW, fieldMod)
	
	start := time.Now()
	err = plonk.Verify(proof, vk, publicWt)
	ms := time.Since(start).Milliseconds()
	
	fmt.Printf("PLONK Verify: %dms, err=%v\n", ms, err)
}
