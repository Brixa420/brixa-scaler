package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// Simple: A * B = C (221 constraints)
type Circuit struct {
	A frontend.Variable `gnark:",public"`
	B frontend.Variable `gnark:"secret"`
	C frontend.Variable `gnark:"secret"`
}

func (c *Circuit) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(c.B, c.C), c.A)
	return nil
}

type PublicCircuit struct {
	A frontend.Variable `gnark:",public"`
}

func (c *PublicCircuit) Define(api.API) error {
	return nil
}

var pk groth16.ProvingKey
var vk groth16.VerifyingKey
var csFull, csPublic interface{ GetNbConstraints() int }
var fieldMod *big.Int

func main() {
	port := "4111"
	fieldMod = ecc.BN254.ScalarField()

	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║     THROUGHPUT TEST (Prove Only, No Verify)           ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")

	csFull, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &Circuit{})
	csPublic, _ = frontend.Compile(fieldMod, r1cs.NewBuilder, &PublicCircuit{})
	pk, vk, _ = groth16.Setup(csFull)

	fmt.Println("Circuit constraints:", csFull.GetNbConstraints())
	fmt.Println("🚀 Ready on port", port)

	// Build tree endpoint (just returns dummy proofs)
	http.HandleFunc("/buildtree", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Leaves []string `json:"leaves"` }
		json.NewDecoder(r.Body).Decode(&req)

		n := len(req.Leaves)
		type Proof struct {
			A, B, C string
		}
		proofs := make([]Proof, n)
		root := "1000"
		for i := 0; i < n; i++ {
			proofs[i] = Proof{A: root, B: req.Leaves[i], C: "1000"}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"proofs": proofs,
			"root": root,
		})
	})

	// Prove endpoint - just prove, don't verify
	http.HandleFunc("/prove", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			A string `json:"A"`
			B string `json:"B"`
			C string `json:"C"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		a := big.NewInt(1000)
		if req.A != "" {
			a.SetString(req.A, 0)
		}
		a.Mod(a, fieldMod)

		b := big.NewInt(1)
		if req.B != "" {
			b.SetString(req.B, 0)
		}
		b.Mod(b, fieldMod)

		c := big.NewInt(1000)
		if req.C != "" {
			c.SetString(req.C, 0)
		}
		c.Mod(c, fieldMod)

		var witness Circuit
		witness.A = a
		witness.B = b
		witness.C = c

		startProve := time.Now()
		wt, _ := frontend.NewWitness(&witness, fieldMod)
		proof, err := groth16.Prove(csFull, pk, wt)
		proveMs := float64(time.Since(startProve).Microseconds()) / 1000

		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		var proofBuf bytes.Buffer
		proof.WriteTo(&proofBuf)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"proof": hex.EncodeToString(proofBuf.Bytes()),
			"proveMs": proveMs,
			"proofSize": proofBuf.Len(),
		})
	})

	fmt.Println("Server starting...")
	http.ListenAndServe(":"+port, nil)
}
