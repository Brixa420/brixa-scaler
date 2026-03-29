package main

import (
	"fmt"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/plonk"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/scs"
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
	cs, _ := frontend.Compile(mod, scs.NewBuilder, &Simple{})
	pk, vk, _ := plonk.Setup(cs)
	fmt.Println("Constraints:", cs.GetNbConstraints())

	w := &Simple{Res: 100, A: 10, B: 10}
	wt, _ := frontend.NewWitness(w, mod)
	p, _ := plonk.Prove(cs, pk, wt)
	
	var pub Simple
	pub.Res = 100
	pwt, _ := frontend.NewWitness(&pub, mod)
	err := plonk.Verify(cs, p, vk, pwt)
	
	fmt.Println("Valid:", err == nil)
}
