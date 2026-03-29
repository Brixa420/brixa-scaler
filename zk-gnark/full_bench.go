package main

import (
	"fmt"
	"crypto/sha256"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

type Circuit struct {
	Res frontend.Variable `gnark:",public"`
	A, B frontend.Variable `gnark:"secret"`
}

func (c *Circuit) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(c.A, c.B), c.Res)
	return nil
}

type PublicCircuit struct {
	Res frontend.Variable `gnark:",public"`
}

func (c *PublicCircuit) Define(api frontend.API) error {
	return nil
}

// Simple Merkle tree using SHA256 - handles odd counts
func buildMerkleTree(leaves [][]byte) []byte {
	if len(leaves) == 0 {
		return nil
	}
	current := make([][]byte, len(leaves))
	copy(current, leaves)
	
	for len(current) > 1 {
		var next [][]byte
		for i := 0; i < len(current); i += 2 {
			left := current[i]
			right := left // duplicate last if odd
			if i+1 < len(current) {
				right = current[i+1]
			}
			combined := append(left, right...)
			h := sha256.Sum256(combined)
			next = append(next, h[:])
		}
		current = next
	}
	return current[0]
}

func main() {
	fieldMod := ecc.BN254.ScalarField()
	
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         FULL SYSTEM: Merkle + ZK Prove + Verify              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	
	csFull, _ := frontend.Compile(fieldMod, r1cs.NewBuilder, &Circuit{})
	pk, vk, _ := groth16.Setup(csFull)
	
	fmt.Printf("Circuit: %d constraints\n\n", csFull.GetNbConstraints())
	
	// Full system benchmark
	for _, n := range []int{10, 50, 100, 500, 1000, 5000} {
		totalStart := time.Now()
		
		// Layer 1: Build Merkle tree from inputs
		startMerkle := time.Now()
		var leaves [][]byte
		for i := 0; i < n; i++ {
			h := sha256.Sum256([]byte(fmt.Sprintf("tx-%d", i)))
			leaves = append(leaves, h[:])
		}
		root := buildMerkleTree(leaves)
		merkleMs := time.Since(startMerkle).Milliseconds()
		_ = root
		
		// Layer 2: ZK Prove each transaction
		startProve := time.Now()
		var proofs []interface{}
		for i := 0; i < n; i++ {
			a := int64(i + 1)
			res := a * a
			w := &Circuit{Res: res, A: a, B: a}
			wt, _ := frontend.NewWitness(w, fieldMod)
			p, _ := groth16.Prove(csFull, pk, wt)
			proofs = append(proofs, p)
		}
		proveMs := time.Since(startProve).Milliseconds()
		
		// Layer 3: Verify all proofs
		startVerify := time.Now()
		for i := 0; i < n; i++ {
			a := int64(i + 1)
			res := a * a
			pf := &PublicCircuit{Res: res}
			pwt, _ := frontend.NewWitness(pf, fieldMod)
			groth16.Verify(proofs[i].(groth16.Proof), vk, pwt)
		}
		verifyMs := time.Since(startVerify).Milliseconds()
		
		totalMs := time.Since(totalStart).Milliseconds()
		tps := n * 1000 / int(totalMs)
		
		fmt.Printf("%5d txs | Merkle: %4dms | Prove: %4dms | Verify: %4dms | Total: %5dms = %d TPS\n", 
			n, merkleMs, proveMs, verifyMs, totalMs, tps)
	}
}
