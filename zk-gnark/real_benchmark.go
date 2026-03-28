package main

import (
	"fmt"
	"time"
	"runtime"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

type BatchCircuit struct {
	TxHashes   []frontend.Variable `gnark:"tx_hashes"`
	MerkleRoot frontend.Variable   `gnark:"merkle_root,public"`
}

func (c *BatchCircuit) Define(api frontend.API) error {
	sum := frontend.Variable(0)
	for _, h := range c.TxHashes {
		sum = api.Add(sum, h)
	}
	api.AssertIsEqual(c.MerkleRoot, sum)
	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║          REAL GNARK GROTH16 BENCHMARK                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	
	curve := ecc.BN254
	
	// Test different circuit sizes
	for _, numHashes := range []int{10, 50, 100, 500} {
		circuit := BatchCircuit{
			TxHashes: make([]frontend.Variable, numHashes),
		}
		
		// Compile
		start := time.Now()
		ccs, err := frontend.Compile(curve.ScalarField(), r1cs.NewBuilder, &circuit)
		if err != nil { panic(err) }
		compileMs := time.Since(start).Milliseconds()
		
		// Setup
		start = time.Now()
		pk, _, err := groth16.Setup(ccs)
		if err != nil { panic(err) }
		setupMs := time.Since(start).Milliseconds()
		
		// Create witness
		txHashes := make([]frontend.Variable, numHashes)
		merkleRoot := 0
		for i := 0; i < numHashes; i++ {
			txHashes[i] = i + 1
			merkleRoot += i + 1
		}
		witnessData := BatchCircuit{
			MerkleRoot: merkleRoot,
			TxHashes:   txHashes,
		}
		
		// Prove benchmark
		numProofs := 100
		start = time.Now()
		for i := 0; i < numProofs; i++ {
			w, _ := frontend.NewWitness(&witnessData, curve.ScalarField())
			_, err := groth16.Prove(ccs, pk, w)
			if err != nil { panic(err) }
		}
		proveMs := time.Since(start)
		proveTPS := float64(numProofs) * 1000 / float64(proveMs.Milliseconds())
		
		fmt.Printf("\n  Circuit: %d inputs | Compile: %dms | Setup: %dms\n", 
			numHashes, compileMs, setupMs)
		fmt.Printf("  %d proofs: %.0fms = %.0f TPS\n", 
			numProofs, float64(proveMs.Milliseconds()), proveTPS)
	}
	
	fmt.Printf("\n  ✅ REAL gnark proof generation\n")
}
