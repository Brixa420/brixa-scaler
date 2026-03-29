package main

import (
	"bytes"
	"fmt"
	"math/big"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

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

func main() {
	fieldMod := ecc.BN254.ScalarField()
	
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  ZK PROOF BENCHMARK                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	
	cs, _ := frontend.Compile(fieldMod, r1cs.NewBuilder, &Circuit{})
	pk, _, _ := groth16.Setup(cs)
	fmt.Printf("Circuit: %d constraints\n\n", cs.GetNbConstraints())
	
	// Prepare test data
	leaves := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4)}
	root := hash(hash(leaves[0], leaves[1], fieldMod), hash(leaves[2], leaves[3], fieldMod), fieldMod)
	proof := []*big.Int{leaves[2], leaves[3], big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
	
	var w Circuit
	w.Leaf = leaves[0]
	w.Root = root
	for i := 0; i < 10; i++ {
		w.Siblings[i] = proof[i]
		w.Bits[i] = big.NewInt(0)
	}
	
	fmt.Println("=== PROVING BENCHMARK ===")
	
	for _, count := range []int{10, 50, 100, 500, 1000} {
		start := time.Now()
		for i := 0; i < count; i++ {
			wt, _ := frontend.NewWitness(&w, fieldMod)
			p, _ := groth16.Prove(cs, pk, wt)
			var buf bytes.Buffer
			p.WriteTo(&buf)
		}
		ms := time.Since(start).Milliseconds()
		rate := count * 1000 / int(ms)
		fmt.Printf("%d proofs: %dms = %d proofs/sec\n", count, ms, rate)
	}
	
	fmt.Println("\n=== SINGLE PROOF ===")
	start := time.Now()
	wt, _ := frontend.NewWitness(&w, fieldMod)
	p, _ := groth16.Prove(cs, pk, wt)
	var buf bytes.Buffer
	p.WriteTo(&buf)
	ms := time.Since(start).Microseconds()
	
	fmt.Printf("Prove: %.2fms, Size: %d bytes\n", float64(ms)/1000, buf.Len())
}
