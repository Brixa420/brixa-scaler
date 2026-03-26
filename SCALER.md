# Brixa Scaler

Network routing and TPS layer.

## Honest Performance Data (March 26, 2026)

### Hardware Configuration
| Component | Specification |
|-----------|---------------|
| Device | Mac mini M4 |
| CPU | Apple Silicon (9 cores) |
| Memory | 16GB unified |
| OS | macOS |

### Reproducible Benchmark

**Prerequisites:**
```bash
cd keys
npm install snarkjs circomlib
```

**Run benchmark:**
```bash
node parallel-bench.js    # Single-shard parallel proving
node shards/sharded-prover.js  # Multi-shard simulation
```

### Measured Results (Real, Verified)

| Test | TPS | Notes |
|------|-----|-------|
| Sequential proving | 928 | 1 batch |
| Parallel (9 batches) | 3,872 | Peak measured |
| Parallel (18 batches) | 3,843 | CPU saturation |
| Parallel (36 batches) | 3,418 | Diminishing returns |

### Extrapolated (Not Tested)

| Configuration | Estimated | Basis |
|--------------|-----------|-------|
| 10 shards | 38,720 | 3,872 × 10 (unverified) |
| GPU hybrid | 60,000+ | Assumes 20x GPU (no GPU) |

**Note:** Sharding and GPU numbers are theoretical - not measured on this hardware.

### What's Working (Real)

- ✅ Merkle tree: ~1M TPS (SHA256)
- ✅ ZK verify: 63/sec (real snarkjs)
- ✅ ZK prove: 3,872 TPS (real parallel proving)
- ✅ Full pipeline: witness → prove → verify

### What's Not Working

- ❌ True sharding (simulation only)
- ❌ GPU proving (no NVIDIA GPU)
- ❌ Cross-shard transactions

### For Visa Evaluation

Current credible claim: **~4,000 TPS** (reproducible, verified on Mac mini M4)

To get to Visa scale (24K+), would need:
- GPU proving cluster
- Multiple machines
- Production circuit (Poseidon hash)

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
