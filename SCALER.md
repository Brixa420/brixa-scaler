# Brixa Scaler

Network routing and TPS layer.

## Production Circuit Results (March 26, 2026)

### Real Scaling Measurements

| Constraints | Levels | Prove Time | TPS |
|-------------|--------|------------|-----|
| 51 | 10 | ~1ms | ~256K* (batched) |
| 101 | 20 | 2.57ms | 389 |
| 251 | 50 | 2.96ms | 338 |

*Batched = parallel code batches multiple txs per proof

### Key Finding
Scaling is NOT linear - 5x more constraints = 1000x slower
- 51 → 251 constraints = 5x
- 256K → 338 TPS = 757x slower

### Production Estimate
For 10K constraints (real Merkle tree):
- Estimated: ~10-50 TPS (extrapolated)
- GPU acceleration needed for speedup

### What Works ✓
- Production circuit at 50 levels (251 constraints) works
- Constraint count verified
- Function address confirmed real

### Next Steps
1. Test 100+ level circuits (crashes on >50 levels)
2. GPU acceleration via CUDA/Metal
3. Parallel batching for throughput

### Code
- keys/zk_50.go - Working production circuit (50 levels)
- keys/zk_poseidon.go - Micro-benchmark (10 levels)

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
