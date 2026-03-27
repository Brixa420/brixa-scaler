# BrixaScaler - Benchmark Report (Honest Assessment)

## What's Real

| Layer | Throughput | Status | Notes |
|-------|------------|--------|-------|
| Batching | **2,850,000 TPS** | ✅ Real | Go + SHA256 + Merkle |
| ZK Proving | **~800 TPS** | ✅ Real | gnark (3-constraint circuit) |
| circom/snarkjs | **~2.5 TPS** | ✅ Real | Verification only |

## What's Theoretical/Simulated

| Layer | Claimed | Reality |
|-------|---------|---------|
| ZK Proving | 337K TPS | ❌ Not measured |
| Settlement | 1 tx/10M txs | ❌ Not verified on-chain |
| Compression | 1000:1 | ❌ Math only |

**Note:** The 337K TPS was calculated as `batchSize / time`, which is wrong. ZK proving time doesn't scale inversely with batch size.

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        BRIXA SCALER                                      │
├─────────────────────────────────────────────────────────────────────────┤
│  Transactions (10M)                                                      │
│      │                                                                  │
│      ▼                                                                  │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────────────────┐    │
│  │   Batching  │────▶│  ZK Prover  │────▶│  Recursive Aggregation  │    │
│  │  2,850,000  │     │    ~800     │     │     (theoretical)      │    │
│  │    TPS      │     │    TPS      │     │                        │    │
│  └─────────────┘     └─────────────┘     └───────────┬─────────────┘    │
│                                                        │                │
│                                                        ▼                │
│                                              ┌─────────────────────┐    │
│                                              │  Settlement Proof   │    │
│                                              │   (not verified)    │    │
│                                              └─────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Real Measurements

### Layer 1: Transaction Batching
| Transactions | Time | Throughput |
|--------------|------|------------|
| 100,000 | 32ms | 3,135,632 TPS |
| 1,000,000 | 334ms | 2,997,251 TPS |
| 10,000,000 | 3.5s | 2,851,371 TPS |

**Method:** SHA256 + Merkle tree (multi-core Go)
**Status:** ✅ Real, reproducible

### Layer 2: ZK Proving (gnark)
```
100 proofs: 125ms = 800 TPS
```
- Circuit: 3 constraints (trivial)
- Backend: Groth16, BN254
- Status: ✅ Real proof generation

### circom/snarkjs
- Verification: ~400ms per proof
- Throughput: ~2.5 TPS
- Status: ✅ Works but slow on CPU

---

## 📁 Project Structure

```
brixa-scaler/
├── benchmark/
│   └── benchmark.go          # Batching layer benchmark
├── zk-gnark/
│   ├── recursive_aggregation.go  # Aggregation pipeline
│   ├── batch_circuit.go      # gnark circuit
│   └── nova_folding.go       # Nova folding
├── zk/
│   └── real_prover.js        # circom/snarkjs prover
└── BENCHMARKS.md             # This file
```

---

## 🔑 Honest Assessment

1. **Batching is fast** - 2.85M TPS is real and reproducible
2. **ZK is slow** - ~800 TPS on CPU with gnark, ~2.5 TPS with circom
3. **337K TPS claim is wrong** - Was calculated incorrectly
4. **Settlement not verified** - No actual on-chain test yet

---

## ✅ What's Complete

- [x] Batching layer (2.85M TPS real)
- [x] ZK circuits compile (gnark + circom)
- [x] Real proof generation works
- [ ] High-throughput ZK (needs GPU)
- [ ] On-chain settlement verification

---

## 🎯 Path Forward

1. **Fix circom compiler** (needs Node 18)
2. **Add GPU proving** (10-100x faster)
3. **Design real batch circuit** (not trivial)
4. **Test on actual L1/L2** (not simulated)

---

## Disclaimer

The 337K TPS ZK claim in earlier versions was incorrect. Real ZK proving is much slower on CPU hardware. The batching layer performance is the only number that's been properly measured and verified.