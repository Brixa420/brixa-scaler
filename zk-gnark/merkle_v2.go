package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/constraint"
)

// MerkleCircuit - pass index bits directly
type MerkleCircuit struct {
	Root     frontend.Variable `gnark:",public"`
	Leaf     frontend.Variable `gnark:"secret"`
	Path0    frontend.Variable `gnark:"secret"` // sibling at level 0
	Path1    frontend.Variable `gnark:"secret"` // sibling at level 1
	Path2    frontend.Variable `gnark:"secret"` // sibling at level 2
	Path3    frontend.Variable `gnark:"secret"` // sibling at level 3
	Bit0     frontend.Variable `gnark:"secret"` // bit 0 of index
	Bit1     frontend.Variable `gnark:"secret"` // bit 1 of index
	Bit2     frontend.Variable `gnark:"secret"` // bit 2 of index
	Bit3     frontend.Variable `gnark:"secret"` // bit 3 of index
}

func hash2(api frontend.API, left, right frontend.Variable) frontend.Variable {
	prod := api.Mul(left, right)
	return api.Add(api.Add(prod, left), right)
}

func (c *MerkleCircuit) Define(api frontend.API) error {
	// Level 0: hash(leaf, path0) or hash(path0, leaf)
	current := api.Add(c.Leaf, c.Path0)
	current = api.Add(api.Mul(c.Leaf, c.Path0), current)
	current = api.Select(c.Bit0, api.Add(c.Path0, c.Leaf), current)
	current = api.Select(c.Bit0, api.Add(api.Mul(c.Path0, c.Leaf), current), current)

	// Level 1
	current = api.Add(current, c.Path1)
	current = api.Add(api.Mul(current, c.Path1), current)
	current = api.Select(c.Bit1, api.Add(c.Path1, current), current)
	current = api.Select(c.Bit1, api.Add(api.Mul(c.Path1, current), current), current)

	// Level 2
	current = api.Add(current, c.Path2)
	current = api.Add(api.Mul(current, c.Path2), current)
	current = api.Select(c.Bit2, api.Add(c.Path2, current), current)
	current = api.Select(c.Bit2, api.Add(api.Mul(c.Path2, current), current), current)

	// Level 3 (root)
	current = api.Add(current, c.Path3)
	current = api.Add(api.Mul(current, c.Path3), current)
	current = api.Select(c.Bit3, api.Add(c.Path3, current), current)
	current = api.Select(c.Bit3, api.Add(api.Mul(c.Path3, current), current), current)

	api.AssertIsEqual(c.Root, current)
	return nil
}

var pk groth16.ProvingKey
var vk groth16.VerifyingKey
var r1csCompiled constraint.ConstraintSystem
var fieldMod *big.Int

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4111"
	}
	
	fieldMod = ecc.BN254.ScalarField()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         MERKLE CIRCUIT V2 - Direct Bit Control             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	r1csCompiled, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &MerkleCircuit{})
	fmt.Println("Circuit: 4-level Merkle, 9 private inputs (leaf + 4 path + 4 bits)")
	
	pk, vk, _ = groth16.Setup(r1csCompiled)
	fmt.Println("PK:", func() int { var b bytes.Buffer; pk.WriteTo(&b); return b.Len() }(), "bytes")
	
	// Build tree endpoint
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
		
		// Build tree with a*b+a+b hash
		hash := func(a, b *big.Int) *big.Int {
			m := new(big.Int).Mul(a, b)
			m.Add(m, a)
			m.Add(m, b)
			return m.Mod(m, fieldMod)
		}
		
		level := leaves
		for len(level) > 1 {
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
		root := level[0]
		
		// Generate proofs
		proofs := make([]map[string]interface{}, len(leaves))
		for i := range leaves {
			// Build proof path
			idx := i
			level := leaves
			path := make([]*big.Int, 0)
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
				path = append(path, level[siblingIdx])
				idx = idx / 2
				next := make([]*big.Int, 0, (len(level)+1)/2)
				for j := 0; j < len(level); j += 2 {
					right := level[j]
					if j+1 < len(level) {
						right = level[j+1]
					}
					next = append(next, hash(level[j], right))
				}
				level = next
			}
			
			// Get index bits
			bits := []int{(i >> 0) & 1, (i >> 1) & 1, (i >> 2) & 1, (i >> 3) & 1}
			
			proofs[i] = map[string]interface{}{
				"leaf":  leaves[i].String(),
				"root":  root.String(),
				"path":  []string{path[0].String(), path[1].String(), path[2].String(), path[3].String()},
				"bits":  bits,
			}
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{"root": root.String(), "proofs": proofs})
	})

	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Leaf  string   `json:"leaf"`
			Root  string   `json:"root"`
			Path  []string `json:"path"`
			Bits  []int    `json:"bits"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		leafVal := new(big.Int).SetUint64(0)
		if req.Leaf != "" {
			leafVal.SetString(req.Leaf, 0)
		}
		leafVal.Mod(leafVal, fieldMod)

		rootVal := new(big.Int).SetUint64(0)
		if req.Root != "" {
			rootVal.SetString(req.Root, 0)
		}
		rootVal.Mod(rootVal, fieldMod)

		var witness MerkleCircuit
		witness.Leaf = leafVal
		witness.Root = rootVal

		pathVals := make([]*big.Int, 4)
		for i := 0; i < 4; i++ {
			p := new(big.Int).SetUint64(0)
			if i < len(req.Path) && req.Path[i] != "" {
				p.SetString(req.Path[i], 0)
			}
			p.Mod(p, fieldMod)
			pathVals[i] = p
		}
		witness.Path0 = pathVals[0]
		witness.Path1 = pathVals[1]
		witness.Path2 = pathVals[2]
		witness.Path3 = pathVals[3]

		witness.Bit0 = big.NewInt(int64(req.Bits[0]))
		witness.Bit1 = big.NewInt(int64(req.Bits[1]))
		witness.Bit2 = big.NewInt(int64(req.Bits[2]))
		witness.Bit3 = big.NewInt(int64(req.Bits[3]))

		startProve := time.Now()
		wt, _ := frontend.NewWitness(&witness, fieldMod)
		proof, err := groth16.Prove(r1csCompiled, pk, wt)
		proveMs := time.Since(startProve).Milliseconds()

		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		startVerify := time.Now()
		publicWt, _ := frontend.NewWitness(&witness, fieldMod)
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
		})
	})

	fmt.Printf("🚀 Ready on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
