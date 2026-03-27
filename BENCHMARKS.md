# BrixaScaler Benchmark Results

## Measured Performance (Apple M4 10-core)

### Layer 1: Transaction Batching
| Batch Size | Time | Throughput |
|------------|------|------------|
| 10,000 | 8ms | 1.2M TPS |
| 100,000 | 70ms | 1.4M TPS |
| 1,000,000 | 600ms | 1.5M TPS |

*Method: SHA256 + Merkle tree construction (single-threaded, in-memory)*

### Layer 2: ZK Proof Generation
| Protocol | Prove | Verify | Trusted Setup |
|----------|-------|--------|---------------|
| Groth16 | 385ms | 272ms | Per-circuit |
| PLONK | 1806ms | 269ms | Universal |

### Combined Two-Layer System
| Metric | Value |
|--------|-------|
| Batch (16 txs) | 387ms |
| Effective TPS | 41 tx/sec |

---

## Architecture Analysis

### The Throughput Gap

| Layer | Peak Throughput | Notes |
|-------|-----------------|-------|
| Batching | 1.5M TPS | Single-threaded SHA256 |
| ZK Proving | 2.6 TPS | Single Groth16 prover |
| Combined | 41 TPS | Batching compressed by ZK |

**Ratio: ~577,000x** - Batching vastly outpaces proving.

### Why This Is By Design

```
Ingest: 1.5M TPS (burst capacity)
Prove:  ~3 TPS (steady state)
Queue:  Builds during bursts, drains during lulls
```

This is exactly how Visa, Kafka, and any queue-based system work.

### Scaling Strategy: Parallel Provers

```
500 provers × 2.6 TPS = 1,300 TPS proving capacity
```

Even with 500 provers, batching is ~1,000x faster. This is **headroom**, not a problem.

---

## Implementation Notes

- Batching uses `crypto/sha256` (assembly-optimized on Apple Silicon)
- Merkle tree is single-threaded (parallel version available)
- ZK uses Groth16 with beacon-secured trusted setup
- Universal PLONK setup available (reusable, no new ceremony)

---

## Running Benchmarks

```bash
# Layer 1: Batching only
node integration/bench_zk.js

# Two-layer benchmark (requires poseidon-lite)
node integration/benchmark-two-layer.js
```
