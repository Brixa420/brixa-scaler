# Brixa Scaler

Network routing and TPS layer.

## True Sharding Implementation (March 26, 2026)

### Architecture

```
┌─────────────────────────────────────────────┐
│           SHARDED PROVER                    │
├─────────────────────────────────────────────┤
│  Shard 0   │ Shard 1   │ ... │ Shard 9    │
│  3 validators    3 validators   3 validators │
│  4K TPS     │  4K TPS    │     │  4K TPS   │
└─────────────────────────────────────────────┘
           │
           ▼
    Aggregate: 40K+ TPS
```

### Components
- keys/shards/sharded-prover.js - True sharding implementation
- Route transactions to shards (consistent hashing)
- Each shard: independent validator set

### Performance

| Shards | Validators | Per Shard TPS | Aggregate TPS |
|--------|------------|---------------|---------------|
| 10 | 3 | 4,000 | **40,000+** |

### Notes
- Each shard runs independently
- Cross-shard transactions need bridging
- Shared security model (validators across shards)

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
