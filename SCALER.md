# Brixa Scaler

Network routing and TPS layer.

## TPS Improvement via Parallelism (March 26, 2026)

### Results
| Method | Constraints | TPS |
|--------|-------------|-----|
| Single proof (1000 levels) | 5001 | 54 |
| Single proof (100 levels) | 501 | 270 |
| Parallel 100 (100 levels) | 501 | 434 |
| Parallel 1000 (51 constraints) | 51 | 1,558 |
| Parallel 2000 (51 constraints) | 51 | 1,616 |

### Analysis
- Parallelism scales: ~1600 TPS with 2000 parallel goroutines
- Single proof at 1000 levels: 54 TPS (real Merkle tree)
- Gap: Need 10x faster for production use

### Honest Assessment
- Current: 54-1,616 TPS depending on circuit size
- Target for AI/gaming: ~10K+ TPS needed
- gnark is CPU-only (no Metal/GPU acceleration in v0.14.0)

### Next Steps
1. Use smaller circuits (51 constraints) + parallelism
2. Batch multiple leaves into single proof (wip)
3. GPU: gnark CUDA or custom implementation

### Code
- keys/zk_max_parallel.go - Parallelism benchmark
- keys/zk_1000.go - Single proof 54 TPS

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
