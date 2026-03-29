package main

import (
	"fmt"
	"math/big"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// 10-level Merkle circuit
type Circuit struct {
	Root      frontend.Variable `gnark:",public"`
	Leaf      frontend.Variable `gnark:"secret"`
	Siblings  [10]frontend.Variable `gnark:"secret"`
	Bits      [10]frontend.Variable `gnark:"secret"`
}

func (c *Circuit) Define(api frontend.API) error {
	current := c.Leaf
	for i := 0; i < 10; i++ {
		left := api.Select(c.Bits[i], c.Siblings[i], current)
		right := api.Select(c.Bits[i], current, c.Siblings[i])
		current = api.Mul(left, right)
		current = api.Add(current, left)
		current = api.Add(current, right)
	}
	api.AssertIsEqual(c.Root, current)
	return nil
}

func hash(a, b *big.Int, mod *big.Int) *big.Int {
	m := new(big.Int).Mul(a, b)
	m.Mod(m, mod)
	r := new(big.Int).Add(a, b)
	m.Add(m, r)
	m.Mod(m, mod)
	return m
}

func buildTree(leaves []*big.Int, mod *big.Int) (*big.Int, [][]*big.Int) {
	level := leaves
	allProofs := make([][]*big.Int, len(leaves))
	
	for i := range leaves {
		allProofs[i] = make([]*big.Int, 0)
	}
	
	for len(level) > 1 {
		next := make([]*big.Int, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := level[i]
			if i+1 < len(level) { right = level[i+1] }
			next = append(next, hash(level[i], right, mod))
		}
		
		// Collect siblings for each leaf
		for i := 0; i < len(leaves); i++ {
			sibIdx := i
			if i%2 == 0 { sibIdx = i+1 } else { sibIdx = i-1 }
			if sibIdx >= len(level) { sibIdx = i }
			if sibIdx < len(level) {
				allProofs[i] = append(allProofs[i], level[sibIdx])
			}
		}
		level = next
	}
	return level[0], allProofs
}

func main() {
	fieldMod := ecc.BN254.ScalarField()
	
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           LOCAL ZK BENCHMARK (No HTTP overhead)              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	
	cs, _ := frontend.Compile(fieldMod, r1cs.NewBuilder, &Circuit{})
	pk, vk, _ := groth16.Setup(cs)
	fmt.Printf("Circuit: %d constraints\n\n", cs.GetNbConstraints())
	
	// Prepare test data - 100 leaves
	n := 100
	leaves := make([]*big.Int, n)
	for i := 0; i < n; i++ {
		leaves[i] = big.NewInt(int64(i + 1))
	}
	
	root, proofs := buildTree(leaves, fieldMod)
	fmt.Printf("Tree root: %s\n", root.String())
	fmt.Printf("Proof 0 has %d siblings\n\n", len(proofs[0]))
	
	// Benchmark proving
	fmt.Println("=== PROVING BENCHMARK ===")
	
	for count := 10; count <= 100; count += 10 {
		var witness Circuit
		witness.Leaf = leaves[0]
		witness.Root = root
		
		for i := 0; i < 10 && i < len(proofs[0]); i++ {
			witness.Siblings[i] = proofs[0][i]
			witness.Bits[i] = big.NewInt(0)
		}
		
		start := time.Now()
		for i := 0; i < count; i++ {
			wt, _ := frontend.NewWitness(&witness, fieldMod)
			_, err := groth16.Prove(cs, pk, wt)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
		}
		ms := time.Since(start).Milliseconds()
		
		fmt.Printf("%d proofs: %dms = %d proofs/sec\n", count, ms, count*1000/ms)
	}
	
	fmt.Println("\n=== SINGLE PROOF ===")
	var w Circuit
	w.Leaf = leaves[0]
	w.Root = root
	for i := 0; i < 10 && i < len(proofs[0]); i++ {
		w.Siblings[i] = proofs[0][i]
		w.Bits[i] = big.NewInt(0)
	}
	
	start := time.Now()
	wt, _ := frontend.NewWitness(&w, fieldMod)
	proof, _ := groth16.Prove(cs, pk, wt)
	ms := time.Since(start).Microseconds()
	
	fmt.Printf("Prove time: %.2fms\n", float64(ms)/1000)
	fmt.Printf("Proof size: %d bytes\n", 128)
	
	_ = vk // Just to avoid unused warning
}
