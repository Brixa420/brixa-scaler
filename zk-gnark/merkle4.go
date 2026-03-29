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

// Fixed 4-level Merkle circuit (16 leaves → 1 root)
// Using simple hash: h(a,b) = a*b + a + b (mod BN254)
type MerkleCircuit struct {
	Root     frontend.Variable `gnark:",public"`
	Leaf     frontend.Variable `gnark:"secret"`
	Sibling0 frontend.Variable `gnark:"secret"` // Level 0 sibling
	Sibling1 frontend.Variable `gnark:"secret"` // Level 1 sibling
	Sibling2 frontend.Variable `gnark:"secret"` // Level 2 sibling
	Sibling3 frontend.Variable `gnark:"secret"` // Level 3 sibling
	Bit0     frontend.Variable `gnark:"secret"` // Path bit for level 0
	Bit1     frontend.Variable `gnark:"secret"` // Path bit for level 1
	Bit2     frontend.Variable `gnark:"secret"` // Path bit for level 2
	Bit3     frontend.Variable `gnark:"secret"` // Path bit for level 3
}

func hash(api frontend.API, a, b frontend.Variable) frontend.Variable {
	// Simple hash: a*b + a + b
	prod := api.Mul(a, b)
	sum := api.Add(a, b)
	return api.Add(prod, sum)
}

func (c *MerkleCircuit) Define(api frontend.API) error {
	// Level 0: hash(Leaf, Sibling0) or hash(Sibling0, Leaf) based on Bit0
	h0a := hash(api, c.Leaf, c.Sibling0)
	h0b := hash(api, c.Sibling0, c.Leaf)
	current := api.Select(c.Bit0, h0b, h0a)

	// Level 1: hash(current, Sibling1) or hash(Sibling1, current) based on Bit1
	h1a := hash(api, current, c.Sibling1)
	h1b := hash(api, c.Sibling1, current)
	current = api.Select(c.Bit1, h1b, h1a)

	// Level 2: hash(current, Sibling2) or hash(Sibling2, current) based on Bit2
	h2a := hash(api, current, c.Sibling2)
	h2b := hash(api, c.Sibling2, current)
	current = api.Select(c.Bit2, h2b, h2a)

	// Level 3: hash(current, Sibling3) or hash(Sibling3, current) based on Bit3
	h3a := hash(api, current, c.Sibling3)
	h3b := hash(api, c.Sibling3, current)
	current = api.Select(c.Bit3, h3b, h3a)

	// Assert root matches
	api.AssertIsEqual(c.Root, current)
	return nil
}

var pk groth16.ProvingKey
var vk groth16.VerifyingKey
var ccs constraint.ConstraintSystem
var fieldMod *big.Int

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4120"
	}
	fieldMod = ecc.BN254.ScalarField()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         MERKLE VERIFICATION CIRCUIT (4-level, Fixed)        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	// Compile circuit
	var err error
	ccs, err = frontend.Compile(fieldMod, r1cs.NewBuilder, &MerkleCircuit{})
	if err != nil {
		fmt.Println("Compile error:", err)
		return
	}

	// Trusted setup
	pk, vk, err = groth16.Setup(ccs)
	if err != nil {
		fmt.Println("Setup error:", err)
		return
	}

	fmt.Printf("Circuit compiled: %d constraints\n", ccs.GetNbConstraints())
	fmt.Printf("🚀 Ready on port %s\n", port)

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "ok",
			"circuit":   "merkle4",
			"constraints": fmt.Sprintf("%d", ccs.GetNbConstraints()),
		})
	})

	// Build Merkle tree and generate proofs
	http.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Leaves []string `json:"leaves"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		// Build tree (simple hash)
		tree := buildMerkleTree(req.Leaves, fieldMod)

		// Generate proofs for each leaf
		proofs := []map[string]interface{}{}
		for i := range req.Leaves {
			leaf, _ := new(big.Int).SetString(req.Leaves[i], 10)
			siblings, bits := getMerkleProof(req.Leaves, i, fieldMod)
			root := tree[len(tree)-1]

			proofs = append(proofs, map[string]interface{}{
				"index":     i,
				"leaf":      leaf.String(),
				"root":      root.String(),
				"siblings":  siblings,
				"bits":      bits,
			})
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"root":   tree[len(tree)-1].String(),
			"proofs": proofs,
		})
	})

	// Prove single leaf verification
	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Leaf     string   `json:"leaf"`
			Root     string   `json:"root"`
			Siblings []string `json:"siblings"`
			Bits     []int    `json:"bits"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		leafVal, _ := new(big.Int).SetString(req.Leaf, 10)
		rootVal, _ := new(big.Int).SetString(req.Root, 10)

		// Create witness
		witness := &MerkleCircuit{
			Root:     rootVal,
			Leaf:     leafVal,
			Sibling0: mustParse(req.Siblings[0]),
			Sibling1: mustParse(req.Siblings[1]),
			Sibling2: mustParse(req.Siblings[2]),
			Sibling3: mustParse(req.Siblings[3]),
			Bit0:     req.Bits[0],
			Bit1:     req.Bits[1],
			Bit2:     req.Bits[2],
			Bit3:     req.Bits[3],
		}

		startProve := time.Now()
		wt, _ := frontend.NewWitness(witness, fieldMod)
		proof, err := groth16.Prove(ccs, pk, wt)
		proveMs := time.Since(startProve).Milliseconds()
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error(), "proveMs": proveMs})
			return
		}

		// Verify
		startVerify := time.Now()
		publicWt, _ := frontend.NewWitness(&MerkleCircuit{Root: rootVal}, fieldMod, frontend.PublicOnly())
		err = groth16.Verify(proof, vk, publicWt)
		verifyMs := time.Since(startVerify).Milliseconds()

		var proofBuf bytes.Buffer
		proof.WriteTo(&proofBuf)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"proof":      hex.EncodeToString(proofBuf.Bytes()),
			"proveMs":    proveMs,
			"verifyMs":   verifyMs,
			"proofSize":  proofBuf.Len(),
			"valid":     err == nil,
			"constraints": ccs.GetNbConstraints(),
		})
	})

	// Benchmark endpoint
	http.HandleFunc("/benchmark", func(w http.ResponseWriter, r *http.Request) {
		results := []map[string]interface{}{}

		// Only test with 16 leaves (exactly 4 levels)
		numLeaves := 16

		// Build tree
		leaves := make([]string, numLeaves)
		for i := 0; i < numLeaves; i++ {
			leaves[i] = fmt.Sprintf("%d", i+1)
		}
		tree := buildMerkleTree(leaves, fieldMod)
		root := tree[len(tree)-1]

		// Prove for leaf 0
		leafVal := big.NewInt(1)
		siblings, bits := getMerkleProof(leaves, 0, fieldMod)

		witness := &MerkleCircuit{
			Root:     root,
			Leaf:     leafVal,
			Sibling0: mustParse(siblings[0]),
			Sibling1: mustParse(siblings[1]),
			Sibling2: mustParse(siblings[2]),
			Sibling3: mustParse(siblings[3]),
			Bit0:     bits[0],
			Bit1:     bits[1],
			Bit2:     bits[2],
			Bit3:     bits[3],
		}

		start := time.Now()
		wt, _ := frontend.NewWitness(witness, fieldMod)
		proof, _ := groth16.Prove(ccs, pk, wt)
		publicWt, _ := frontend.NewWitness(&MerkleCircuit{Root: root}, fieldMod, frontend.PublicOnly())
		err := groth16.Verify(proof, vk, publicWt)
		ms := time.Since(start).Milliseconds()

		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		results = append(results, map[string]interface{}{
			"leaves":      numLeaves,
			"timeMs":      ms,
			"proofsPerSec": 1000 / ms,
		})

		json.NewEncoder(w).Encode(results)
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Simple hash function in Go
func hashGo(a, b *big.Int, mod *big.Int) *big.Int {
	prod := new(big.Int).Mul(a, b)
	sum := new(big.Int).Add(a, b)
	result := new(big.Int).Add(prod, sum)
	return result.Mod(result, mod)
}

func buildMerkleTree(leaves []string, mod *big.Int) []*big.Int {
	nodes := make([]*big.Int, len(leaves))
	for i, l := range leaves {
		nodes[i], _ = new(big.Int).SetString(l, 10)
	}

	for len(nodes) > 1 {
		next := make([]*big.Int, (len(nodes)+1)/2)
		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				next[i/2] = hashGo(nodes[i], nodes[i+1], mod)
			} else {
				next[i/2] = hashGo(nodes[i], nodes[i], mod)
			}
		}
		nodes = next
	}
	return nodes
}

func getMerkleProof(leaves []string, index int, mod *big.Int) ([]string, []int) {
	// Build tree with full levels (power of 2)
	nLeaves := 1
	for nLeaves < len(leaves) {
		nLeaves *= 2
	}

	// Pad leaves
	paddedLeaves := make([]*big.Int, nLeaves)
	for i := 0; i < nLeaves; i++ {
		if i < len(leaves) {
			paddedLeaves[i], _ = new(big.Int).SetString(leaves[i], 10)
		} else {
			paddedLeaves[i] = big.NewInt(0)
		}
	}

	// Build tree level by level
	tree := [][]*big.Int{paddedLeaves}
	for len(tree[len(tree)-1]) > 1 {
		level := tree[len(tree)-1]
		next := make([]*big.Int, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			next[i/2] = hashGo(level[i], level[i+1], mod)
		}
		tree = append(tree, next)
	}

	depth := len(tree) - 1
	var siblings []string
	var bits []int

	currentIdx := index
	for level := 0; level < depth; level++ {
		siblingIdx := currentIdx ^ 1
		if siblingIdx < len(tree[level]) {
			siblings = append(siblings, tree[level][siblingIdx].String())
		} else {
			siblings = append(siblings, "0")
		}
		bits = append(bits, currentIdx&1)
		currentIdx /= 2
	}

	// Pad to exactly 4 levels for the circuit
	for len(siblings) < 4 {
		siblings = append(siblings, "0")
		bits = append(bits, 0)
	}

	return siblings, bits
}

func mustParse(s string) *big.Int {
	v, _ := new(big.Int).SetString(s, 10)
	return v
}