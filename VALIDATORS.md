# BrixaScaler Validator Network

## From Prover-Centric to Validator Model

### Current (Single Prover Bottleneck)
```
Batching Layer → 1 Prover (2.6 TPS) → Settlement
                      ↑
                   Bottleneck
```

### Proposed (Validator/Shard Network)
```
                    ┌─ Validator 1 (shard 1) ─┐
                    ├─ Validator 2 (shard 2) ─┤
Batching Layer ────→ Split ── Validator 3 (shard 3) ──→ Aggregate → Settlement
                    ├─ Validator 4 (shard 4) ─┤
                    └─ Validator N (shard N) ─┘
```

---

## Why Validator Model Wins

| Aspect | Single Prover | Validator Network |
|--------|---------------|-------------------|
| Throughput | 2.6 TPS | N × 2.6 TPS |
| Trust | Single entity | Distributed |
| Fault tolerance | SPoF | Byzantine fault tolerant |
| Economics | No incentives | Staking + rewards |
| Scalability | Hardware bound | Add validators |

---

## Architecture

### Shard Assignment
```go
// Each transaction batch is assigned to a validator shard
shardID := batchRoot % numValidators
```

### Validator Network
```go
type Validator struct {
    ID        string
    Stake     uint64
    ShardIDs  []uint32
    IsActive  bool
}

type ValidatorNetwork struct {
    Validators map[string]*Validator
    Quorum     int  // Required for consensus
}
```

### Proof Aggregation
```
Validator 1: Proof₁ (validates shards 0,5,10...)
Validator 2: Proof₂ (validates shards 1,6,11...)
Validator 3: Proof₃ (validates shards 2,7,12...)
...
Validator N: Proofₙ

→ Recursive aggregation → Final proof → Settlement
```

---

## Design Decisions

### Shard Assignment
- **Random**: Round-robin across validators
- **Stake-weighted**: More stake = more shards
- **Periodic rotation**: Epoch-based shuffling for fairness

### Proof Aggregation
- Uses **recursive SNARKs** (Circuit B from your design)
- Each validator submits proof for their shard range
- Aggregator combines N proofs into one final proof

### Validator Rewards
- **Batch fee**: Per-transaction fee
- **Stake yield**: APY on staked tokens
- **Slashing**: Penalty for invalid proofs

### Slashing Conditions
1. **Invalid proof**: Submitted proof fails verification
2. **Downtime**: Missing more than X consecutive blocks
3. **Equivocation**: Double-signing same batch

---

## Implementation

### Validator Registration
```bash
# Register as a validator
brixascaler validator register --stake 10000

# View validator status
brixascaler validator status
```

### Running a Validator
```bash
# Start validator node
brixascaler validator run --shards 4 --prover-type groth16
```

### Validator Config
```json
{
  "validator": {
    "stake": 10000,
    "shards": [0, 1, 2, 3],
    "prover": {
      "type": "groth16",
      "circuit": "batch_merkle",
      "parallel": 4
    }
  },
  "network": {
    "peers": ["validator1:9000", "validator2:9000"],
    "quorum": 3
  }
}
```

---

## Economic Security

### Staking
- Minimum stake: 10,000 BRIX
- Slashing: 50% of stake for invalid proof
- Unbonding period: 7 days

### Rewards
- Per batch: 0.001 BRIX per transaction
- Block reward: Proportional to shards assigned
- Honest participation: No slashing = stake yield

---

## Fork Choice

### Honest Validator Behavior
1. Receive batch → verify transactions
2. Generate proof for assigned shards
3. Submit proof to aggregator
4. Receive final proof → verify and sign

### Malicious Validator
1. Submit invalid proof → Slashed! 🚫
2. Go offline → Slashed! 🚫
3. Double-submit → Slashed! 🚫

---

## Why This Beats Single Prover

```
Single Prover:     2.6 TPS (fixed by hardware)
Validator Network: N × 2.6 TPS (scales with N)
                          ↑
                    Add more validators!
```

**10 validators = 26 TPS**
**100 validators = 260 TPS**
**1,000 validators = 2,600 TPS** ← beats batching layer!

This turns your ZK "bottleneck" into a **decentralized, economically secure prover network** - arguably better than a single optimized prover.
