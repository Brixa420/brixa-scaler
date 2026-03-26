# Brixa Scaler

Network routing and TPS layer.

## Parallelism + Batching (March 26, 2026)

### Results

| Parallel | Batch | Total TXS | TPS |
|----------|-------|-----------|-----|
| 1 | 5000 | 5,000 | 9,660 |
| 4 | 5000 | 20,000 | **15,143** |
| 8 | 5000 | 40,000 | 15,187 |
| 16 | 5000 | 80,000 | 15,181 |
| 64 | 5000 | 320,000 | 16,001 |
| 128 | 5000 | 640,000 | 15,949 |

### Finding
- **Plateau: ~16K TPS** (CPU limited on M4)
- Combining batching + parallelism gives best results
- More provers doesn't help beyond 4-8 (CPU saturation)

### Best Config
- **4 parallel provers** = 15,143 TPS
- Sweet spot: minimal latency + high throughput

### Next Steps
1. GPU acceleration (gnark has no Metal, need custom)
2. Multiple machines (horizontal scaling)
3. Smaller circuit (fewer levels)

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
