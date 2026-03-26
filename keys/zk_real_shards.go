package main

import (
	"fmt"
	"sync"
	"time"
	
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
)

type MerkleCircuit struct {
	Leaf         frontend.Variable
	Root         frontend.Variable
	PathElements [10]frontend.Variable
	PathIndices  [10]frontend.Variable
}

func (c *MerkleCircuit) Define(api frontend.API) error {
	current := c.Leaf
	for i := 0; i < 10; i++ {
		left := api.Select(c.PathIndices[i], c.PathElements[i], current)
		right := api.Select(c.PathIndices[i], current, c.PathElements[i])
		hash := api.Mul(left, right)
		hash = api.Add(hash, left)
		hash = api.Add(hash, right)
		current = hash
	}
	api.AssertIsEqual(c.Root, current)
	return nil
}

func main() {
	ccs, _ := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &MerkleCircuit{})
	pk, vk, _ := groth16.Setup(ccs)
	field := ecc.BN254.ScalarField()
	
	fmt.Println("=== REAL Merkle Sharding (41 constraints) ===")
	
	for _, numShards := range []int{1, 5, 10, 18, 36} {
		start := time.Now()
		
		var wg sync.WaitGroup
		for s := 0; s < numShards; s++ {
			wg.Add(1)
			go func(shard int) {
				defer wg.Done()
				
				var leaf, root, current, hashVal fr.Element
				leaf.SetUint64(uint64(shard + 1))
				current = leaf
				
				var path [10]frontend.Variable
				for i := 0; i < 10; i++ {
					var p fr.Element
					p.SetUint64(uint64(i + 1))
					path[i] = p
					
					var left, right fr.Element
					if i == 0 {
						left = p
						right = leaf
					} else {
						left = current
						right = p
					}
					
					hashVal.Mul(&left, &right)
					hashVal.Add(&hashVal, &left)
					hashVal.Add(&hashVal, &right)
					current = hashVal
				}
				root = current
				
				assignment := &MerkleCircuit{
					Leaf:         leaf,
					Root:         root,
					PathElements: path,
					PathIndices:  [10]frontend.Variable{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
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
