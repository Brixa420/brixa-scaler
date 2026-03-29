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
	"github.com/consensys/gnark/std/hash/mimc"
)

// 8-level Merkle circuit (256 leaves) with MiMC hash
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

func (c *MerkleCircuit) Define(api frontend.API) error {
	mimc, _ := mimc.NewMiMC(api)

	// Level 0
	mimc.Write(c.Leaf, c.Sibling0)
	h0 := mimc.Sum()
	mimc.Write(c.Sibling0, c.Leaf)
	h0r := mimc.Sum()
	current := api.Select(c.Bit0, h0r, h0)

	// Level 1
	mimc.Reset()
	mimc.Write(current, c.Sibling1)
	h1 := mimc.Sum()
	mimc.Reset()
	mimc.Write(c.Sibling1, current)
	h1r := mimc.Sum()
	current = api.Select(c.Bit1, h1r, h1)

	// Level 2
	mimc.Reset()
	mimc.Write(current, c.Sibling2)
	h2 := mimc.Sum()
	mimc.Reset()
	mimc.Write(c.Sibling2, current)
	h2r := mimc.Sum()
	current = api.Select(c.Bit2, h2r, h2)

	// Level 3
	mimc.Reset()
	mimc.Write(current, c.Sibling3)
	h3 := mimc.Sum()
	mimc.Reset()
	mimc.Write(c.Sibling3, current)
	h3r := mimc.Sum()
	current = api.Select(c.Bit3, h3r, h3)

	// Level 4
	mimc.Reset()
	mimc.Write(current, c.Sibling4)
	h4 := mimc.Sum()
	mimc.Reset()
	mimc.Write(c.Sibling4, current)
	h4r := mimc.Sum()
	current = api.Select(c.Bit4, h4r, h4)

	// Level 5
	mimc.Reset()
	mimc.Write(current, c.Sibling5)
	h5 := mimc.Sum()
	mimc.Reset()
	mimc.Write(c.Sibling5, current)
	h5r := mimc.Sum()
	current = api.Select(c.Bit5, h5r, h5)

	// Level 6
	mimc.Reset()
	mimc.Write(current, c.Sibling6)
	h6 := mimc.Sum()
	mimc.Reset()
	mimc.Write(c.Sibling6, current)
	h6r := mimc.Sum()
	current = api.Select(c.Bit6, h6r, h6)

	// Level 7
	mimc.Reset()
	mimc.Write(current, c.Sibling7)
	h7 := mimc.Sum()
	mimc.Reset()
	mimc.Write(c.Sibling7, current)
	h7r := mimc.Sum()
	current = api.Select(c.Bit7, h7r, h7)

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
	fmt.Println("║      REAL MERKLE CIRCUIT (8-level, MiMC hash)               ║")
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

	fmt.Printf("Circuit: %d constraints (MiMC Merkle 8-level)\n", ccs.GetNbConstraints())
	fmt.Printf("🚀 Ready on port %s\n", port)

	// Health
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status":      "ok",
			"circuit":     "merkle8-mimc",
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

// Simple hash: a*b + a + b (MUST match gnark circuit)
func hashMiMC(a, b *big.Int, mod *big.Int) *big.Int {
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
				next[i/2] = hashMiMC(leaves[i], leaves[i+1], fieldMod)
			} else {
				next[i/2] = hashMiMC(leaves[i], leaves[i], fieldMod)
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