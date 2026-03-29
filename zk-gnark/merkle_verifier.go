package main

import (
	"bytes"
	"crypto/sha256"
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
)

// MerkleTreeCircuit verifies a Merkle proof for transactions
type MerkleTreeCircuit struct {
	Root  frontend.Variable `gnark:",public"` // Expected Merkle root
	Index frontend.Variable `gnark:",public"` // Leaf index
	
	Leaf  frontend.Variable `gnark:"secret"` // The leaf being proven
	Path0 frontend.Variable `gnark:"secret"` // Proof path level 0
	Path1 frontend.Variable `gnark:"secret"` // Proof path level 1
	Path2 frontend.Variable `gnark:"secret"` // Proof path level 2
	Path3 frontend.Variable `gnark:"secret"` // Proof path level 3
}

// Simple hash function for Merkle tree
func hash2(api frontend.API, left, right frontend.Variable) frontend.Variable {
	combined := api.Mul(left, right)
	return api.Add(combined, api.Add(left, right))
}

func (c *MerkleTreeCircuit) Define(api frontend.API) error {
	current := c.Leaf
	
	// Level 0
	bit0 := api.ToBinary(c.Index, 1)[0]
	path0 := api.Select(bit0, c.Path0, c.Leaf)
	sibling0 := api.Select(bit0, c.Leaf, c.Path0)
	current = hash2(api, path0, sibling0)
	
	// Level 1
	bit1 := api.ToBinary(c.Index, 2)[1]
	path1 := api.Select(bit1, c.Path1, current)
	sibling1 := api.Select(bit1, current, c.Path1)
	current = hash2(api, path1, sibling1)
	
	// Level 2
	bit2 := api.ToBinary(c.Index, 3)[2]
	path2 := api.Select(bit2, c.Path2, current)
	sibling2 := api.Select(bit2, current, c.Path2)
	current = hash2(api, path2, sibling2)
	
	// Level 3 (root)
	bit3 := api.ToBinary(c.Index, 4)[3]
	path3 := api.Select(bit3, c.Path3, current)
	sibling3 := api.Select(bit3, current, c.Path3)
	current = hash2(api, path3, sibling3)
	
	api.AssertIsEqual(c.Root, current)
	return nil
}

var pk groth16.ProvingKey
var vk groth16.VerifyingKey
var r1csCompiled interface{}

type MerkleProofRequest struct {
	Leaf  string   `json:"leaf"`
	Root  string   `json:"root"`
	Index int      `json:"index"`
	Path  []string `json:"path"`
}

type ProofResponse struct {
	Proof       string   `json:"proof"`
	PublicInput string   `json:"publicInput"`
	ProveMs     int64    `json:"proveMs"`
	VerifyMs    int64    `json:"verifyMs"`
	ProofSize   int      `json:"proofSize"`
	Valid       bool     `json:"valid"`
	Error       string   `json:"error,omitempty"`
}

func hashStrings(a, b string) string {
	h := sha256.Sum256([]byte(a + b))
	return hex.EncodeToString(h[:])
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4111"
	}

	mod := ecc.BN254.ScalarField()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         MERKLE TREE ZK - Real Transaction Verification      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	startCompile := time.Now()
	r1csCompiled, err := frontend.Compile(mod, r1cs.NewBuilder, &MerkleTreeCircuit{})
	compileMs := time.Since(startCompile).Milliseconds()
	if err != nil {
		log.Fatal("compile:", err)
	}
	
	fmt.Printf("Circuit compiled in %dms\n", compileMs)
	fmt.Println("  - 4-level Merkle tree (16 leaves)")
	fmt.Println("  - Public: root, index")
	fmt.Println("  - Private: leaf + 4 proof elements")
	fmt.Println()

	startSetup := time.Now()
	pk, vk, err = groth16.Setup(r1csCompiled)
	setupMs := time.Since(startSetup).Milliseconds()
	if err != nil {
		log.Fatal("setup:", err)
	}
	
	var pkBuf bytes.Buffer
	pk.WriteTo(&pkBuf)
	var vkBuf bytes.Buffer
	vk.WriteTo(&vkBuf)
	
	fmt.Printf("Trusted setup: %dms\n", setupMs)
	fmt.Printf("PK: %d bytes, VK: %d bytes\n", pkBuf.Len(), vkBuf.Len())
	
	os.WriteFile("merkle_pk.bin", pkBuf.Bytes(), 0644)
	os.WriteFile("merkle_vk.bin", vkBuf.Bytes(), 0644)
	fmt.Println("Keys saved\n")

	fmt.Printf("🚀 MerkleVerifier ready on port %s\n\n", port)

	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req MerkleProofRequest
		json.NewDecoder(r.Body).Decode(&req)

		leafVal := big.NewInt(0)
		if req.Leaf != "" {
			leafVal.SetString(req.Leaf, 16)
		}
		leafVal.Mod(leafVal, mod)

		rootVal := big.NewInt(0)
		if req.Root != "" {
			rootVal.SetString(req.Root, 16)
		}
		rootVal.Mod(rootVal, mod)

		var witness MerkleTreeCircuit
		witness.Leaf = leafVal
		witness.Root = rootVal
		witness.Index = big.NewInt(int64(req.Index))

		pathVals := make([]*big.Int, 4)
		for i := 0; i < 4; i++ {
			p := big.NewInt(0)
			if i < len(req.Path) && req.Path[i] != "" {
				p.SetString(req.Path[i], 16)
			}
			p.Mod(p, mod)
			pathVals[i] = p
		}
		witness.Path0 = pathVals[0]
		witness.Path1 = pathVals[1]
		witness.Path2 = pathVals[2]
		witness.Path3 = pathVals[3]

		startProve := time.Now()
		wt, _ := frontend.NewWitness(&witness, mod)
		proof, err := groth16.Prove(r1csCompiled, pk, wt)
		proveMs := time.Since(startProve).Milliseconds()

		if err != nil {
			json.NewEncoder(w).Encode(ProofResponse{Error: err.Error()})
			return
		}

		startVerify := time.Now()
		publicWt, _ := frontend.NewWitness(&witness, mod)
		err = groth16.Verify(proof, vk, publicWt)
		verifyMs := time.Since(startVerify).Milliseconds()
		valid := err == nil

		var proofBuf bytes.Buffer
		proof.WriteTo(&proofBuf)

		json.NewEncoder(w).Encode(ProofResponse{
			Proof:       hex.EncodeToString(proofBuf.Bytes()),
			PublicInput: fmt.Sprintf("%s:%d", req.Root, req.Index),
			ProveMs:     proveMs,
			VerifyMs:    verifyMs,
			ProofSize:   proofBuf.Len(),
			Valid:       valid,
		})
	})

	http.HandleFunc("/provebatch", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Proofs []MerkleProofRequest `json:"proofs"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		start := time.Now()
		var proveMsTotal, verifyMsTotal int64
		var validCount, totalProofSize int

		for _, p := range req.Proofs {
			leafVal := big.NewInt(0)
			if p.Leaf != "" {
				leafVal.SetString(p.Leaf, 16)
			}
			leafVal.Mod(leafVal, mod)

			rootVal := big.NewInt(0)
			if p.Root != "" {
				rootVal.SetString(p.Root, 16)
			}
			rootVal.Mod(rootVal, mod)

			var witness MerkleTreeCircuit
			witness.Leaf = leafVal
			witness.Root = rootVal
			witness.Index = big.NewInt(int64(p.Index))

			pathVals := make([]*big.Int, 4)
			for i := 0; i < 4; i++ {
				pv := big.NewInt(0)
				if i < len(p.Path) && p.Path[i] != "" {
					pv.SetString(p.Path[i], 16)
				}
				pv.Mod(pv, mod)
				pathVals[i] = pv
			}
			witness.Path0 = pathVals[0]
			witness.Path1 = pathVals[1]
			witness.Path2 = pathVals[2]
			witness.Path3 = pathVals[3]

			pStart := time.Now()
			wt, _ := frontend.NewWitness(&witness, mod)
			proof, err := groth16.Prove(r1csCompiled, pk, wt)
			proveMsTotal += time.Since(pStart).Milliseconds()

			if err != nil {
				continue
			}

			vStart := time.Now()
			publicWt, _ := frontend.NewWitness(&witness, mod)
			err = groth16.Verify(proof, vk, publicWt)
			if err == nil {
				validCount++
			}
			verifyMsTotal += time.Since(vStart).Milliseconds()

			var proofBuf bytes.Buffer
			proof.WriteTo(&proofBuf)
			totalProofSize += proofBuf.Len()
		}

		totalMs := time.Since(start).Milliseconds()
		proofCount := len(req.Proofs)
		if proofCount == 0 {
			proofCount = 1
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":        validCount,
			"totalMs":      totalMs,
			"proveMs":      proveMsTotal,
			"verifyMs":     verifyMsTotal,
			"avgProofSize": totalProofSize / proofCount,
			"avgProveMs":   proveMsTotal / int64(proofCount),
			"avgVerifyMs":  verifyMsTotal / int64(proofCount),
			"throughput":   float64(proofCount) / float64(totalMs) * 1000,
		})
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}