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
	numWorkers := runtime.NumCPU()
	numBatches := 100
	
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &BatchVerifyCircuit{})
	if err != nil {
		panic(err)
	}
	pk, _, err := groth16.Setup(ccs)
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("=== Parallel Batch Proving (%d workers, %d batches) ===\n", numWorkers, numBatches)
	fmt.Printf("Circuit: %d constraints\n\n", ccs.GetNbConstraints())
	
	// Sequential: 100 batches one after another
	start := time.Now()
	for batch := 0; batch < numBatches; batch++ {
		txDataRaw := make([]*big.Int, 32)
		var txData [32]frontend.Variable
		for i := 0; i < 32; i++ {
			txDataRaw[i] = big.NewInt(int64(batch*32 + i + 1))
			txData[i] = big.NewInt(int64(batch*32 + i + 1))
		}
		merkleRoot := computeMerkleRoot(txDataRaw)
		witness := &BatchVerifyCircuit{
			MerkleRoot: merkleRoot,
			TxCount:    32,
			TxData:     txData,
		}
		w, _ := frontend.NewWitness(witness, ecc.BN254.ScalarField())
		groth16.Prove(ccs, pk, w)
	}
	seqMs := time.Since(start).Milliseconds()
	seqTPS := float64(numBatches*32) / (float64(seqMs) / 1000)
	fmt.Printf("Sequential: %dms, %.0f TPS (%d batches)\n", seqMs, seqTPS, numBatches)
	
	// Parallel: each worker does different batches
	type result struct {
		batch int
		timeMs int64
	}
	results := make(chan result, numBatches)
	
	start = time.Now()
	var wg sync.WaitGroup
	
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for batch := workerID; batch < numBatches; batch += numWorkers {
				batchStart := time.Now()
				
				txDataRaw := make([]*big.Int, 32)
				var txData [32]frontend.Variable
				for i := 0; i < 32; i++ {
					txDataRaw[i] = big.NewInt(int64(batch*32 + i + 1))
					txData[i] = big.NewInt(int64(batch*32 + i + 1))
				}
				merkleRoot := computeMerkleRoot(txDataRaw)
				witness := &BatchVerifyCircuit{
					MerkleRoot: merkleRoot,
					TxCount:    32,
					TxData:     txData,
				}
				w, _ := frontend.NewWitness(witness, ecc.BN254.ScalarField())
				groth16.Prove(ccs, pk, w)
				
				results <- result{batch: batch, timeMs: time.Since(batchStart).Milliseconds()}
			}
		}(w)
	}
	
	go func() {
		wg.Wait()
		close(results)
	}()
	
	for range results {
		// just drain
	}
	
	parallelMs := time.Since(start).Milliseconds()
	parallelTPS := float64(numBatches*32) / (float64(parallelMs) / 1000)
	
	fmt.Printf("Parallel:   %dms, %.0f TPS (%d batches)\n", parallelMs, parallelTPS, numBatches)
	fmt.Printf("Speedup: %.1fx\n", float64(seqMs)/float64(parallelMs))
	
	// Extrapolate to 12K TPS
	targetTPS := 12000.0
	neededWorkers := int(targetTPS / seqTPS)
	if neededWorkers < 1 { neededWorkers = 1 }
	fmt.Printf("\n→ Current: %.0f TPS\n", seqTPS)
	fmt.Printf("→ For 12K TPS: need %.1fx speedup\n", targetTPS/seqTPS)
}
