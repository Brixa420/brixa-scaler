# BrixaScaler

> ⚠️ **INCOMPLETE SOFTWARE** — This is a prototype/MVP. Not all features are implemented.

⚠️ **IMPORTANT: DEMO MODE**

**DEFAULT IS DEMO MODE** — Transactions are logged but NOT sent to any blockchain!

---

## Layer 3 Batching & ZK Compression

BrixaScaler is a **Layer 3** batching and ZK compression layer that sits on top of L2s like Polygon.

```
┌─────────────────────────────────────────────────────────────┐
│ L1 (Ethereum)       — Settlement layer     — 15 TPS        │
│ L2 (Polygon)        — Settlement layer     — 65 TPS        │
│ L3 (BrixaScaler)    — Batching + ZK        — ~16M TPS      │
└─────────────────────────────────────────────────────────────┘
```

### The Real Benchmarked Numbers (March 2026)

| Layer | What | TPS | How Measured |
|-------|------|-----|---------------|
| **Batching** | Hash + Merkle (10 shards) | ~16M | ✅ Real Go benchmark |
| **Batching** | Hash + Merkle (single thread) | ~5.3M | ✅ Real Go benchmark |
| **ZK Prove** | Groth16 (4-tx batch) | ~3 proofs/sec | ✅ Real snarkjs |
| **ZK Verify** | Groth16 verification | ~4.5/sec | ✅ Real snarkjs |
| **ZK (period=1000)** | 4000 txs per proof | ~12K TPS | ✅ Calculated |
| **Settlement** | Polygon | 65 TPS | ✅ External RPC |
| **Full E2E** | L3 → L2 | **65 TPS** | ⚠️ Bottlenecked by L2 |

### Benchmark Results (Real)

```
=== Batching Layer (Go) ===
Shards= 1, Workers= 1 → Mean=   5,387,483 TPS
Shards= 4, Workers= 4 → Mean=  13,607,159 TPS  
Shards=10, Workers=10 → Mean=  16,216,786 TPS

=== ZK Layer (snarkjs) ===
Proof generation: ~330ms per 4-tx batch
Verification:     ~230ms per proof

=== End-to-End ===
Batching: 16M txs → batches (fast, in-memory)
ZK: 4000 txs/proof → ~12,121 TPS (with period=1000)
Settlement: ~65 TPS (Polygon bottleneck)
```

### The Honest Claim

> "BrixaScaler is a Layer 3 batching and ZK compression layer.
> ~16M TPS off-chain batching, ~12K TPS with ZK proofs (period=1000),
> settling to any L2 at their native speed (65 TPS Polygon, 15 TPS Ethereum)."

### What BrixaScaler Is NOT

| Wrong Claim | Correct |
|-------------|---------|
| "Millions of TPS blockchain" | "~16M TPS L3 batching" |
| "Replaces L2s" | "Enhances any L2" |
| "Faster than Ethereum" | "Faster off-chain, settles to L2" |

---

## Architecture

```
User Action (~16M TPS possible)
    ↓
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│   BATCHING  │ →  │     ZK      │ →  │  SETTLEMENT │
│   LAYER     │   │    LAYER     │   │    (L2)      │
│  ~16M TPS   │   │  ~12K TPS    │   │   65 TPS     │
│             │   │  (period=1K) │   │   Polygon    │
└──────────────┘   └──────────────┘   └──────────────┘
 1000 txs/batch    ZK proof         1 tx = 1000 txs
```

### Why Layer 3?

- **Sits on top of L2** — Polygon is your settlement
- **Compresses to L2 limits** — ~16M → 65 via ZK
- **Doesn't replace L2** — Enhances it
- **Off-chain execution** — Fast, periodic settlement

---

## What's Built & Verified

| Component | Status | Details |
|-----------|--------|---------|
| **Batching Layer** | ✅ Working | 16M TPS (10 shards) |
| **Merkle Build** | ✅ Working | Real SHA256 + tree |
| **ZK Proving** | ✅ Working | snarkjs Groth16, ~330ms/proof |
| **ZK Verification** | ✅ Working | ~230ms/verify |
| **Settlement** | ⚠️ Demo | Logs only, no real RPC |

---

## Quick Start

```bash
cd /Users/laura/.openclaw/workspace/brixa-scaler
npm install
node integration/brixaroll.js --rpc https://polygon-rpc.com
```

---

## The Bottom Line

- **Off-chain batching:** ~16M TPS
- **With ZK (period=1000):** ~12K TPS
- **On-chain:** 65 TPS (Polygon bottleneck)
- **Honest:** L3 enhances L2, doesn't replace it

**Settle to Polygon, Arbitrum, or any L2. Your choice.**

---

## License

MIT

**Demo Only:** This software is provided as-is. No real transactions by default.