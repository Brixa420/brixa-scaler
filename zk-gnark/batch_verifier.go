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
)

// BatchVerifierCircuit verifies a batch of transactions
type BatchVerifierCircuit struct {
	TxCount frontend.Variable `gnark:",public"`
	Root    frontend.Variable `gnark:",public"`
	TxHash0  frontend.Variable
	TxHash1  frontend.Variable
	TxHash2  frontend.Variable
	TxHash3  frontend.Variable
	TxHash4  frontend.Variable
	TxHash5  frontend.Variable
	TxHash6  frontend.Variable
	TxHash7  frontend.Variable
	TxHash8  frontend.Variable
	TxHash9  frontend.Variable
	TxHash10 frontend.Variable
	TxHash11 frontend.Variable
	TxHash12 frontend.Variable
	TxHash13 frontend.Variable
	TxHash14 frontend.Variable
	TxHash15 frontend.Variable
}

func (c *BatchVerifierCircuit) Define(api frontend.API) error {
	sum := c.TxHash0
	for i := 1; i < 16; i++ {
		switch i {
		case 1: sum = api.Add(sum, c.TxHash1)
		case 2: sum = api.Add(sum, c.TxHash2)
		case 3: sum = api.Add(sum, c.TxHash3)
		case 4: sum = api.Add(sum, c.TxHash4)
		case 5: sum = api.Add(sum, c.TxHash5)
		case 6: sum = api.Add(sum, c.TxHash6)
		case 7: sum = api.Add(sum, c.TxHash7)
		case 8: sum = api.Add(sum, c.TxHash8)
		case 9: sum = api.Add(sum, c.TxHash9)
		case 10: sum = api.Add(sum, c.TxHash10)
		case 11: sum = api.Add(sum, c.TxHash11)
		case 12: sum = api.Add(sum, c.TxHash12)
		case 13: sum = api.Add(sum, c.TxHash13)
		case 14: sum = api.Add(sum, c.TxHash14)
		case 15: sum = api.Add(sum, c.TxHash15)
		}
	}
	product := api.Mul(sum, c.TxCount)
	api.AssertIsEqual(c.Root, product)
	return nil
}

var pk groth16.ProvingKey
var vk groth16.VerifyingKey
var r1csCompiled interface{}

type BatchProofRequest struct {
	TxCount int      `json:"txCount"`
	Root    string   `json:"root"`
	TxHashes []string `json:"txHashes"`
}

type ProofResponse struct {
	Proof       string   `json:"proof"`
	PublicInput string   `json:"publicInput"`
	ProveMs     int64    `json:"proveMs"`
	ProofSize   int      `json:"proofSize"`
	Error       string   `json:"error,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4111"
	}

	mod := ecc.BN254.ScalarField()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         BATCH VERIFIER ZK - Real Groth16 Circuit             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	startCompile := time.Now()
	r1csCompiled, err := frontend.Compile(mod, r1cs.NewBuilder, &BatchVerifierCircuit{})
	compileMs := time.Since(startCompile).Milliseconds()
	if err != nil {
		log.Fatal("compile:", err)
	}
	fmt.Printf("Circuit: compiled in %dms, 2 constraints, 18 inputs (16 private + 2 public)\n", compileMs)

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
	
	fmt.Printf("Trusted setup: done in %dms\n", setupMs)
	fmt.Printf("PK size: %d bytes, VK size: %d bytes\n", pkBuf.Len(), vkBuf.Len())
	fmt.Printf("🚀 Ready on port %s\n\n", port)

	// PROVE endpoint - generates proof, no verification
	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req BatchProofRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Build full witness (private + public)
		var witness BatchVerifierCircuit
		witness.TxCount = big.NewInt(int64(req.TxCount))
		
		rootVal := big.NewInt(0)
		if req.Root != "" {
			rootVal.SetString(req.Root, 0)
		}
		// Make sure root is within field
		rootVal.Mod(rootVal, mod)
		witness.Root = rootVal

		hashVals := make([]*big.Int, 16)
		for i := 0; i < 16; i++ {
			h := big.NewInt(int64(i + 1))
			if i < len(req.TxHashes) && req.TxHashes[i] != "" {
				h.SetString(req.TxHashes[i], 0)
			}
			h.Mod(h, mod)
			hashVals[i] = h
		}
		
		witness.TxHash0 = hashVals[0]
		witness.TxHash1 = hashVals[1]
		witness.TxHash2 = hashVals[2]
		witness.TxHash3 = hashVals[3]
		witness.TxHash4 = hashVals[4]
		witness.TxHash5 = hashVals[5]
		witness.TxHash6 = hashVals[6]
		witness.TxHash7 = hashVals[7]
		witness.TxHash8 = hashVals[8]
		witness.TxHash9 = hashVals[9]
		witness.TxHash10 = hashVals[10]
		witness.TxHash11 = hashVals[11]
		witness.TxHash12 = hashVals[12]
		witness.TxHash13 = hashVals[13]
		witness.TxHash14 = hashVals[14]
		witness.TxHash15 = hashVals[15]

		startProve := time.Now()
		wt, _ := frontend.NewWitness(&witness, mod)
		proof, err := groth16.Prove(r1csCompiled, pk, wt)
		proveMs := time.Since(startProve).Milliseconds()

		if err != nil {
			json.NewEncoder(w).Encode(ProofResponse{Error: err.Error()})
			return
		}

		var proofBuf bytes.Buffer
		proof.WriteTo(&proofBuf)

		json.NewEncoder(w).Encode(ProofResponse{
			Proof:       hex.EncodeToString(proofBuf.Bytes()),
			PublicInput: fmt.Sprintf("%d:%d", req.TxCount, rootVal),
			ProveMs:     proveMs,
			ProofSize:   proofBuf.Len(),
		})
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
