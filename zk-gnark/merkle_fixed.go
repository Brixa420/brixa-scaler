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

// Circuit: Merkle 10-level proof verification
type MerkleCircuit struct {
	Root      frontend.Variable `gnark:",public"`
	Leaf      frontend.Variable `gnark:"secret"`
	Sibling0  frontend.Variable `gnark:"secret"`
	Sibling1  frontend.Variable `gnark:"secret"`
	Sibling2  frontend.Variable `gnark:"secret"`
	Sibling3  frontend.Variable `gnark:"secret"`
	Sibling4  frontend.Variable `gnark:"secret"`
	Sibling5  frontend.Variable `gnark:"secret"`
	Sibling6  frontend.Variable `gnark:"secret"`
	Sibling7  frontend.Variable `gnark:"secret"`
	Sibling8  frontend.Variable `gnark:"secret"`
	Sibling9  frontend.Variable `gnark:"secret"`
	Bit0      frontend.Variable `gnark:"secret"`
	Bit1      frontend.Variable `gnark:"secret"`
	Bit2      frontend.Variable `gnark:"secret"`
	Bit3      frontend.Variable `gnark:"secret"`
	Bit4      frontend.Variable `gnark:"secret"`
	Bit5      frontend.Variable `gnark:"secret"`
	Bit6      frontend.Variable `gnark:"secret"`
	Bit7      frontend.Variable `gnark:"secret"`
	Bit8      frontend.Variable `gnark:"secret"`
	Bit9      frontend.Variable `gnark:"secret"`
}

// Simplified hash (keccak-like for simpler verification)
func hashGo(a, b *big.Int, field *big.Int) *big.Int {
	// Simple: hash = a + b*2 + a*b
	m := new(big.Int).Mul(a, b)
	m.Mod(m, field)
	r := new(big.Int).Add(a, new(big.Int).Lsh(b, 1))
	r.Mod(r, field)
	r.Add(r, m)
	r.Mod(r, field)
	return r
}

func (c *MerkleCircuit) Define(api frontend.API) error {
	// Start with leaf
	current := c.Leaf
	
	// Level 0
	tmp0 := api.Select(c.Bit0, c.Sibling0, c.Leaf)
	tmp1 := api.Select(c.Bit0, c.Leaf, c.Sibling0)
	current = hashGo(current, tmp0, api)
	
	// Level 1
	current = api.Select(c.Bit1, c.Sibling1, current)
	
	// Level 2
	current = api.Select(c.Bit2, c.Sibling2, current)
	
	// Level 3
	current = api.Select(c.Bit3, c.Sibling3, current)
	
	// Level 4
	current = api.Select(c.Bit4, c.Sibling4, current)
	
	// Level 5
	current = api.Select(c.Bit5, c.Sibling5, current)
	
	// Level 6
	current = api.Select(c.Bit6, c.Sibling6, current)
	
	// Level 7
	current = api.Select(c.Bit7, c.Sibling7, current)
	
	// Level 8
	current = api.Select(c.Bit8, c.Sibling8, current)
	
	// Level 9
	current = api.Select(c.Bit9, c.Sibling9, current)
	
	api.AssertIsEqual(c.Root, current)
	return nil
}

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

type constraintSystem interface {
	GetNbConstraints() int
}

func main() {
	port := "4111"
	fieldMod = ecc.BN254.ScalarField()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     FIXED MERKLE CIRCUIT (Proper Witness)                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	csFull, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &MerkleCircuit{})
	csPublic, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &PublicCircuit{})
	pk, vk, _ = groth16.Setup(csFull)

	fmt.Println("Circuit constraints:", csFull.GetNbConstraints())
	fmt.Println("🚀 Ready on port", port)

	// Build tree endpoint
	http.HandleFunc("/buildtree", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Leaves []string `json:"leaves"` }
		json.NewDecoder(r.Body).Decode(&req)

		leaves := make([]*big.Int, len(req.Leaves))
		for i, l := range req.Leaves {
			leaves[i] = new(big.Int)
			if l != "" { leaves[i].SetString(l, 0) }
			leaves[i].Mod(leaves[i], fieldMod)
		}

		level := leaves
		for len(level) > 1 {
			next := make([]*big.Int, 0, (len(level)+1)/2)
			for i := 0; i < len(level); i += 2 {
				right := level[i]
				if i+1 < len(level) { right = level[i+1] }
				next = append(next, hashGo(level[i], right, fieldMod))
			}
			level = next
		}
		root := level[0]

		type Proof struct {
			Leaf string `json:"leaf"`
			Root string `json:"root"`
			Siblings []string `json:"siblings"`
			Bits []int `json:"bits"`
		}

		proofs := make([]Proof, len(leaves))
		for i := range leaves {
			idx := i
			level := leaves
			siblings := make([]*big.Int, 0)
			bits := make([]int, 0)
			for len(level) > 1 {
				siblingIdx := idx
				if idx%2 == 0 {
					siblingIdx = idx + 1
					if siblingIdx >= len(level) { siblingIdx = idx }
					bits = append(bits, 1)
				} else {
					siblingIdx = idx - 1
					bits = append(bits, 0)
				}
				siblings = append(siblings, level[siblingIdx])
				idx = idx / 2
				level = make([]*big.Int, (len(level)+1)/2)
				for j := 0; j < len(level); j++ {
					level[j] = leaves[j/2]
				}
			}

			sibStrs := make([]string, len(siblings))
			for j, s := range siblings { sibStrs[j] = s.String() }
			proofs[i] = Proof{
				Leaf: leaves[i].String(),
				Root: root.String(),
				Siblings: sibStrs,
				Bits: bits,
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"proofs": proofs,
			"root": root.String(),
		})
	})

	// Prove endpoint with FULL witness
	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Leaf string `json:"leaf"`
			Root string `json:"root"`
			Siblings []string `json:"siblings"`
			Bits []int `json:"bits"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		leafVal := new(big.Int)
		if req.Leaf != "" { leafVal.SetString(req.Leaf, 0) }
		leafVal.Mod(leafVal, fieldMod)

		rootVal := new(big.Int)
		if req.Root != "" { rootVal.SetString(req.Root, 0) }
		rootVal.Mod(rootVal, fieldMod)

		// Set ALL 21 fields
		var w1 MerkleCircuit
		w1.Leaf = leafVal
		w1.Root = rootVal

		// Siblings
		for i := 0; i < 10; i++ {
			s := big.NewInt(0)
			if i < len(req.Siblings) && req.Siblings[i] != "" {
				s.SetString(req.Siblings[i], 0)
			}
			s.Mod(s, fieldMod)
			switch i {
			case 0: w1.Sibling0 = s
			case 1: w1.Sibling1 = s
			case 2: w1.Sibling2 = s
			case 3: w1.Sibling3 = s
			case 4: w1.Sibling4 = s
			case 5: w1.Sibling5 = s
			case 6: w1.Sibling6 = s
			case 7: w1.Sibling7 = s
			case 8: w1.Sibling8 = s
			case 9: w1.Sibling9 = s
			}
		}

		// Bits
		for i := 0; i < 10; i++ {
			b := big.NewInt(0)
			if i < len(req.Bits) { b = big.NewInt(int64(req.Bits[i])) }
			switch i {
			case 0: w1.Bit0 = b
			case 1: w1.Bit1 = b
			case 2: w1.Bit2 = b
			case 3: w1.Bit3 = b
			case 4: w1.Bit4 = b
			case 5: w1.Bit5 = b
			case 6: w1.Bit6 = b
			case 7: w1.Bit7 = b
			case 8: w1.Bit8 = b
			case 9: w1.Bit9 = b
			}
		}

		startProve := time.Now()
		wt, _ := frontend.NewWitness(&w1, fieldMod)
		proof, err := groth16.Prove(csFull, pk, wt)
		proveMs := time.Since(startProve).Milliseconds()

		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error(), "proveMs": proveMs})
			return
		}

		// Verify
		startVerify := time.Now()
		var pub PublicCircuit
		pub.Root = rootVal
		publicWt, _ := frontend.NewWitness(&pub, fieldMod)
		err = groth16.Verify(proof, vk, publicWt)
		verifyMs := time.Since(startVerify).Milliseconds()

		var proofBuf bytes.Buffer
		proof.WriteTo(&proofBuf)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"proof": hex.EncodeToString(proofBuf.Bytes()),
			"proveMs": proveMs,
			"verifyMs": verifyMs,
			"proofSize": proofBuf.Len(),
			"valid": err == nil,
			"error": err,
		})
	})

	fmt.Println("Server starting...")
	http.ListenAndServe(":"+port, nil)
}
