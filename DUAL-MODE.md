# BrixaScaler - Dual Mode Architecture

## Two Modes

### ⚡ FAST MODE (Default)
- **Speed:** ~16M TPS batching, ~12K TPS with ZK
- **Privacy:** None (transactions visible)
- **ZK:** Disabled
- **Use Case:** High-volume, non-sensitive transactions

### 🔐 PRIVACY MODE
- **Speed:** ~25K TPS
- **Privacy:** Full zero-knowledge proofs
- **ZK:** Enabled (snarkjs groth16)
- **Use Case:** Financial transactions, DeFi, privacy-sensitive operations

## Performance Comparison

| Batch | Fast Mode | Privacy Mode |
|-------|-----------|---------------|
| 100 | 100K+ TPS | ~260 TPS |
| 1,000 | 500K TPS | ~2.6K TPS |
| 10,000 | 833K TPS | ~25K TPS |
| 100,000 | **1M+ TPS** | ~200K TPS |

## Usage

### Fast Mode (default)
```bash
node brixa-scaler.js --rpc https://your-rpc
# No ZK, max speed
```

### Privacy Mode
```bash
# Set env var to enable ZK
ZK_MODE=full node brixa-scaler.js --rpc https://your-rpc
# Or use benchmark
node benchmark.js
```

## Architecture

```
Client Request
      │
      ▼
┌─────────────────┐
│  BrixaScaler    │
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
FAST     PRIVACY
   │         │
   └────┬────┘
        ▼
   Submit to L1/L2
```

## Decision Matrix

| Need | Use |
|------|-----|
| Max throughput | Fast mode |
| Privacy | Privacy mode |
| Settlement cost | Privacy (1 tx = 1000s batched) |
| Microtransactions | Fast mode |

## Implementation

- **Fast mode:** SHA256 Merkle tree only
- **Privacy mode:** SHA256 + snarkjs ZK verification
- **Config:** `ZK_MODE=full` enables privacy

The user chooses based on their needs. Both are valid use cases for BrixaScaler.