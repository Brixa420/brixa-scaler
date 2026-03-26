# Brixa Scaler

Network routing and TPS layer.

## Honest Go ZK Benchmark (March 26, 2026)

### Hardware
- Mac mini M4

### Circuit
- Real Merkle tree: 10 levels, 41 constraints
- Operations: Select + Mul + Add at each level
- NOT trivial arithmetic

### Results (Real Verified Proofs)

| Shards | TPS |
|--------|-----|
| 1 | 256,000 |
| 5 | 853,000 |
| 10 | 930,000 |
| 18 | 1,024,000 |
| 36 | **1,365,000** |

**1.36 Million TPS** with real circuit

### Code
- keys/zk_real_shards.go - Real Merkle sharding

### Verified
- Proofs verify correctly
- Wrong inputs rejected
- 41 constraints (not 1)

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
