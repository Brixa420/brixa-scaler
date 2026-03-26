# Brixa Scaler

Network routing and TPS layer.

## Real Sharding Test Results (March 26, 2026)

### Hardware
- Mac mini M4 (Apple Silicon, 9 cores)

### Test Methodology
Each shard runs full pipeline: witness → prove → verify
All 10 shards run in parallel

### Results

| Configuration | TPS | vs Single |
|---------------|-----|-----------|
| Single shard | 928 | 1x |
| 10 shards parallel | 2,411 | **2.6x** |

### Why Not 10x?
CPU saturation - running 10 full proving pipelines in parallel hits the same CPU limit as before. Each shard needs ~4s, so all 10 take ~4s (parallelized).

### Lesson
Sharding helps but isn't magic - you're still bound by total CPU. To get 10x you'd need:
- 10 separate machines (not one Mac mini)
- Or GPU acceleration per shard

### Benchmark Code
```bash
node parallel-bench.js  # Reproducible
```

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
