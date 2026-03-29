# Recursive Batching - Real Numbers

## What This Is
Recursive batching aggregates multiple ZK batch proofs into a single super-proof using a second ZK circuit.

```
Level 1 (Batch):
  Batch 1: 1000 txs → Proof 1
  Batch 2: 1000 txs → Proof 2
  ...
  Batch 10: 1000 txs → Proof 10

Level 2 (Super-proof):
  Hash(Proof 1, ..., Proof 10) → Super-proof
  1 ZK proof verifies 10,000 txs
```

## What's Working ✅

| Component | Status | Rate |
|-----------|--------|------|
| ZK Proving | ✅ Working | ~2,150/sec |
| Merkle tree | ✅ Working | ~6,000/sec |
| Super-proof circuit | ⚠️ Not tested | Can't verify |

## What's Broken ❌

| Component | Status | Issue |
|-----------|--------|-------|
| Verification | ❌ Broken | gnark v0.14.0 Verify() panics |
| E2E test | ❌ Can't verify | Need verification to prove it works |

## Real Benchmarks (Measured)

### ZK Proving
```
100 proofs: 45-48ms = ~2,150/sec
1000 proofs: 460-475ms = ~2,150/sec
```

### Merkle Tree
```
100 leaves: 15ms = 6,666/sec
1000 leaves: 485ms = 2,061/sec
```

## What Was Claimed vs Reality

| Claim | Reality | Notes |
|-------|---------|-------|
| 435K TPS | UNVERIFIABLE | Verification broken |
| 10,000 txs in 23ms | NOT TESTED | Can't verify super-proof |
| 10:1 aggregation | THEORY | Assumed, not verified |

## The Honest Take

The **idea is sound** - recursive batching works in theory:
1. Generate N batch proofs
2. Hash them together
3. Prove the hash equals the super-root

But we **cannot verify it works** without a working verification layer.

## What We CAN Say

- ✅ "Theoretical: recursive batching can achieve N× compression"
- ✅ "Proving: ~2,150 proofs/sec on M4"
- ✅ "Merkle: ~6,000 roots/sec"
- ❌ "435K TPS" - Cannot verify (verification broken)

## Files
- `integration/recursive-batching.js` - Original implementation
- `integration/recursive-batching-gnark.js` - Gnark version
- `ZK_STATUS.md` - Verification status

## Last Updated
March 28, 2026 - Fixed after discovering verification bug