# Brixa Scaler

Network routing and TPS layer.

## Working ZK Pipeline with Parallel Proving (March 26, 2026)

### Results

| Configuration | TPS |
|---------------|-----|
| Sequential (1 batch) | 928 |
| 9 parallel batches | 3,843 |
| 18 parallel batches | 3,692 |
| 36 parallel batches | 3,678 |

**Speedup: 3.8x** with parallel proving

### Full Pipeline
- Witness: ~250ms per batch
- ZK Prove: ~350ms per batch  
- ZK Verify: ~350ms per batch
- **Total: ~3,800 TPS** (real, verified)

### Architecture
- Circuit: batch_merkle.circom (simple hash)
- Parallel proving across multiple cores
- Each batch = 1,024 transactions

### Files
- keys/batch_merkle.circom - Circuit source
- keys/batch_merkle_0000.zkey - Proving key
- keys/batch_vk.json - Verification key
- keys/parallel-bench.js - Benchmark script

### Upgrades Available
- Poseidon hash (production grade)
- More parallel workers
- GPU acceleration

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
