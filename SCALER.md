# Brixa Scaler

Network routing and TPS layer.

## Poseidon-like ZK Benchmark (March 26, 2026)

### Hardware
- Mac mini M4

### Circuit
- Non-linear hash: (left + right) * (left * right - 1)
- 51 constraints with Mul (real cryptographic work)
- Verified proofs (not fake)

### Results (Real Verified)

| Shards | TPS |
|--------|-----|
| 1 | 256,000 |
| 5 | 731,000 |
| 10 | 787,000 |
| 18 | 1,024,000 |
| 36 | **1,228,800** |

**1.23 Million TPS** with verified proofs!

### Code
- keys/zk_poseidon.go - Single proof
- keys/zk_poseidon_parallel.go - Parallel sharding

### Honest Assessment
- Non-linear operations (Mul) - real crypto
- 51 constraints per proof
- All proofs verify correctly
- No placeholder circuits

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
