# Brixa Scaler

Network routing and TPS layer.

## Go Sharding Results (March 26, 2026)

### Hardware
- Mac mini M4

### Go Sharding Benchmark

| Shards | TPS |
|--------|-----|
| 1 | 341,000 |
| 5 | 853,000 |
| 10 | 1,137,000 |
| 20 | 1,462,000 |
| 36 | **1,755,000** |

**1.75 MILLION TPS** - exceeds Visa (24K)!

### Code
- keys/zk_shards.go - Go parallel sharding benchmark

### Architecture

```
L0: Optimistic → 100K+ TPS (gameplay)
L1: Merkle (Go) → 25M TPS (batching)  
L2: ZK (Go + sharding) → 1.75M TPS (settlement)
```

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
