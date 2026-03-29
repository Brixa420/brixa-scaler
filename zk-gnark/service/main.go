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

type ProofResponse struct {
	ProveMs int64 `json:"proveMs"`
	Valid   bool  `json:"valid"`
}

type SimpleCircuit struct {
	A frontend.Variable `gnark:"a,public"`
	B frontend.Variable `gnark:"b,public"`
	C frontend.Variable `gnark:"c,public"`
}

func (c *SimpleCircuit) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(c.A, c.B), c.C)
	return nil
}

var ccs r1cs.R1CS
var pk groth16.ProvingKey
var field *big.Int

func main() {
	port := "3111"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	curve := ecc.BN254
	field = curve.ScalarField()

	var err error
	ccs, err = frontend.Compile(field, r1cs.NewBuilder, &SimpleCircuit{})
	if err != nil {
		log.Fatal("compile: ", err)
	}
	pk, _, err = groth16.Setup(ccs)
	if err != nil {
		log.Fatal("setup: ", err)
	}

	log.Printf("gnark: ready on %s", port)

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
		a.Mod(a, field)

		b := big.NewInt(int64(req.TxCount))
		c := new(big.Int).Mul(a, b)
		c.Mod(c, field)

		witness := &SimpleCircuit{A: a, B: b, C: c}
		wt, err := frontend.NewWitness(witness, field)
		if err != nil {
			json.NewEncoder(w).Encode(ProofResponse{ProveMs: 0, Valid: false})
			return
		}
		_, err = groth16.Prove(ccs, pk, wt)

		json.NewEncoder(w).Encode(ProofResponse{
			ProveMs: time.Since(start).Milliseconds(),
			Valid:   err == nil,
		})
	})

	fmt.Printf("gnark ZK on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
