package main

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	gnarkcrypto_mimc "github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/hash"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	gnark_mimc "github.com/consensys/gnark/std/hash/mimc"
)

// BatchVerifyCircuit from before
type BatchVerifyCircuit struct {
	MerkleRoot frontend.Variable `gnark:"merkleRoot,public"`
	TxCount    frontend.Variable `gnark:"txCount,public"`
	TxData     [32]frontend.Variable `gnark:"txData"`
}

func (c *BatchVerifyCircuit) Define(api frontend.API) error {
	mimc, _ := gnark_mimc.NewMiMC(api)
	leaves := make([]frontend.Variable, 32)
	for i := 0; i < 32; i++ {
		mimc.Reset()
		mimc.Write(c.TxData[i])
		leaves[i] = mimc.Sum()
	}
	current := leaves
	for len(current) > 1 {
		nextLen := (len(current) + 1) / 2
		next := make([]frontend.Variable, nextLen)
		for i := 0; i < nextLen; i++ {
			mimc.Reset()
			left := current[i*2]
			if i*2+1 < len(current) {
				mimc.Write(left)
				mimc.Write(current[i*2+1])
			} else {
				mimc.Write(left)
				mimc.Write(left)
			}
			next[i] = mimc.Sum()
		}
		current = next
	}
	api.AssertIsEqual(c.MerkleRoot, current[0])
	api.AssertIsEqual(c.TxCount, 32)
	return nil
}

func computeMerkleRoot(txData []*big.Int) *big.Int {
	h := gnarkcrypto_mimc.NewMiMC().(hash.StateStorer)
	leaves := make([]fr.Element, 32)
	for i := 0; i < 32; i++ {
		h.Reset()
		var e fr.Element
		e.SetBigInt(txData[i])
		bytes := e.Bytes()
		h.Write(bytes[:])
		sum := h.Sum(nil)
		leaves[i].SetBytes(sum)
	}
	current := leaves
	for len(current) > 1 {
		nextLen := (len(current) + 1) / 2
		next := make([]fr.Element, nextLen)
		for i := 0; i < nextLen; i++ {
			h.Reset()
			left := current[i*2]
			leftBytes := left.Bytes()
			rightBytes := current[i*2+1].Bytes()
			if i*2+1 < len(current) {
				h.Write(leftBytes[:])
				h.Write(rightBytes[:])
			} else {
				h.Write(leftBytes[:])
				h.Write(leftBytes[:])
			}
			sum := h.Sum(nil)
			next[i].SetBytes(sum)
		}
		current = next
	}
	return current[0].BigInt(new(big.Int))
}

func main() {
	numTxs := 32
	numProofs := 100
	numWorkers := runtime.NumCPU()
	
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &BatchVerifyCircuit{})
	if err != nil {
		panic(err)
	}
	pk, _, err := groth16.Setup(ccs)
	if err != nil {
		panic(err)
	}
	
	txDataRaw := make([]*big.Int, 32)
	var txData [32]frontend.Variable
	for i := 0; i < 32; i++ {
		txDataRaw[i] = big.NewInt(int64(i + 1))
		txData[i] = big.NewInt(int64(i + 1))
	}
	merkleRoot := computeMerkleRoot(txDataRaw)
	witness := &BatchVerifyCircuit{
		MerkleRoot: merkleRoot,
		TxCount:    32,
		TxData:     txData,
	}
	
	fmt.Printf("=== Parallel ZK Proving (%d workers) ===\n", numWorkers)
	fmt.Printf("Circuit: %d constraints, %d txs\n\n", ccs.GetNbConstraints(), numTxs)
	
	// Single-threaded baseline
	start := time.Now()
	for i := 0; i < numProofs; i++ {
		w, _ := frontend.NewWitness(witness, ecc.BN254.ScalarField())
		groth16.Prove(ccs, pk, w)
	}
	singleMs := time.Since(start).Milliseconds()
	singleTPS := float64(numProofs) / (float64(singleMs) / 1000)
	fmt.Printf("Single-threaded: %dms, %.0f TPS\n", singleMs, singleTPS)
	
	// Parallel proving
	start = time.Now()
	var wg sync.WaitGroup
	 proofs := make(chan int, numProofs)
	
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range proofs {
				w, _ := frontend.NewWitness(witness, ecc.BN254.ScalarField())
				groth16.Prove(ccs, pk, w)
			}
		}()
	}
	
	for i := 0; i < numProofs; i++ {
		proofs <- i
	}
	close(proofs)
	wg.Wait()
	
	parallelMs := time.Since(start).Milliseconds()
	parallelTPS := float64(numProofs) / (float64(parallelMs) / 1000)
	
	fmt.Printf("Parallel (%dw):   %dms, %.0f TPS\n", numWorkers, parallelMs, parallelTPS)
	fmt.Printf("Speedup: %.1fx\n", float64(singleMs)/float64(parallelMs))
	
	// Extrapolate to 12K TPS
	targetTPS := 12000.0
	neededWorkers := int(targetTPS / singleTPS)
	if neededWorkers < 1 {
		neededWorkers = 1
	}
	fmt.Printf("\n→ For 12K TPS: need ~%d parallel workers\n", neededWorkers)
}
