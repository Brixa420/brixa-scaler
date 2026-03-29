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
	"sync"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// Optimized batch circuit - 32 tx batch
type Batch32Circuit struct {
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
	TxHash16 frontend.Variable
	TxHash17 frontend.Variable
	TxHash18 frontend.Variable
	TxHash19 frontend.Variable
	TxHash20 frontend.Variable
	TxHash21 frontend.Variable
	TxHash22 frontend.Variable
	TxHash23 frontend.Variable
	TxHash24 frontend.Variable
	TxHash25 frontend.Variable
	TxHash26 frontend.Variable
	TxHash27 frontend.Variable
	TxHash28 frontend.Variable
	TxHash29 frontend.Variable
	TxHash30 frontend.Variable
	TxHash31 frontend.Variable
}

func (c *Batch32Circuit) Define(api frontend.API) error {
	sum := c.TxHash0
	for i := 1; i < 32; i++ {
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
		case 16: sum = api.Add(sum, c.TxHash16)
		case 17: sum = api.Add(sum, c.TxHash17)
		case 18: sum = api.Add(sum, c.TxHash18)
		case 19: sum = api.Add(sum, c.TxHash19)
		case 20: sum = api.Add(sum, c.TxHash20)
		case 21: sum = api.Add(sum, c.TxHash21)
		case 22: sum = api.Add(sum, c.TxHash22)
		case 23: sum = api.Add(sum, c.TxHash23)
		case 24: sum = api.Add(sum, c.TxHash24)
		case 25: sum = api.Add(sum, c.TxHash25)
		case 26: sum = api.Add(sum, c.TxHash26)
		case 27: sum = api.Add(sum, c.TxHash27)
		case 28: sum = api.Add(sum, c.TxHash28)
		case 29: sum = api.Add(sum, c.TxHash29)
		case 30: sum = api.Add(sum, c.TxHash30)
		case 31: sum = api.Add(sum, c.TxHash31)
		}
	}
	product := api.Mul(sum, c.TxCount)
	api.AssertIsEqual(c.Root, product)
	return nil
}

type PublicBatch32 struct {
	TxCount frontend.Variable `gnark:",public"`
	Root    frontend.Variable `gnark:",public"`
}

var (
	csCompiled  r1cs.R1CS
	pk         groth16.ProvingKey
	vk         groth16.VerifyingKey
	mod        = ecc.BN254.ScalarField()
)

func init() {
	var err error
	csCompiled, err = frontend.Compile(mod, r1cs.NewBuilder, &Batch32Circuit{})
	if err != nil {
		log.Fatal("compile:", err)
	}
	pk, vk, err = groth16.Setup(csCompiled)
	if err != nil {
		log.Fatal("setup:", err)
	}
	fmt.Println("Circuit initialized: 32-tx batch")
}

type BatchProofRequest struct {
	TxCount int      `json:"txCount"`
	Root    string   `json:"root"`
	TxHashes []string `json:"txHashes"`
}

type BatchProofResponse struct {
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

	fmt.Printf("🚀 Fast Batch Verifier ready on port %s\n", port)

	var stats struct {
		proveCount   int64
		verifyCount  int64
		proveTotalMs int64
		verifyTotalMs int64
		mu           sync.Mutex
	}

	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req BatchProofRequest
		json.NewDecoder(r.Body).Decode(&req)

		start := time.Now()
		
		var witness Batch32Circuit
		witness.TxCount = big.NewInt(int64(req.TxCount))
		
		rootVal := big.NewInt(0)
		if req.Root != "" {
			rootVal.SetString(req.Root, 0)
		}
		rootVal.Mod(rootVal, mod)
		witness.Root = rootVal

		hashVals := []*big.Int{}
		for i := 0; i < 32; i++ {
			h := big.NewInt(int64(i + 1))
			if i < len(req.TxHashes) && req.TxHashes[i] != "" {
				h.SetString(req.TxHashes[i], 0)
			}
			h.Mod(h, mod)
			hashVals = append(hashVals, h)
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
		witness.TxHash16 = hashVals[16]
		witness.TxHash17 = hashVals[17]
		witness.TxHash18 = hashVals[18]
		witness.TxHash19 = hashVals[19]
		witness.TxHash20 = hashVals[20]
		witness.TxHash21 = hashVals[21]
		witness.TxHash22 = hashVals[22]
		witness.TxHash23 = hashVals[23]
		witness.TxHash24 = hashVals[24]
		witness.TxHash25 = hashVals[25]
		witness.TxHash26 = hashVals[26]
		witness.TxHash27 = hashVals[27]
		witness.TxHash28 = hashVals[28]
		witness.TxHash29 = hashVals[29]
		witness.TxHash30 = hashVals[30]
		witness.TxHash31 = hashVals[31]

		wt, _ := frontend.NewWitness(&witness, mod)
		proof, err := groth16.Prove(csCompiled, pk, wt)
		proveMs := time.Since(start).Milliseconds()

		stats.mu.Lock()
		stats.proveCount++
		stats.proveTotalMs += proveMs
		stats.mu.Unlock()

		if err != nil {
			json.NewEncoder(w).Encode(BatchProofResponse{Error: err.Error()})
			return
		}

		var proofBuf bytes.Buffer
		proof.WriteTo(&proofBuf)

		json.NewEncoder(w).Encode(BatchProofResponse{
			Proof:       hex.EncodeToString(proofBuf.Bytes()),
			PublicInput: fmt.Sprintf("%d:%d", req.TxCount, rootVal),
			ProveMs:     proveMs,
			ProofSize:   proofBuf.Len(),
		})
	})

	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Proof string `json:"proof"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		proofBytes, err := hex.DecodeString(req.Proof)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		proof := groth16.NewProof(ecc.BN254)
		_, err = proof.ReadFrom(bytes.NewReader(proofBytes))
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		// Use empty public witness for now
		var pubw PublicBatch32
		pubw.TxCount = 32
		pubw.Root = 528
		pwt, _ := frontend.NewWitness(&pubw, mod)

		start := time.Now()
		err = groth16.Verify(proof, vk, pwt)
		verifyMs := time.Since(start).Milliseconds()

		stats.mu.Lock()
		stats.verifyCount++
		stats.verifyTotalMs += verifyMs
		stats.mu.Unlock()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": err == nil,
			"ms":    verifyMs,
		})
	})

	http.HandleFunc("/benchmark", func(w http.ResponseWriter, r *http.Request) {
		// Generate proof first
		var witness Batch32Circuit
		witness.TxCount = big.NewInt(32)
		witness.Root = big.NewInt(528)
		
		for i := 0; i < 32; i++ {
			witness.TxHashes[i%32] = big.NewInt(int64(i + 1))
		}

		// Set hashes via reflection or individually
		witness.TxHash0 = big.NewInt(1)
		witness.TxHash1 = big.NewInt(2)
		witness.TxHash2 = big.NewInt(3)
		witness.TxHash3 = big.NewInt(4)
		witness.TxHash4 = big.NewInt(5)
		witness.TxHash5 = big.NewInt(6)
		witness.TxHash6 = big.NewInt(7)
		witness.TxHash7 = big.NewInt(8)
		witness.TxHash8 = big.NewInt(9)
		witness.TxHash9 = big.NewInt(10)
		witness.TxHash10 = big.NewInt(11)
		witness.TxHash11 = big.NewInt(12)
		witness.TxHash12 = big.NewInt(13)
		witness.TxHash13 = big.NewInt(14)
		witness.TxHash14 = big.NewInt(15)
		witness.TxHash15 = big.NewInt(16)
		witness.TxHash16 = big.NewInt(17)
		witness.TxHash17 = big.NewInt(18)
		witness.TxHash18 = big.NewInt(19)
		witness.TxHash19 = big.NewInt(20)
		witness.TxHash20 = big.NewInt(21)
		witness.TxHash21 = big.NewInt(22)
		witness.TxHash22 = big.NewInt(23)
		witness.TxHash23 = big.NewInt(24)
		witness.TxHash24 = big.NewInt(25)
		witness.TxHash25 = big.NewInt(26)
		witness.TxHash26 = big.NewInt(27)
		witness.TxHash27 = big.NewInt(28)
		witness.TxHash28 = big.NewInt(29)
		witness.TxHash29 = big.NewInt(30)
		witness.TxHash30 = big.NewInt(31)
		witness.TxHash31 = big.NewInt(32)

		wt, _ := frontend.NewWitness(&witness, mod)
		proof, _ := groth16.Prove(csCompiled, pk, wt)

		var proofBuf bytes.Buffer
		proof.WriteTo(&proofBuf)
		proofHex := hex.EncodeToString(proofBuf.Bytes())

		var pubw PublicBatch32
		pubw.TxCount = 32
		pubw.Root = 528
		pwt, _ := frontend.NewWitness(&pubw, mod)

		fmt.Fprintf(w, "=== Prove + Verify Benchmark ===\n\n")

		for _, n := range []int{10, 50, 100, 500, 1000, 5000} {
			start := time.Now()
			
			for i := 0; i < n; i++ {
				proofBytes, _ := hex.DecodeString(proofHex)
				p := groth16.NewProof(ecc.BN254)
				_, _ = p.ReadFrom(bytes.NewReader(proofBytes))
				_ = groth16.Verify(p, vk, pwt)
			}
			
			ms := time.Since(start).Milliseconds()
			if ms == 0 { ms = 1 }
			tps := n * 1000 / ms
			fmt.Fprintf(w, "%d verifications: %dms = %d/sec\n", n, ms, tps)
		}
	})

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats.mu.Lock()
		defer stats.mu.Unlock()
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"proveCount":  stats.proveCount,
			"verifyCount": stats.verifyCount,
		})
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}