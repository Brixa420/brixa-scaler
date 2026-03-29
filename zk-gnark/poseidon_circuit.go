package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/constraint"
)

// Poseidon-like hash using basic operations
type PoseidonCircuit struct {
	InputA frontend.Variable `gnark:",public"`
	InputB frontend.Variable `gnark:"secret"`
	Output frontend.Variable `gnark:"secret"`
}

// sbox: x -> x^5 (like Poseidon)
func sbox(api frontend.API, x frontend.Variable) frontend.Variable {
	x2 := api.Mul(x, x)
	x4 := api.Mul(x2, x2)
	return api.Mul(x4, x)
}

// Hash two field elements
func hash2(api frontend.API, a, b frontend.Variable) frontend.Variable {
	sa := sbox(api, a)
	sb := sbox(api, b)
	m := api.Mul(sa, sb)
	result := api.Add(m, a)
	result = api.Add(result, b)
	return sbox(api, result)
}

func (c *PoseidonCircuit) Define(api frontend.API) error {
	result := hash2(api, c.InputA, c.InputB)
	api.AssertIsEqual(c.Output, result)
	return nil
}

// PublicCircuit for verification
type PublicCircuit struct {
	InputA frontend.Variable `gnark:",public"`
}

func (c *PublicCircuit) Define(api frontend.API) error {
	return nil
}

var pk groth16.ProvingKey
var vk groth16.VerifyingKey
var csFull, csPublic constraint.ConstraintSystem
var fieldMod *big.Int

func main() {
	fieldMod = ecc.BN254.ScalarField()
	
	fmt.Println("=== Poseidon-like Hash Circuit ===")
	
	csFull, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &PoseidonCircuit{})
	csPublic, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &PublicCircuit{})
	pk, vk, _ = groth16.Setup(csFull)
	
	fmt.Println("Constraints:", csFull.GetNbConstraints())
	
	// Test with a=5, b=7
	a := big.NewInt(5)
	b := big.NewInt(7)
	
	// Compute expected: hash2(5, 7)
	a5 := new(big.Int).Exp(a, big.NewInt(5), fieldMod)
	b5 := new(big.Int).Exp(b, big.NewInt(5), fieldMod)
	m := new(big.Int).Mul(a5, b5)
	m.Mod(m, fieldMod)
	t := new(big.Int).Add(m, a)
	t.Add(t, b)
	t.Mod(t, fieldMod)
	t5 := new(big.Int).Exp(t, big.NewInt(5), fieldMod)
	expected := new(big.Int).Mod(t5, fieldMod)
	
	fmt.Println("Input a:", a, "b:", b)
	fmt.Println("Expected output:", expected)
	
	// Create witness
	var witness PoseidonCircuit
	witness.InputA = a
	witness.InputB = b
	witness.Output = expected
	
	start := time.Now()
	wt, _ := frontend.NewWitness(&witness, fieldMod)
	proof, err := groth16.Prove(csFull, pk, wt)
	proveMs := time.Since(start).Milliseconds()
	
	if err != nil {
		fmt.Println("Prove error:", err)
		return
	}
	fmt.Println("Proved in", proveMs, "ms")
	
	// Verify with public witness
	var pubWitness PublicCircuit
	pubWitness.InputA = a
	publicWt, _ := frontend.NewWitness(&pubWitness, fieldMod)
	
	start = time.Now()
	err = groth16.Verify(proof, vk, publicWt)
	verifyMs := time.Since(start).Milliseconds()
	
	fmt.Println("Verified in", verifyMs, "ms, valid:", err == nil)
	
	var buf bytes.Buffer
	proof.WriteTo(&buf)
	fmt.Println("Proof size:", buf.Len(), "bytes")
	fmt.Println("Proof:", hex.EncodeToString(buf.Bytes()[:32]), "...")
}
