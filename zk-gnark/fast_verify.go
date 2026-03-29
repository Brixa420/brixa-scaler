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

// Simple batch circuit: verify sum(txHashes) * txCount == root
type BatchCircuit struct {
	TxCount frontend.Variable `gnark:",public"`
	Root    frontend.Variable `gnark:",public"`
	H0, H1, H2, H3, H4, H5, H6, H7 frontend.Variable
	H8, H9, H10, H11, H12, H13, H14, H15 frontend.Variable
}

func (c *BatchCircuit) Define(api frontend.API) error {
	sum := api.Add(c.H0, c.H1, c.H2, c.H3, c.H4, c.H5, c.H6, c.H7)
	sum = api.Add(sum, c.H8, c.H9, c.H10, c.H11, c.H12, c.H13, c.H14, c.H15)
	product := api.Mul(sum, c.TxCount)
	api.AssertIsEqual(c.Root, product)
	return nil
}

var (
	curve       = ecc.BN254
	mod         = curve.ScalarField()
	r1csCS      interface{}
	pk          groth16.ProvingKey
	vk          groth16.VerifyingKey
	preproof    []byte
	preproofHex string
)

func init() {
	var err error
	r1csCS, err = frontend.Compile(mod, r1cs.NewBuilder, &BatchCircuit{})
	if err != nil {
		log.Fatal("compile:", err)
	}
	pk, vk, err = groth16.Setup(r1csCS)
	if err != nil {
		log.Fatal("setup:", err)
	}
	
	// Pre-generate proof for benchmarking
	witness := &BatchCircuit{
		TxCount: 16,
		Root:    big.NewInt(136),
		H0: big.NewInt(1), H1: big.NewInt(2), H2: big.NewInt(3), H3: big.NewInt(4),
		H4: big.NewInt(5), H5: big.NewInt(6), H6: big.NewInt(7), H7: big.NewInt(8),
		H8: big.NewInt(9), H9: big.NewInt(10), H10: big.NewInt(11), H11: big.NewInt(12),
		H12: big.NewInt(13), H13: big.NewInt(14), H14: big.NewInt(15), H15: big.NewInt(16),
	}
	wt, _ := frontend.NewWitness(witness, mod)
	proof, _ := groth16.Prove(r1csCS, pk, wt)
	
	var buf bytes.Buffer
	proof.WriteTo(&buf)
	preproof = buf.Bytes()
	preproofHex = hex.EncodeToString(preproof)
	
	fmt.Println("Fast verifier initialized")
}

type ProofRequest struct {
	Proof string `json:"proof"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4111"
	}

	fmt.Printf("🚀 Fast ZK Verifier on port %s\n", port)

	var stats struct {
		mu       sync.Mutex
		proved   int64
		verified int64
		proveMs  int64
		verifyMs int64
	}

	// Single proof verification
	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		var req ProofRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		start := time.Now()
		
		proofBytes, err := hex.DecodeString(req.Proof)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid proof"})
			return
		}

		// Deserialize proof
		proof := groth16.NewProof(curve)
		if _, err := proof.ReadFrom(bytes.NewReader(proofBytes)); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "read failed"})
			return
		}

		// Create public witness (same as preproof)
		witness := &BatchCircuit{
			TxCount: 16,
			Root:    big.NewInt(136),
		}
		wt, _ := frontend.NewWitness(witness, mod)

		err = groth16.Verify(proof, vk, wt)
		ms := time.Since(start).Milliseconds()

		stats.mu.Lock()
		stats.verified++
		stats.verifyMs += ms
		stats.mu.Unlock()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": err == nil,
			"ms":    ms,
		})
	})

	// Benchmark endpoint
	http.HandleFunc("/benchmark", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "=== ZK Verification Benchmark ===")
		fmt.Fprintln(w)

		for _, n := range []int{10, 50, 100, 500, 1000, 5000} {
			start := time.Now()
			for i := 0; i < n; i++ {
				proof := groth16.NewProof(curve)
				proof.ReadFrom(bytes.NewReader(preproof))
				
				witness := &BatchCircuit{TxCount: 16, Root: big.NewInt(136)}
				wt, _ := frontend.NewWitness(witness, mod)
				_ = groth16.Verify(proof, vk, wt)
			}
			ms := time.Since(start).Milliseconds()
			if ms == 0 { ms = 1 }
			tps := n * 1000 / ms
			fmt.Fprintf(w, "%5d verifications: %5dms = %5d/sec\n", n, ms, tps)
		}
	})

	// Stats
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats.mu.Lock()
		defer stats.mu.Unlock()
		if stats.verified == 0 {
			json.NewEncoder(w).Encode(map[string]int64{"verified": 0})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"verified":    stats.verified,
			"avgVerifyMs": float64(stats.verifyMs) / float64(stats.verified),
		})
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}