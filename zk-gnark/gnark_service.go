package main

import (
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

type ProofRequest struct {
	TxCount    int    `json:"txCount"`
	BatchHash  string `json:"batchHash"`
}

type BatchProofRequest struct {
	Proofs []ProofRequest `json:"proofs"`
}

type BatchProofResponse struct {
	Count int `json:"count"`
}

type SimpleCircuit struct {
	A, B, C frontend.Variable
}

func (c *SimpleCircuit) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(c.A, c.B), c.C)
	return nil
}

var pk groth16.ProvingKey

func main() {
	useUnix := len(os.Args) > 1 && os.Args[1] == "unix"
	port := "4111"
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	curve := ecc.BN254
	mod := curve.ScalarField()

	r1, err := frontend.Compile(mod, r1cs.NewBuilder, &SimpleCircuit{})
	if err != nil {
		log.Fatal("compile: ", err)
	}

	pk, _, err = groth16.Setup(r1)
	if err != nil {
		log.Fatal("setup: ", err)
	}

	log.Printf("gnark: ready")

	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req ProofRequest
		json.NewDecoder(r.Body).Decode(&req)

		start := time.Now()
		a := new(big.Int)
		if len(req.BatchHash) > 16 {
			a.SetString(req.BatchHash[:16], 16)
		} else {
			a.SetString(req.BatchHash, 16)
		}
		a.Mod(a, mod)

		b := big.NewInt(int64(req.TxCount))
		c := new(big.Int).Mul(a, b)
		c.Mod(c, mod)

		witness := &SimpleCircuit{A: a, B: b, C: c}
		wt, _ := frontend.NewWitness(witness, mod)
		proof, err := groth16.Prove(r1, pk, wt)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		_ = proof

		json.NewEncoder(w).Encode(map[string]interface{}{
			"proveMs": time.Since(start).Milliseconds(),
			"valid":   err == nil,
		})
	})

	http.HandleFunc("/provebatch", func(w http.ResponseWriter, r *http.Request) {
		var req BatchProofRequest
		json.NewDecoder(r.Body).Decode(&req)

		count := 0
		for _, pr := range req.Proofs {
			a := new(big.Int)
			if len(pr.BatchHash) > 16 {
				a.SetString(pr.BatchHash[:16], 16)
			} else {
				a.SetString(pr.BatchHash, 16)
			}
			a.Mod(a, mod)

			b := big.NewInt(int64(pr.TxCount))
			c := new(big.Int).Mul(a, b)
			c.Mod(c, mod)

			witness := &SimpleCircuit{A: a, B: b, C: c}
			wt, _ := frontend.NewWitness(witness, mod)
			proof, err := groth16.Prove(r1, pk, wt)
			if err != nil {
				continue
			}
			_ = proof
			count++
		}

		json.NewEncoder(w).Encode(BatchProofResponse{Count: count})
	})

	if useUnix {
		os.Remove("/tmp/gnark.sock")
		fmt.Println("gnark on unix socket /tmp/gnark.sock")
		log.Fatal(http.ListenAndServe("unix:///tmp/gnark.sock", nil))
	} else {
		fmt.Printf("gnark ZK on port %s\n", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}
}
