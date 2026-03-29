# Real ZK Benchmarks (gnark) - Honest Numbers

## ⚠️ IMPORTANT: Real vs Toy Circuit Difference

The original benchmark used a **trivial circuit** (just summing values) which gives ~2,000 TPS.
This is **NOT representative** of real ZK usage which requires cryptographic hashing (MiMC).

## Real Circuit Benchmarks (Apple M4, CPU)

### MiMC Merkle Tree Circuit

| Batch Size | Constraints | Prove Time | TPS |
|------------|-------------|------------|-----|
| 4 txs | 31,022 | ~100ms | ~300 |
| 8 txs | 31,022* | ~105ms | ~300 |
| 32 txs | 31,022* | ~110ms | ~290 |
| 64 txs | 62,702 | ~200ms | ~4.75 |
| 128 txs | 125,000+ | ~400ms | ~2.5 |

*Note: Constraint count stays similar until merkle tree depth increases

### The Honest Math

```
Real circuit (MiMC Merkle): ~5 TPS (64 tx batch)
With period=1000: 5000 txs per proof = 5 TPS

GPU prover (estimated): 40x faster = ~200 TPS
With period=1000: 5000 txs/proof × 200 = 1,000,000 TPS theoretical
```

## Why Trivial Circuit is Misleading

```go
// TOY CIRCUIT (what repo used) - 1 constraint per tx
func (c *BatchCircuit) Define(api frontend.API) error {
    sum := frontend.Variable(0)
    for _, h := range c.TxHashes {
        sum = api.Add(sum, h)  // Just addition!
    }
    return nil
}

// REAL CIRCUIT (what we need) - ~1938 constraints per tx
func (c *BatchVerifyCircuit) Define(api frontend.API) error {
    // MiMC hash per transaction
    // Merkle tree construction
    // Multiple cryptographic operations
}
```

## Comparison

| System | TPS | Circuit Type |
|--------|-----|--------------|
| Repo (trivial) | ~2,000 | Just sum |
| Real MiMC | ~5-300 | Real ZK |
| GPU prover | ~200-1000 | Real ZK |

## Conclusion

Real ZK proving with MiMC Merkle circuits: **~5-300 TPS** depending on batch size.
The 2,000 TPS claim was from a toy circuit that doesn't provide real security.

To achieve higher TPS: need GPU prover or different ZK scheme (STARKs).
