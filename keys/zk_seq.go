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
	ccs, _ := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &MerkleCircuit{})
	pk, vk, _ := groth16.Setup(ccs)
	field := ecc.BN254.ScalarField()
	
	fmt.Println("=== Sequential Proving ===")
	
	for _, n := range []int{1, 10, 100} {
		start := time.Now()
		
		for i := 0; i < n; i++ {
			assignment := &MerkleCircuit{
				Leaf: new(big.Int).SetUint64(uint64(100 + i)),
				Root: new(big.Int).SetUint64(uint64(650 + i)),
				PathElements: [10]frontend.Variable{
					new(big.Int).SetUint64(10), new(big.Int).SetUint64(20),
					new(big.Int).SetUint64(30), new(big.Int).SetUint64(40),
					new(big.Int).SetUint64(50), new(big.Int).SetUint64(60),
					new(big.Int).SetUint64(70), new(big.Int).SetUint64(80),
					new(big.Int).SetUint64(90), new(big.Int).SetUint64(100),
				},
			}
			w, _ := frontend.NewWitness(assignment, field)
			p, _ := groth16.Prove(ccs, pk, w)
			pub, _ := w.Public()
			_ = groth16.Verify(p, vk, pub)
		}
		
		t := time.Since(start)
		tps := n * 1024 * 1000 / int(t.Milliseconds())
		fmt.Printf("%d proofs: %v | %d TPS\n", n, t, tps)
	}
}
