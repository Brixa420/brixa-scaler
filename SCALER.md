# Brixa Scaler

Network routing and TPS layer.

## Honest ZK Benchmark Status (March 26, 2026)

### Hardware
- Mac mini M4

### Circuit Comparison

| Circuit | Constraints | TPS | Status |
|---------|-------------|-----|--------|
| Trivial (add) | 1 | 850K | ❌ Fake |
| Simple Merkle | 41 | 256K | ✓ Working |
| MiMC-style | 80 | N/A | ❌ Witness bug |

### Working Benchmark: 41-constraint circuit
- Code: `keys/zk_real_shards.go`  
- Operations: Select + Mul + Add
- Verified: Proofs validate, wrong inputs rejected
- TPS: 256K (single shard), 1.36M (36 shards)

### Honest Assessment
- 256K TPS is real and verified
- Not production-grade (needs Poseidon/MiMC with 10K+ constraints)
- gnark is fast but real ZK circuits are slower

### Code Available
All Go files for Kimi to verify:
- keys/zk_real_shards.go - Working 41-constraint benchmark
- keys/zk_check.go - Verification test

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
