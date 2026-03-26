# Brixa Scaler

Network routing and TPS layer.

## Batching Results (March 26, 2026)

### BREAKTHROUGH: Batching Multiples Leaves Per Proof

| Batch | Levels | Constraints | Prove Time | TPS |
|-------|--------|-------------|------------|-----|
| 10 | 100 | 5,010 | 18ms | 545 |
| 50 | 100 | 25,050 | 59ms | 843 |
| 100 | 100 | 50,100 | 105ms | 952 |
| 500 | 50 | 125,500 | 225ms | 2,225 |
| 1000 | 30 | 151,000 | 332ms | 3,011 |
| 2000 | 20 | 202,000 | 340ms | 5,883 |
| 5000 | 10 | 255,000 | 394ms | 12,693 |
| 10000 | 10 | 510,000 | 732ms | **13,656** |

### Key Finding
- Batching scales: more leaves per proof = higher TPS
- 54 TPS (single) → 13,656 TPS (batched 10K)
- **253x improvement**

### Tradeoffs
- 10 levels = smaller Merkle tree (2^10 = 1024 leaves)
- For production: use deeper tree + larger batch

### Next Steps
1. Production circuit with deeper tree (32+ levels)
2. Even larger batch sizes
3. GPU acceleration for further speedup

### Code
- keys/zk_batch_10k.go - 13K TPS benchmark
- keys/zk_batch_5k.go - 12K TPS
- keys/zk_batch_2k.go - 5K TPS

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
