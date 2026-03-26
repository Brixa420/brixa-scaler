# Brixa Scaler

Network routing and TPS layer.

## Peak ZK Performance (March 26, 2026)

### Scaling Test Results

| Batches | Time | Txs | TPS |
|---------|------|-----|-----|
| 9 | 2,410ms | 9,216 | 3,824 |
| 18 | 4,760ms | 18,432 | **3,872** |
| 36 | 10,786ms | 36,864 | 3,418 |
| 72 | 22,802ms | 73,728 | 3,233 |
| 100 | 34,747ms | 102,400 | 2,947 |
| 144 | 79,900ms | 147,456 | 1,846 |

### Peak Performance
**3,872 TPS** at 18 parallel batches (Mac mini M4)

### Analysis
- Optimal: ~18 batches (balancing parallelism vs overhead)
- Beyond 18: CPU saturation causes diminishing returns
- Bottleneck: CPU cores (not memory or I/O)

### Current Limits
- Circuit: Simple addition hash
- Batches: 1,024 txs each
- Workers: ~9 parallel

### Next Upgrades
- Poseidon hash (production grade)
- Larger circuit (more txs per batch)
- GPU acceleration (20x potential)

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
