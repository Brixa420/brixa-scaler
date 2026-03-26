package main

import (
	"fmt"
	"math/big"
	"sync"
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
	ccs, _ := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &MerkleCircuit{})
	pk, vk, _ := groth16.Setup(ccs)
	field := ecc.BN254.ScalarField()
	
	fmt.Println("=== Go Sharding ===")
	
	for _, numShards := range []int{1, 5, 10, 20, 36} {
		start := time.Now()
		
		var wg sync.WaitGroup
		for s := 0; s < numShards; s++ {
			wg.Add(1)
			go func(shard int) {
				defer wg.Done()
				
				assignment := &MerkleCircuit{
					Leaf: new(big.Int).SetUint64(uint64(100 + shard)),
					Root: new(big.Int).SetUint64(uint64(650 + shard)),
					PathElements: [10]frontend.Variable{
						new(big.Int).SetUint64(10), new(big.Int).SetUint64(20),
						new(big.Int).SetUint64(30), new(big.Int).SetUint64(40),
						new(big.Int).SetUint64(50), new(big.Int).SetUint64(60),
						new(big.Int).SetUint64(70), new(big.Int).SetUint64(80),
						new(big.Int).SetUint64(90), new(big.Int).SetUint64(100),
					},
				}
				
				witness, _ := frontend.NewWitness(assignment, field)
				proof, _ := groth16.Prove(ccs, pk, witness)
				pubWitness, _ := witness.Public()
				_ = groth16.Verify(proof, vk, pubWitness)
			}(s)
		}
		
		wg.Wait()
		t := time.Since(start)
		txs := numShards * 1024
		tps := txs * 1000 / int(t.Milliseconds())
		
		fmt.Printf("%d shards: %v | %d txs | %d TPS\n", numShards, t, txs, tps)
	}
}
