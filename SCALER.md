# Brixa Scaler

Network routing and TPS layer.

## Layered Architecture

```
┌─────────────────────────────────────────┐
│ LAYER 0: OPTIMISTIC EXECUTION           │
│ 100K+ TPS, <10ms latency                │
│ Centralized or small committee first    │
│ (Game loop needs speed, not ZK yet)     │
└─────────────────────────────────────────┘
 │
 ▼
┌─────────────────────────────────────────┐
│ LAYER 1: BATCHING + MERKLE              │
│ Aggregate actions into batches          │
│ Cryptographic commitment, not proof     │
│ ~25M TPS                                │
└─────────────────────────────────────────┘
 │
 ▼
┌─────────────────────────────────────────┐
│ LAYER 2: ZK PROVING (periodic)          │
│ Every 1000 ticks, prove state valid     │
│ 938ms acceptable for settlement         │
│ (Not real-time, just audit trail)       │
└─────────────────────────────────────────┘
```

### Performance by Layer

| Layer | TPS | Latency | Use Case |
|-------|-----|---------|----------|
| L0 Optimistic | 100K+ | <10ms | Active gameplay |
| L1 Merkle | 25M | ~1ms | Batch commits |
| L2 ZK | ~4K | 938ms | Settlement/audit |

### Why This Works
- Game loop doesn't need ZK (too slow for real-time)
- ZK is for settlement finality, not real-time
- 938ms is acceptable for periodic checkpoints

### Hardware Baseline
- **Mac mini M4**: Proof of concept baseline (~4K TPS ZK)
- Not ceiling: GPU + cluster = million+ TPS

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
