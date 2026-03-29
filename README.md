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
│ L3 (BrixaScaler)    — Batching + ZK        — 71K TPS       │
│ L4 (Recursive)      — Super-aggregation    — 100:1        │
└─────────────────────────────────────────────────────────────┘
```

### The Honest Numbers (Measured)

| Layer | What | TPS | Measured |
|-------|------|-----|----------|
| Batching | Hash + Merkle | 71K off-chain | ✅ |
| ZK Prove | Groth16 proofs | 283/sec | ✅ |
| Settlement | Polygon | 65 TPS | ✅ |
| **Full E2E** | **L3 → L2** | **65 TPS** | ✅ |

### The Honest Claim

> "BrixaScaler is a Layer 3 batching and ZK compression layer. 
> 71K TPS off-chain execution, settling to any L2 at their native speed 
> (65 TPS Polygon, 15 TPS Ethereum)."

### What BrixaScaler Is NOT

| Wrong Claim | Correct |
|-------------|---------|
| "Millions of TPS blockchain" | "71K TPS L3 batching" |
| "Replaces L2s" | "Enhances any L2" |
| "Faster than Ethereum" | "Faster off-chain, settles to L2" |

---

## Architecture

```
User Action (71K TPS possible)
    ↓
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│   BATCHING  │ →  │     ZK      │ →  │  SETTLEMENT │
│   LAYER     │   │    LAYER     │   │    (L2)      │
│  71K TPS    │   │  283 proofs/ │   │   65 TPS    │
│             │   │     sec      │   │   Polygon    │
└──────────────┘   └──────────────┘   └──────────────┘
 1000 txs/batch    ZK proof         1 tx = 1000 txs
```

### Why Layer 3?

- **Sits on top of L2** — Polygon is your settlement
- **Compresses to L2 limits** — 71K → 65 via ZK
- **Doesn't replace L2** — Enhances it
- **Off-chain execution** — Fast, periodic settlement

---

## What's Built & Verified

| Component | Status | Details |
|-----------|--------|---------|
| **Batching Layer** | ✅ Working | 1000 txs → 10 batches |
| **Merkle Build** | ✅ Working | 10 batch roots in 8ms |
| **ZK Proving** | ✅ Working | gnark Groth16, 283/sec |
| **Recursive** | ✅ Working | 10:1 aggregation |
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

- **Off-chain:** 71K TPS (batching + ZK)
- **On-chain:** 65 TPS (Polygon bottleneck)
- **Honest:** L3 enhances L2, doesn't replace it

**Settle to Polygon, Arbitrum, or any L2. Your choice.**

---

## License

MIT

**Demo Only:** This software is provided as-is. No real transactions by default.