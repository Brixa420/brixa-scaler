package main

import (
	"crypto/sha256"
	"fmt"
	
	"runtime"
	"sync"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

type SimpleCircuit struct {
	A, B, C frontend.Variable
}

func (c *SimpleCircuit) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(c.A, c.B), c.C)
	return nil
}

type MerkleShardProof struct {
	ShardID int
	Root    [32]byte
	Valid   bool
}

func CreateMerkleRoot(shardRoots [][]byte) [32]byte {
	var root [32]byte
	if len(shardRoots) == 0 {
		return root
	}
	h := sha256.New()
	for _, sr := range shardRoots {
		h.Write(sr)
	}
	copy(root[:], h.Sum(nil))
	return root
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	curve := ecc.BN254
	mod := curve.ScalarField()

	ccs, _ := frontend.Compile(mod, r1cs.NewBuilder, &SimpleCircuit{})
	pk, _, _ := groth16.Setup(ccs)

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           MERKLE SHARDING - Parallel Proofs                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	proveShard := func(shardID, size int) MerkleShardProof {
		h := sha256.New()
		for i := 0; i < size; i++ {
			h.Write([]byte(fmt.Sprintf("tx:%d:%d", shardID, i)))
		}
		shardRoot := h.Sum(nil)
		
		a := 0
		for _, b := range shardRoot[:8] {
			a += int(b)
		}
		b := size
		c := a * b
		
		witness := &SimpleCircuit{A: a, B: b, C: c}
		wt, _ := frontend.NewWitness(witness, mod)
		_, err := groth16.Prove(ccs, pk, wt)
		
		return MerkleShardProof{
			ShardID: shardID,
			Root:    sha256.Sum256(shardRoot),
			Valid:   err == nil,
		}
	}

	for _, totalTXs := range []int{100, 1000, 10000} {
		for _, shardCount := range []int{1, 10, 50, 100} {
			txsPerShard := totalTXs / shardCount
			if txsPerShard == 0 {
				continue
			}
			
			start := time.Now()
			var wg sync.WaitGroup
			results := make(chan MerkleShardProof, shardCount)
			
			for s := 0; s < shardCount; s++ {
				wg.Add(1)
				go func(sid int) {
					defer wg.Done()
					proof := proveShard(sid, txsPerShard)
					results <- proof
				}(s)
			}
			wg.Wait()
			close(results)
			
			var shardRoots [][]byte
			valid := 0
			for r := range results {
				if r.Valid {
					valid++
				}
				shardRoots = append(shardRoots, r.Root[:])
			}
			_ = CreateMerkleRoot(shardRoots)
			
			ms := time.Since(start).Milliseconds()
			tps := float64(totalTXs) / float64(ms) * 1000
			fmt.Printf("%5d TXs / %2d shards (%3d TX/shard): %5dms = %7.0f TPS\n",
				totalTXs, shardCount, txsPerShard, ms, tps)
		}
	}
}
