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

// SimpleMerkleCircuit with correct hash
type SimpleMerkleCircuit struct {
	Root   frontend.Variable `gnark:",public"`
	Index  frontend.Variable `gnark:",public"`
	Leaf   frontend.Variable `gnark:"secret"`
	Path0  frontend.Variable `gnark:"secret"`
	Path1  frontend.Variable `gnark:"secret"`
	Path2  frontend.Variable `gnark:"secret"`
	Path3  frontend.Variable `gnark:"secret"`
}

func hash2(api frontend.API, left, right frontend.Variable) frontend.Variable {
	prod := api.Mul(left, right)
	return api.Add(api.Add(prod, left), right)
}

func (c *SimpleMerkleCircuit) Define(api frontend.API) error {
	current := c.Leaf
	
	bit0 := api.ToBinary(c.Index, 1)[0]
	path0 := api.Select(bit0, c.Path0, c.Leaf)
	sibling0 := api.Select(bit0, c.Leaf, c.Path0)
	current = hash2(api, path0, sibling0)
	
	bit1 := api.ToBinary(c.Index, 2)[1]
	path1 := api.Select(bit1, c.Path1, current)
	sibling1 := api.Select(bit1, current, c.Path1)
	current = hash2(api, path1, sibling1)
	
	bit2 := api.ToBinary(c.Index, 3)[2]
	path2 := api.Select(bit2, c.Path2, current)
	sibling2 := api.Select(bit2, current, c.Path2)
	current = hash2(api, path2, sibling2)
	
	bit3 := api.ToBinary(c.Index, 4)[3]
	path3 := api.Select(bit3, c.Path3, current)
	sibling3 := api.Select(bit3, current, c.Path3)
	current = hash2(api, path3, sibling3)
	
	api.AssertIsEqual(c.Root, current)
	return nil
}

func hashGo(a, b *big.Int) *big.Int {
	mod := ecc.BN254.ScalarField()
	prod := new(big.Int).Mul(a, b)
	prod.Add(prod, a)
	prod.Add(prod, b)
	return prod.Mod(prod, mod)
}

func buildMerkle(leaves []*big.Int) *big.Int {
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
	return level[0]
}

func getProof(leaves []*big.Int, index int) []*big.Int {
	proof := make([]*big.Int, 0)
	level := leaves
	idx := index
	
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
		proof = append(proof, level[siblingIdx])
		idx = idx / 2
		
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
	return proof
}

var pk groth16.ProvingKey
var vk groth16.VerifyingKey
var r1csCompiled constraint.ConstraintSystem
var fieldMod *big.Int

type Request struct {
	Leaf  string   `json:"leaf"`
	Root  string   `json:"root"`
	Index int      `json:"index"`
	Path  []string `json:"path"`
}

type Response struct {
	Proof       string `json:"proof"`
	PublicInput string `json:"publicInput"`
	ProveMs     int64  `json:"proveMs"`
	VerifyMs    int64  `json:"verifyMs"`
	ProofSize   int    `json:"proofSize"`
	Valid       bool   `json:"valid"`
	Error       string `json:"error,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4111"
	}
	
	fieldMod = ecc.BN254.ScalarField()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         REAL MERKLE CIRCUIT - Working Verification          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	startCompile := time.Now()
	var err error
	r1csCompiled, err = frontend.Compile(fieldMod, r1cs.NewBuilder, &SimpleMerkleCircuit{})
	compileMs := time.Since(startCompile).Milliseconds()
	if err != nil {
		log.Fatal("compile:", err)
	}
	
	fmt.Printf("Circuit compiled in %dms\n", compileMs)
	fmt.Println("  - 4-level Merkle tree (16 leaves)")
	fmt.Println("  - Hash: left*right + left + right")
	fmt.Println("  - Public: root, index")
	fmt.Println("  - Private: leaf + 4 proof elements")
	fmt.Println()

	startSetup := time.Now()
	pk, vk, err = groth16.Setup(r1csCompiled)
	setupMs := time.Since(startSetup).Milliseconds()
	if err != nil {
		log.Fatal("setup:", err)
	}
	
	var pkBuf, vkBuf bytes.Buffer
	pk.WriteTo(&pkBuf)
	vk.WriteTo(&vkBuf)
	
	fmt.Printf("Trusted setup: %dms\n", setupMs)
	fmt.Printf("PK: %d bytes, VK: %d bytes\n\n", pkBuf.Len(), vkBuf.Len())

	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req Request
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

		var witness SimpleMerkleCircuit
		witness.Leaf = leafVal
		witness.Root = rootVal
		witness.Index = big.NewInt(int64(req.Index))

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

		startProve := time.Now()
		wt, _ := frontend.NewWitness(&witness, fieldMod)
		proof, err := groth16.Prove(r1csCompiled, pk, wt)
		proveMs := time.Since(startProve).Milliseconds()

		if err != nil {
			json.NewEncoder(w).Encode(Response{Error: err.Error()})
			return
		}

		startVerify := time.Now()
		publicWt, _ := frontend.NewWitness(&witness, fieldMod)
		err = groth16.Verify(proof, vk, publicWt)
		verifyMs := time.Since(startVerify).Milliseconds()
		valid := err == nil

		var proofBuf bytes.Buffer
		proof.WriteTo(&proofBuf)

		json.NewEncoder(w).Encode(Response{
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
			Proofs []Request `json:"proofs"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		start := time.Now()
		var proveMsTotal, verifyMsTotal int64
		var validCount, totalProofSize int

		for _, p := range req.Proofs {
			leafVal := new(big.Int).SetUint64(0)
			if p.Leaf != "" {
				leafVal.SetString(p.Leaf, 0)
			}
			leafVal.Mod(leafVal, fieldMod)

			rootVal := new(big.Int).SetUint64(0)
			if p.Root != "" {
				rootVal.SetString(p.Root, 0)
			}
			rootVal.Mod(rootVal, fieldMod)

			var witness SimpleMerkleCircuit
			witness.Leaf = leafVal
			witness.Root = rootVal
			witness.Index = big.NewInt(int64(p.Index))

			pathVals := make([]*big.Int, 4)
			for i := 0; i < 4; i++ {
				pv := new(big.Int).SetUint64(0)
				if i < len(p.Path) && p.Path[i] != "" {
					pv.SetString(p.Path[i], 0)
				}
				pv.Mod(pv, fieldMod)
				pathVals[i] = pv
			}
			witness.Path0 = pathVals[0]
			witness.Path1 = pathVals[1]
			witness.Path2 = pathVals[2]
			witness.Path3 = pathVals[3]

			pStart := time.Now()
			wt, _ := frontend.NewWitness(&witness, fieldMod)
			proof, err := groth16.Prove(r1csCompiled, pk, wt)
			proveMsTotal += time.Since(pStart).Milliseconds()

			if err != nil {
				continue
			}

			vStart := time.Now()
			publicWt, _ := frontend.NewWitness(&witness, fieldMod)
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

	http.HandleFunc("/buildtree", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Leaves []string `json:"leaves"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		
		leaves := make([]*big.Int, len(req.Leaves))
		for i, l := range req.Leaves {
			leaves[i] = new(big.Int).SetUint64(0)
			if l != "" {
				leaves[i].SetString(l, 0)
			}
			leaves[i].Mod(leaves[i], fieldMod)
		}
		
		root := buildMerkle(leaves)
		proofs := make([]map[string]interface{}, len(leaves))
		
		for i := range leaves {
			proof := getProof(leaves, i)
			proofs[i] = map[string]interface{}{
				"leaf":  leaves[i].String(),
				"root":  root.String(),
				"index": i,
				"path":  []string{proof[0].String(), proof[1].String(), proof[2].String(), proof[3].String()},
			}
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"root":   root.String(),
			"proofs": proofs,
		})
	})

	fmt.Printf("🚀 Ready on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
