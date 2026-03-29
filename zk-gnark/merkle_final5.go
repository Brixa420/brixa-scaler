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

// Public-only circuit for verification
type PublicCircuit struct {
	Root frontend.Variable `gnark:",public"`
}

func (c *PublicCircuit) Define(api frontend.API) error {
	return nil
}

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

func hash(api frontend.API, a, b frontend.Variable) frontend.Variable {
	return api.Add(api.Add(api.Mul(a, b), a), b)
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

var pk groth16.ProvingKey
var vk groth16.VerifyingKey
var r1csFull constraint.ConstraintSystem
var r1csPublic constraint.ConstraintSystem
var fieldMod *big.Int

func hashGo(a, b *big.Int) *big.Int {
	m := new(big.Int).Mul(a, b)
	m.Add(m, a)
	m.Add(m, b)
	return m.Mod(m, fieldMod)
}

func main() {
	port := "4111"
	fieldMod = ecc.BN254.ScalarField()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         MERKLE CIRCUIT - Public Witness Fix                 ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	r1csFull, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &MerkleCircuit{})
	r1csPublic, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &PublicCircuit{})
	fmt.Println("Circuit compiled (full + public)")
	
	pk, vk, _ = groth16.Setup(r1csFull)

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
				"leaf":  leaves[i].String(),
				"root":  root.String(),
				"siblings": []string{
					siblings[0].String(), siblings[1].String(), 
					siblings[2].String(), siblings[3].String(),
				},
				"bits":  bits,
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

		sibVals := make([]*big.Int, 4)
		for i := 0; i < 4; i++ {
			p := new(big.Int)
			if i < len(req.Siblings) && req.Siblings[i] != "" {
				p.SetString(req.Siblings[i], 0)
			}
			sibVals[i] = p.Mod(p, fieldMod)
		}
		witness.Sibling0 = sibVals[0]
		witness.Sibling1 = sibVals[1]
		witness.Sibling2 = sibVals[2]
		witness.Sibling3 = sibVals[3]

		witness.Bit0 = big.NewInt(int64(req.Bits[0]))
		witness.Bit1 = big.NewInt(int64(req.Bits[1]))
		witness.Bit2 = big.NewInt(int64(req.Bits[2]))
		witness.Bit3 = big.NewInt(int64(req.Bits[3]))

		startProve := time.Now()
		wt, _ := frontend.NewWitness(&witness, fieldMod)
		proof, err := groth16.Prove(r1csFull, pk, wt)
		proveMs := time.Since(startProve).Milliseconds()

		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		startVerify := time.Now()
		
		// Use PUBLIC CIRCUIT for verification witness
		var publicWitness PublicCircuit
		publicWitness.Root = rootVal
		publicWt, _ := frontend.NewWitness(&publicWitness, fieldMod)
		
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
