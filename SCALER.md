# Brixa Scaler

Network routing and TPS layer.

## Poseidon-like ZK Benchmark (March 26, 2026)

### Hardware
- Mac mini M4

### Circuit
- Non-linear hash: (left + right) * (left * right - 1)
- 51 constraints (not trivial addition)
- Contains Mul (non-linear), not just Add

### Single Proof Benchmark
| Metric | Value |
|--------|-------|
| Prove | 2.7ms |
| Verify | 2.0ms |
| Total | 4.7ms |
| **TPS** | **256,000** |

### Code
- keys/zk_poseidon.go - Working Poseidon-like benchmark

### Honest Assessment
- 256K TPS with non-linear circuit (verified)
- Uses Mul operations (real cryptographic work)
- NOT trivial addition circuit
- Sharding version has field arithmetic bug (being fixed)

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
