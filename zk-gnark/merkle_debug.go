package main

import (
	"fmt"
	"math/big"
	"github.com/consensys/gnark-crypto/ecc"
)

func main() {
	fieldMod := ecc.BN254.ScalarField()
	
	// Hash function - same as circuit
	hash := func(a, b *big.Int) *big.Int {
		m := new(big.Int).Mul(a, b)
		m.Add(m, a)
		m.Add(m, b)
		return m.Mod(m, fieldMod)
	}
	
	// Test case: leaf=6, siblings=[5,71,119,980179199], bits=[1,0,1,0]
	leaf := big.NewInt(6)
	siblings := []*big.Int{big.NewInt(5), big.NewInt(71), big.NewInt(119), big.NewInt(980179199)}
	bits := []int{1, 0, 1, 0}
	
	fmt.Println("=== Debug ===")
	fmt.Println("Leaf:", leaf)
	fmt.Println("Field mod:", fieldMod)
	fmt.Println()
	
	// Level 0
	fmt.Println("Level 0 (bit=", bits[0], "):")
	if bits[0] == 0 {
		fmt.Printf("  hash(%v, %v) = ", leaf, siblings[0])
	} else {
		fmt.Printf("  hash(%v, %v) = ", siblings[0], leaf)
	}
	// Note: api.Select(bit, if1, if0) - if bit=1, select if1, else if0
	// In my circuit: api.Select(c.Bit0, h0b, h0a) - if Bit0=1, select h0b
	// h0a = hash(leaf, sibling) = hash(6, 5)
	// h0b = hash(sibling, leaf) = hash(5, 6)
	// So if bit=1, use hash(sibling, leaf)
	// if bit=0, use hash(leaf, sibling)
	
	var current *big.Int
	if bits[0] == 0 {
		current = hash(leaf, siblings[0])
	} else {
		current = hash(siblings[0], leaf)
	}
	fmt.Println(current)
	
	// Level 1
	fmt.Println("Level 1 (bit=", bits[1], "):")
	if bits[1] == 0 {
		current = hash(current, siblings[1])
	} else {
		current = hash(siblings[1], current)
	}
	fmt.Println("  hash =", current)
	
	// Level 2
	fmt.Println("Level 2 (bit=", bits[2], "):")
	if bits[2] == 0 {
		current = hash(current, siblings[2])
	} else {
		current = hash(siblings[2], current)
	}
	fmt.Println("  hash =", current)
	
	// Level 3
	fmt.Println("Level 3 (bit=", bits[3], "):")
	if bits[3] == 0 {
		current = hash(current, siblings[3])
	} else {
		current = hash(siblings[3], current)
	}
	fmt.Println("  hash =", current)
	
	fmt.Println("\n=== Computed root:", current, "===")
	
	// Verify the tree is being built correctly
	fmt.Println("\n=== Verify tree build ===")
	leaves := make([]*big.Int, 16)
	for i := 0; i < 16; i++ {
		leaves[i] = big.NewInt(int64(i + 1))
	}
	
	level := leaves
	for len(level) > 1 {
		fmt.Println("Level len:", len(level))
		next := make([]*big.Int, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := level[i]
			if i+1 < len(level) {
				right = level[i+1]
			}
			next = append(next, hash(level[i], right))
		}
		level = next
	}
	fmt.Println("Final root:", level[0])
}
