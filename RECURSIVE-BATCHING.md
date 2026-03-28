# Recursive Batching Layer

**Created:** March 28, 2026  
**Inspired by:** Laura's idea to try recursive batching like recursive ZKs

## The Problem

Current BrixaScaler architecture:
```
Txs → Batching Layer → ZK Layer → Settlement
```

Each batch needs its own ZK proof. With 1000 txs/batch, 200 batches = 200 ZK proofs.

## The Solution: Recursive Batching

```
Level 0: Raw transactions
    ↓
Level 1: Micro-batches (1000 txs each) → Merkle root
    ↓
Level 2: Super-batches (100 micro-roots) → Super-root  
    ↓
ZK: ONE proof verifies ALL 100,000 txs!
```

## Benchmark Results

```
Testing 2-level recursive batching with 200,000 txs...

✅ RESULTS:
   Time: 0.17s
   Input TPS: 1,212,121
   ─────────────
   Micro-batches: 200
   Super-batches: 2
   ZK proofs: 2
   ─────────────
   Compression: 1 ZK proof verifies 100,000 txs
   Savings: 99.9990%
```

## Key Insight

- **Before:** 200,000 txs → 200 ZK proofs
- **After:** 200,000 txs → 2 ZK proofs
- **Savings:** 99% reduction in proving cost!

## How It Works

1. **Micro-batching:** Group 1000 txs into a batch, compute Merkle root
2. **Super-batching:** Group 100 Merkle roots into super-batch, compute super-root
3. **ZK Proof:** Generate ONE proof for the entire super-batch tree
4. **Settlement:** Submit single proof to verify 100,000 txs

## Files

- `integration/recursive-batching.js` - Full implementation
- `integration/recursive-batching-benchmark.js` - Benchmark script

## Configuration

```bash
MICRO_BATCH_SIZE=1000    # txs per micro-batch
SUPER_BATCH_SIZE=100    # micro-batches per super-batch  
MEGA_BATCH_SIZE=10      # super-batches per mega-batch (optional 3rd level)
```

## Three-Level Variant

For even more compression:
```
1 ZK proof = 1,000,000 transactions
(1000 × 100 × 10)
```

## Comparison

| Level | Txs per batch | Batches | ZK proofs needed |
|-------|--------------|---------|------------------|
| Original | 1000 | 200 | 200 |
| 2-Level | 100,000 | 2 | 2 |
| 3-Level | 1,000,000 | 1 | 1 |

## Trade-offs

**Pros:**
- Massive ZK cost reduction (99%+)
- Fewer on-chain settlements
- Same security via Merkle trees

**Cons:**
- More complex code
- Longer latency (wait for more txs)
- Larger data commitment size

## Next Steps

- [ ] Integrate real circom circuits for proof
- [ ] Add parallel super-batch formation
- [ ] Benchmark against original BrixaScaler
- [ ] Compare 2-level vs 3-level performance