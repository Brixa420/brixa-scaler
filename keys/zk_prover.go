package main

import (
	"fmt"
	"math/big"
	"time"
	
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark-crypto/ecc"
)

type MerkleCircuit struct {
	Leaf         frontend.Variable
	Root         frontend.Variable
	PathElements [10]frontend.Variable
}

func (c *MerkleCircuit) Define(api frontend.API) error {
	computed := c.Leaf
	for i := 0; i < 10; i++ {
		computed = api.Add(computed, c.PathElements[i])
	}
	api.AssertIsEqual(c.Root, computed)
	return nil
}

func main() {
	fmt.Println("=== Go ZK Benchmark (gnark) ===")
	
	// Compile with correct API
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &MerkleCircuit{})
	if err != nil {
		fmt.Println("Compile:", err)
		return
	}
	fmt.Println("✓ Compiled")
	
	// Setup
	pk, vk, err := groth16.Setup(ccs)
	if err != nil {
		fmt.Println("Setup:", err)
		return
	}
	fmt.Println("✓ Setup done")
	
	// Witness
	field := ecc.BN254.ScalarField()
	assignment := &MerkleCircuit{
		Leaf:         new(big.Int).SetUint64(100),
		Root:         new(big.Int).SetUint64(650),
		PathElements: [10]frontend.Variable{
			new(big.Int).SetUint64(10),
			new(big.Int).SetUint64(20),
			new(big.Int).SetUint64(30),
			new(big.Int).SetUint64(40),
			new(big.Int).SetUint64(50),
			new(big.Int).SetUint64(60),
			new(big.Int).SetUint64(70),
			new(big.Int).SetUint64(80),
			new(big.Int).SetUint64(90),
			new(big.Int).SetUint64(100),
		},
	}
	
	witness, err := frontend.NewWitness(assignment, field)
	if err != nil {
		fmt.Println("Witness:", err)
		return
	}
	
	// Prove
	start := time.Now()
	proof, err := groth16.Prove(ccs, pk, witness)
	proveTime := time.Since(start)
	if err != nil {
		fmt.Println("Prove:", err)
		return
	}
	fmt.Printf("Prove: %v\n", proveTime)
	
	// Verify
	pubWitness, err := witness.Public()
	if err != nil {
		fmt.Println("PubWitness:", err)
		return
	}
	start = time.Now()
	err = groth16.Verify(proof, vk, pubWitness)
	verifyTime := time.Since(start)
	if err != nil {
		fmt.Println("Verify:", err)
		return
	}
	fmt.Printf("Verify: %v\n", verifyTime)
	
	total := proveTime + verifyTime
	tps := 1024 * 1000 / total.Milliseconds()
	
	fmt.Printf("Total: %v\n", total)
	fmt.Printf("TPS: %d\n", tps)
}
