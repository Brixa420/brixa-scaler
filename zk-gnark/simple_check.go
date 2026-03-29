package main

import (
	"fmt"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

type Simple struct {
	Res frontend.Variable `gnark:",public"`
	A, B frontend.Variable `gnark:"secret"`
}

func (s *Simple) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(s.A, s.B), s.Res)
	return nil
}

func main() {
	mod := ecc.BN254.ScalarField()
	cs, _ := frontend.Compile(mod, r1cs.NewBuilder, &Simple{})
	pk, vk, _ := groth16.Setup(cs)
	fmt.Println("Constraints:", cs.GetNbConstraints())

	w := &Simple{Res: 100, A: 10, B: 10}
	wt, _ := frontend.NewWitness(w, mod)
	p, _ := groth16.Prove(cs, pk, wt)
	
	var pub Simple
	pub.Res = 100
	pwt, _ := frontend.NewWitness(&pub, mod)
	err := groth16.Verify(p, vk, pwt)
	
	fmt.Println("Valid:", err == nil)
}
