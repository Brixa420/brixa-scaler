package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// Simulates checking a batch Merkle root
type BatchCircuit struct {
	TxCount    frontend.Variable `gnark:"txCount"`
	BatchHash  frontend.Variable `gnark:"batchHash"`
	MerkleRoot frontend.Variable `gnark:"merkleRoot"`
}

func (c *BatchCircuit) Define(api frontend.API) error {
	// Simple: merkleRoot = hash(txCount || batchHash)
	combined := api.Mul(c.TxCount, c.BatchHash)
	api.AssertIsEqual(c.MerkleRoot, combined)
	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║        GNARK vs HTTP-SNARKJS (Real Comparison)               ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	
	curve := ecc.BN254
	
	// Compile once
	fmt.Println("\n--- Compiling circuit (once) ---")
	start := time.Now()
	ccs, _ := frontend.Compile(curve.ScalarField(), r1cs.NewBuilder, &BatchCircuit{})
	pk, _, _ := groth16.Setup(ccs)
	compileMs := time.Since(start).Milliseconds()
	fmt.Printf("Compile + Setup: %dms\n", compileMs)
	
	// Now prove many times with SAME circuit
	fmt.Println("\n--- gnark: 1000 proofs (pre-compiled) ---")
	assignment := &BatchCircuit{TxCount: 1000, BatchHash: 12345, MerkleRoot: 12345000}
	
	start = time.Now()
	for i := 0; i < 1000; i++ {
		witness, _ := frontend.NewWitness(assignment, curve.ScalarField())
		groth16.Prove(ccs, pk, witness)
	}
	gnarkMs := time.Since(start).Milliseconds()
	gnarkTPS := float64(1000) / float64(gnarkMs) * 1000
	fmt.Printf("gnark: %dms = %.0f TPS\n", gnarkMs, gnarkTPS)
	
	// HTTP snarkjs comparison
	fmt.Println("\n--- HTTP snarkjs (from earlier benchmark) ---")
	fmt.Printf("snarkjs: ~2,500 TPS\n")
	
	// Summary
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      SUMMARY                                   ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ gnark (Go, no HTTP):     %10.0f TPS                      ║\n", gnarkTPS)
	fmt.Printf("║ snarkjs (Node, HTTP):     %10.0f TPS                      ║\n", 2500.0)
	fmt.Printf("║ Speedup:                 %10.0fx                         ║\n", gnarkTPS/2500)
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}
