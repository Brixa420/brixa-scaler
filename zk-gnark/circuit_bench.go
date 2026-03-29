package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// Trivial: A * B = C (1 constraint)
type Circuit1 struct {
	A, B, C frontend.Variable
}
func (c *Circuit1) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(c.A, c.B), c.C)
	return nil
}

// Medium: 10 multiplications
type Circuit10 struct {
	A, B, C, D, E, F, G frontend.Variable
}
func (c *Circuit10) Define(api frontend.API) error {
	api.AssertIsEqual(
		api.Add(api.Mul(c.A, c.B), api.Mul(c.C, c.D), api.Mul(c.E, c.F)),
		c.G,
	)
	return nil
}

// Large: 100 multiplications
type Circuit100 struct {
	Values [101]frontend.Variable
}
func (c *Circuit100) Define(api frontend.API) error {
	result := c.Values[0]
	for i := 0; i < 100; i++ {
		result = api.Mul(result, c.Values[i+1])
	}
	api.AssertIsEqual(result, c.Values[100])
	return nil
}

func runBenchmark(name string, cir frontend.Circuit) {
	curve := ecc.BN254
	mod := curve.ScalarField()
	
	ccs, _ := frontend.Compile(mod, r1cs.NewBuilder, cir)
	pk, vk, _ := groth16.Setup(ccs)
	fmt.Printf("%s: %d constraints\n", name, ccs.GetNbConstraints())
	
	start := time.Now()
	for i := 0; i < 100; i++ {
		wt, _ := frontend.NewWitness(cir, mod)
		groth16.Prove(ccs, pk, wt)
	}
	ms := time.Since(start).Milliseconds()
	fmt.Printf("  100 proofs: %dms = %.0f/sec\n", ms, float64(100)/float64(ms)*1000)
	_ = vk
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("=== CIRCUIT COMPLEXITY vs PROVING SPEED ===\n")
	
	runBenchmark("1 mul (trivial)", &Circuit1{})
	runBenchmark("10 muls", &Circuit10{})
	runBenchmark("100 muls", &Circuit100{})
}
