# Brixa Scaler

Network routing and TPS layer.

## Final Results (March 26, 2026)

### Best Configuration: 6 Levels + 4 Parallel Provers

| Levels | Constraints | Parallel | TPS |
|--------|-------------|----------|-----|
| **6** | **155K** | **4** | **19,467** |
| 8 | 205K | 4 | 15,143 |
| 10 | 510K | 1 | 13,656 |
| 20 | 1.01M | 1 | 6,339 |
| 32 | 1.61M | 1 | 3,219 |

### Sweet Spot
- **19,467 TPS** at only 1 second latency
- 4 parallel provers (minimal resources)
- 155K constraints (light circuit)

### Code
- keys/zk_6_levels_4.go - Best config (19,467 TPS)
- keys/zk_batch_10k.go - 10 levels (13,656 TPS)

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
