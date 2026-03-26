# Brixa Scaler

Network routing and TPS layer.

## Final ZK Results (March 26, 2026)

### Batching with Different Tree Depths

| Levels | Tree Size | Constraints | TPS |
|--------|-----------|------------|-----|
| 10 | 1K | 510K | **13,656** |
| 20 | 1M | 1.01M | **6,339** |
| 32 | 4B | 1.61M | **3,219** |

### Recommendation: 20 Levels
- **6,339 TPS** with 1 million leaf capacity
- Good balance of security (2^20) vs speed
- Sweet spot for most applications

### Code
- keys/zk_batch_10k.go - 13K TPS (10 levels)
- keys/zk_batch_20_10k.go - 6K TPS (20 levels, 1M leaves)
- keys/zk_batch_32_10k.go - 3K TPS (32 levels, 4B leaves)

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
