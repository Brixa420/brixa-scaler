package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
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

var pk groth16.ProvingKey
var vk groth16.VerifyingKey
var cs interface{ GetNbConstraints() int }
var fieldMod *big.Int

func main() {
	fieldMod = ecc.BN254.ScalarField()
	cs, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &Circuit{})
	pk, vk, _ = groth16.Setup(cs)
	
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     MERKLE PROVER (Prove-Only, No Verify)                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("Circuit: %d constraints\n", cs.GetNbConstraints())
	
	http.HandleFunc("/buildtree", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Leaves []string `json:"leaves"` }
		json.NewDecoder(r.Body).Decode(&req)
		
		n := len(req.Leaves)
		depth := 0
		for i := n; i > 1; i = (i+1)/2 { depth++ }
		
		// Build tree
		level := make([]*big.Int, n)
		for i, l := range req.Leaves {
			level[i] = new(big.Int)
			if l != "" { level[i].SetString(l, 0) }
			level[i].Mod(level[i], fieldMod)
		}
		for len(level) > 1 {
			next := make([]*big.Int, 0, (len(level)+1)/2)
			for i := 0; i < len(level); i += 2 {
				right := level[i]
				if i+1 < len(level) { right = level[i+1] }
				m := new(big.Int).Mul(level[i], right)
				m.Mod(m, fieldMod)
				r := new(big.Int).Add(level[i], right)
				m.Add(m, r)
				m.Mod(m, fieldMod)
				next = append(next, m)
			}
			level = next
		}
		root := level[0]
		
		// Build proofs
		type Proof struct {
			Leaf      string   `json:"leaf"`
			Root      string   `json:"root"`
			Siblings  []string `json:"siblings"`
			Bits      []int    `json:"bits"`
		}
		proofs := make([]Proof, n)
		for i := 0; i < n; i++ {
			idx := i
			sibs := make([]*big.Int, 0)
			bits := make([]int, 0)
			lvl := level
			for j := 0; j < depth && len(lvl) > 1; j++ {
				sibIdx := idx
				if idx%2 == 0 { sibIdx = idx+1; bits = append(bits, 1) }
				else { sibIdx = idx-1; bits = append(bits, 0) }
				if sibIdx >= len(lvl) { sibIdx = idx }
				sibs = append(sibs, lvl[sibIdx])
				idx /= 2
				next := make([]*big.Int, (len(lvl)+1)/2)
				for k := 0; k < len(next); k++ { next[k] = lvl[k/2] }
				lvl = next
			}
			sstr := make([]string, len(sibs))
			for j, s := range sibs { sstr[j] = s.String() }
			proofs[i] = Proof{Leaf: req.Leaves[i], Root: root.String(), Siblings: sstr, Bits: bits}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"proofs": proofs, "root": root.String()})
	})
	
	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Leaf      string   `json:"leaf"`
			Root      string   `json:"root"`
			Siblings  []string `json:"siblings"`
			Bits      []int    `json:"bits"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		
		leaf := big.NewInt(0)
		if req.Leaf != "" { leaf.SetString(req.Leaf, 0) }
		leaf.Mod(leaf, fieldMod)
		
		root := big.NewInt(0)
		if req.Root != "" { root.SetString(req.Root, 0) }
		root.Mod(root, fieldMod)
		
		var w1 Circuit
		w1.Leaf = leaf
		w1.Root = root
		
		for i := 0; i < 10; i++ {
			s := big.NewInt(0)
			if i < len(req.Siblings) && req.Siblings[i] != "" {
				s.SetString(req.Siblings[i], 0)
			}
			s.Mod(s, fieldMod)
			w1.Siblings[i] = s
			if i < len(req.Bits) { w1.Bits[i] = big.NewInt(int64(req.Bits[i])) }
			else { w1.Bits[i] = big.NewInt(0) }
		}
		
		start := time.Now()
		wt, _ := frontend.NewWitness(&w1, fieldMod)
		proof, err := groth16.Prove(cs, pk, wt)
		ms := float64(time.Since(start).Microseconds()) / 1000
		
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		
		var buf bytes.Buffer
		proof.WriteTo(&buf)
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"proof": hex.EncodeToString(buf.Bytes()),
			"proveMs": ms,
			"proofSize": buf.Len(),
		})
	})
	
	fmt.Println("🚀 Server on :4111")
	http.ListenAndServe(":4111", nil)
}
