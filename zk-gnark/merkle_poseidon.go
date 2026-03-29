package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/constraint"
)

// Poseidon-like hash: uses x^5 s-box (like real Poseidon)
func hash(api frontend.API, a, b frontend.Variable) frontend.Variable {
	// sbox: x -> x^5
	sbox := func(x frontend.Variable) frontend.Variable {
		x2 := api.Mul(x, x)
		x4 := api.Mul(x2, x2)
		return api.Mul(x4, x)
	}
	
	sa := sbox(a)
	sb := sbox(b)
	m := api.Mul(sa, sb)
	result := api.Add(m, a)
	result = api.Add(result, b)
	return sbox(result)
}

// 4-level Merkle tree with Poseidon-like hash
type MerkleCircuit struct {
	Root     frontend.Variable `gnark:",public"`
	Leaf     frontend.Variable `gnark:"secret"`
	Sibling0 frontend.Variable `gnark:"secret"`
	Sibling1 frontend.Variable `gnark:"secret"`
	Sibling2 frontend.Variable `gnark:"secret"`
	Sibling3 frontend.Variable `gnark:"secret"`
	Bit0     frontend.Variable `gnark:"secret"`
	Bit1     frontend.Variable `gnark:"secret"`
	Bit2     frontend.Variable `gnark:"secret"`
	Bit3     frontend.Variable `gnark:"secret"`
}

func (c *MerkleCircuit) Define(api frontend.API) error {
	h0a := hash(api, c.Leaf, c.Sibling0)
	h0b := hash(api, c.Sibling0, c.Leaf)
	current0 := api.Select(c.Bit0, h0b, h0a)

	h1a := hash(api, current0, c.Sibling1)
	h1b := hash(api, c.Sibling1, current0)
	current1 := api.Select(c.Bit1, h1b, h1a)

	h2a := hash(api, current1, c.Sibling2)
	h2b := hash(api, c.Sibling2, current1)
	current2 := api.Select(c.Bit2, h2b, h2a)

	h3a := hash(api, current2, c.Sibling3)
	h3b := hash(api, c.Sibling3, current2)
	current3 := api.Select(c.Bit3, h3b, h3a)

	api.AssertIsEqual(c.Root, current3)
	return nil
}

// Public-only circuit for verification
type PublicCircuit struct {
	Root frontend.Variable `gnark:",public"`
}

func (c *PublicCircuit) Define(api frontend.API) error {
	return nil
}

var pk groth16.ProvingKey
var vk groth16.VerifyingKey
var csFull, csPublic constraint.ConstraintSystem
var fieldMod *big.Int

// Hash function outside circuit (Go)
func hashGo(a, b *big.Int) *big.Int {
	fieldMod := ecc.BN254.ScalarField()
	
	sbox := func(x *big.Int) *big.Int {
		x5 := new(big.Int).Exp(x, big.NewInt(5), fieldMod)
		return x5
	}
	
	sa := sbox(a)
	sb := sbox(b)
	m := new(big.Int).Mul(sa, sb)
	m.Mod(m, fieldMod)
	result := new(big.Int).Add(m, a)
	result.Add(result, b)
	result.Mod(result, fieldMod)
	return sbox(result)
}

func main() {
	port := "4111"
	fieldMod = ecc.BN254.ScalarField()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     MERKLE with POSEIDON-LIKE HASH (x^5 s-box)             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	csFull, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &MerkleCircuit{})
	csPublic, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &PublicCircuit{})
	pk, vk, _ = groth16.Setup(csFull)

	fmt.Println("Circuit constraints:", csFull.GetNbConstraints())

	http.HandleFunc("/buildtree", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Leaves []string `json:"leaves"` }
		json.NewDecoder(r.Body).Decode(&req)

		leaves := make([]*big.Int, len(req.Leaves))
		for i, l := range req.Leaves {
			leaves[i] = new(big.Int)
			if l != "" {
				leaves[i].SetString(l, 0)
			}
			leaves[i].Mod(leaves[i], fieldMod)
		}

		// Build tree
		level := leaves
		for len(level) > 1 {
			next := make([]*big.Int, 0, (len(level)+1)/2)
			for i := 0; i < len(level); i += 2 {
				right := level[i]
				if i+1 < len(level) {
					right = level[i+1]
				}
				next = append(next, hashGo(level[i], right))
			}
			level = next
		}
		root := level[0]

		// Generate proofs
		proofs := make([]map[string]interface{}, len(leaves))
		for i := range leaves {
			idx := i
			level := leaves
			siblings := make([]*big.Int, 0)
			for len(level) > 1 {
				siblingIdx := idx
				if idx%2 == 0 {
					siblingIdx = idx + 1
					if siblingIdx >= len(level) {
						siblingIdx = idx
					}
				} else {
					siblingIdx = idx - 1
				}
				siblings = append(siblings, level[siblingIdx])
				idx = idx / 2

				next := make([]*big.Int, 0, (len(level)+1)/2)
				for j := 0; j < len(level); j += 2 {
					right := level[j]
					if j+1 < len(level) {
						right = level[j+1]
					}
					next = append(next, hashGo(level[j], right))
				}
				level = next
			}

			bits := []int{(i >> 0) & 1, (i >> 1) & 1, (i >> 2) & 1, (i >> 3) & 1}

			proofs[i] = map[string]interface{}{
				"leaf":     leaves[i].String(),
				"root":     root.String(),
				"siblings": []string{siblings[0].String(), siblings[1].String(), siblings[2].String(), siblings[3].String()},
				"bits":     bits,
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"root": root.String(), "proofs": proofs})
	})

	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Leaf     string   `json:"leaf"`
			Root     string   `json:"root"`
			Siblings []string `json:"siblings"`
			Bits     []int    `json:"bits"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		leafVal := new(big.Int)
		if req.Leaf != "" {
			leafVal.SetString(req.Leaf, 0)
		}
		leafVal.Mod(leafVal, fieldMod)

		rootVal := new(big.Int)
		if req.Root != "" {
			rootVal.SetString(req.Root, 0)
		}
		rootVal.Mod(rootVal, fieldMod)

		var witness MerkleCircuit
		witness.Leaf = leafVal
		witness.Root = rootVal

		for i := 0; i < 4; i++ {
			p := new(big.Int)
			if i < len(req.Siblings) && req.Siblings[i] != "" {
				p.SetString(req.Siblings[i], 0)
			}
			p.Mod(p, fieldMod)
			switch i {
			case 0:
				witness.Sibling0 = p
			case 1:
				witness.Sibling1 = p
			case 2:
				witness.Sibling2 = p
			case 3:
				witness.Sibling3 = p
			}
		}

		witness.Bit0 = big.NewInt(int64(req.Bits[0]))
		witness.Bit1 = big.NewInt(int64(req.Bits[1]))
		witness.Bit2 = big.NewInt(int64(req.Bits[2]))
		witness.Bit3 = big.NewInt(int64(req.Bits[3]))

		startProve := time.Now()
		wt, _ := frontend.NewWitness(&witness, fieldMod)
		proof, err := groth16.Prove(csFull, pk, wt)
		proveMs := time.Since(startProve).Milliseconds()

		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		startVerify := time.Now()
		var pubWitness PublicCircuit
		pubWitness.Root = rootVal
		publicWt, _ := frontend.NewWitness(&pubWitness, fieldMod)
		err = groth16.Verify(proof, vk, publicWt)
		verifyMs := time.Since(startVerify).Milliseconds()

		var proofBuf bytes.Buffer
		proof.WriteTo(&proofBuf)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"proof":     hex.EncodeToString(proofBuf.Bytes()),
			"proveMs":   proveMs,
			"verifyMs":  verifyMs,
			"proofSize": proofBuf.Len(),
			"valid":     err == nil,
			"error":     err,
		})
	})

	fmt.Printf("🚀 Ready on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
