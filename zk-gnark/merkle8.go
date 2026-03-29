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

// 8-level Merkle circuit with simple hash: h(a,b) = a*b + a + b (mod BN254)
type MerkleCircuit struct {
	Root     frontend.Variable `gnark:",public"`
	Leaf     frontend.Variable `gnark:"secret"`
	Sibling0 frontend.Variable `gnark:"secret"`
	Sibling1 frontend.Variable `gnark:"secret"`
	Sibling2 frontend.Variable `gnark:"secret"`
	Sibling3 frontend.Variable `gnark:"secret"`
	Sibling4 frontend.Variable `gnark:"secret"`
	Sibling5 frontend.Variable `gnark:"secret"`
	Sibling6 frontend.Variable `gnark:"secret"`
	Sibling7 frontend.Variable `gnark:"secret"`
	Bit0     frontend.Variable `gnark:"secret"`
	Bit1     frontend.Variable `gnark:"secret"`
	Bit2     frontend.Variable `gnark:"secret"`
	Bit3     frontend.Variable `gnark:"secret"`
	Bit4     frontend.Variable `gnark:"secret"`
	Bit5     frontend.Variable `gnark:"secret"`
	Bit6     frontend.Variable `gnark:"secret"`
	Bit7     frontend.Variable `gnark:"secret"`
}

func hash(api frontend.API, a, b frontend.Variable) frontend.Variable {
	prod := api.Mul(a, b)
	sum := api.Add(a, b)
	return api.Add(prod, sum)
}

func (c *MerkleCircuit) Define(api frontend.API) error {
	// Level 0
	h0a := hash(api, c.Leaf, c.Sibling0)
	h0b := hash(api, c.Sibling0, c.Leaf)
	current := api.Select(c.Bit0, h0b, h0a)

	// Level 1
	h1a := hash(api, current, c.Sibling1)
	h1b := hash(api, c.Sibling1, current)
	current = api.Select(c.Bit1, h1b, h1a)

	// Level 2
	h2a := hash(api, current, c.Sibling2)
	h2b := hash(api, c.Sibling2, current)
	current = api.Select(c.Bit2, h2b, h2a)

	// Level 3
	h3a := hash(api, current, c.Sibling3)
	h3b := hash(api, c.Sibling3, current)
	current = api.Select(c.Bit3, h3b, h3a)

	// Level 4
	h4a := hash(api, current, c.Sibling4)
	h4b := hash(api, c.Sibling4, current)
	current = api.Select(c.Bit4, h4b, h4a)

	// Level 5
	h5a := hash(api, current, c.Sibling5)
	h5b := hash(api, c.Sibling5, current)
	current = api.Select(c.Bit5, h5b, h5a)

	// Level 6
	h6a := hash(api, current, c.Sibling6)
	h6b := hash(api, c.Sibling6, current)
	current = api.Select(c.Bit6, h6b, h6a)

	// Level 7
	h7a := hash(api, current, c.Sibling7)
	h7b := hash(api, c.Sibling7, current)
	current = api.Select(c.Bit7, h7b, h7a)

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
		port = "4130"
	}
	fieldMod = ecc.BN254.ScalarField()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║      REAL MERKLE CIRCUIT (8-level, 256 leaves)              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	var err error
	ccs, err = frontend.Compile(fieldMod, r1cs.NewBuilder, &MerkleCircuit{})
	if err != nil {
		fmt.Println("Compile error:", err)
		return
	}

	pk, vk, err = groth16.Setup(ccs)
	if err != nil {
		fmt.Println("Setup error:", err)
		return
	}

	fmt.Printf("Circuit: %d constraints (8-level Merkle)\n", ccs.GetNbConstraints())
	fmt.Printf("🚀 Ready on port %s\n", port)

	// Health
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status":      "ok",
			"circuit":     "merkle8",
			"constraints": fmt.Sprintf("%d", ccs.GetNbConstraints()),
		})
	})

	// Build tree + proofs
	http.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Leaves []string `json:"leaves"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		// Pad to power of 2
		n := 1
		for n < len(req.Leaves) {
			n *= 2
		}

		leaves := make([]*big.Int, n)
		for i := 0; i < n; i++ {
			if i < len(req.Leaves) {
				leaves[i], _ = new(big.Int).SetString(req.Leaves[i], 10)
			} else {
				leaves[i] = big.NewInt(0)
			}
		}

		tree := buildMerkleTree(leaves)
		root := tree[len(tree)-1][0]

		proofs := []map[string]interface{}{}
		for i := 0; i < len(req.Leaves); i++ {
			siblings, bits := getProof(leaves, i)
			proofs = append(proofs, map[string]interface{}{
				"index":     i,
				"leaf":      leaves[i].String(),
				"root":      root.String(),
				"siblings":  siblings,
				"bits":      bits,
			})
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"root":   root.String(),
			"proofs": proofs,
		})
	})

	// Prove
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

		witness := &MerkleCircuit{
			Root:     rootVal,
			Leaf:     leafVal,
			Sibling0: mustParse(req.Siblings[0]),
			Sibling1: mustParse(req.Siblings[1]),
			Sibling2: mustParse(req.Siblings[2]),
			Sibling3: mustParse(req.Siblings[3]),
			Sibling4: mustParse(req.Siblings[4]),
			Sibling5: mustParse(req.Siblings[5]),
			Sibling6: mustParse(req.Siblings[6]),
			Sibling7: mustParse(req.Siblings[7]),
			Bit0:     req.Bits[0],
			Bit1:     req.Bits[1],
			Bit2:     req.Bits[2],
			Bit3:     req.Bits[3],
			Bit4:     req.Bits[4],
			Bit5:     req.Bits[5],
			Bit6:     req.Bits[6],
			Bit7:     req.Bits[7],
		}

		startProve := time.Now()
		wt, _ := frontend.NewWitness(witness, fieldMod)
		proof, err := groth16.Prove(ccs, pk, wt)
		proveMs := time.Since(startProve).Milliseconds()
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error(), "proveMs": proveMs})
			return
		}

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

	// Benchmark
	http.HandleFunc("/benchmark", func(w http.ResponseWriter, r *http.Request) {
		// Build 256-leaf tree
		n := 256
		leaves := make([]*big.Int, n)
		for i := 0; i < n; i++ {
			leaves[i] = big.NewInt(int64(i + 1))
		}

		tree := buildMerkleTree(leaves)
		root := tree[len(tree)-1][0]

		siblings, bits := getProof(leaves, 0)

		witness := &MerkleCircuit{
			Root:     root,
			Leaf:     leaves[0],
			Sibling0: mustParse(siblings[0]),
			Sibling1: mustParse(siblings[1]),
			Sibling2: mustParse(siblings[2]),
			Sibling3: mustParse(siblings[3]),
			Sibling4: mustParse(siblings[4]),
			Sibling5: mustParse(siblings[5]),
			Sibling6: mustParse(siblings[6]),
			Sibling7: mustParse(siblings[7]),
			Bit0:     bits[0],
			Bit1:     bits[1],
			Bit2:     bits[2],
			Bit3:     bits[3],
			Bit4:     bits[4],
			Bit5:     bits[5],
			Bit6:     bits[6],
			Bit7:     bits[7],
		}

		start := time.Now()
		wt, _ := frontend.NewWitness(witness, fieldMod)
		proof, _ := groth16.Prove(ccs, pk, wt)
		publicWt, _ := frontend.NewWitness(&MerkleCircuit{Root: root}, fieldMod, frontend.PublicOnly())
		err := groth16.Verify(proof, vk, publicWt)
		ms := time.Since(start).Milliseconds()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"leaves":       n,
			"timeMs":       ms,
			"proofsPerSec": 1000 / ms,
			"valid":        err == nil,
		})
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Simple hash: a*b + a + b (MUST match circuit)
func hashGo(a, b *big.Int, mod *big.Int) *big.Int {
	prod := new(big.Int).Mul(a, b)
	sum := new(big.Int).Add(a, b)
	result := new(big.Int).Add(prod, sum)
	return result.Mod(result, mod)
}

func buildMerkleTree(leaves []*big.Int) [][]*big.Int {
	tree := [][]*big.Int{leaves}
	for len(leaves) > 1 {
		next := make([]*big.Int, (len(leaves)+1)/2)
		for i := 0; i < len(leaves); i += 2 {
			if i+1 < len(leaves) {
				next[i/2] = hashGo(leaves[i], leaves[i+1], fieldMod)
			} else {
				next[i/2] = hashGo(leaves[i], leaves[i], fieldMod)
			}
		}
		tree = append(tree, next)
		leaves = next
	}
	return tree
}

func getProof(leaves []*big.Int, idx int) ([]string, []int) {
	tree := buildMerkleTree(leaves)
	depth := len(tree) - 1

	var siblings []string
	var bits []int
	currentIdx := idx

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

	// Pad to 8
	for len(siblings) < 8 {
		siblings = append(siblings, "0")
		bits = append(bits, 0)
	}

	return siblings, bits
}

func mustParse(s string) *big.Int {
	v, _ := new(big.Int).SetString(s, 10)
	return v
}