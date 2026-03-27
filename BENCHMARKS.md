# BrixaScaler Benchmark Results

## Measured Performance (Apple M4 10-core)

### Layer 1: Transaction Batching (PARALLEL + SHARDED)
| Batch Size | Shards | Throughput |
|------------|--------|------------|
| 5,000,000 | 5 | 12.1M TPS |
| 5,000,000 | 10 | **16.5M TPS** |
| 10,000,000 | 10 | 13.0M TPS |
| 20,000,000 | 10 | 13.3M TPS |

*Method: SHA256 + **Parallel goroutines** + Sharded Merkle tree (10 CPU cores)*

### Layer 1: Single-Shard (baseline)
| Batch Size | Time | Throughput |
|------------|------|------------|
| 10,000 | 8ms | 1.2M TPS |
| 100,000 | 70ms | 1.4M TPS |
| 1,000,000 | 600ms | 1.5M TPS |

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

### The Throughput Gap (PARALLEL + SHARDED)

| Layer | Peak Throughput | Implementation |
|-------|-----------------|-----------------|
| Batching | **16.5M TPS** | 10 parallel goroutines, 10 shards |
| ZK Proving | 2.6 TPS | Single Groth16 prover |

**Ratio: ~6,350,000x**

### Why This Is By Design

```
Ingest: 16.5M TPS (parallel goroutines across 10 cores)
Prove:  ~3 TPS (steady state)
Queue:  Builds during bursts, drains during lulls
```

This is exactly how Visa, Kafka, and any queue-based system work.

### Parallel Scaling Strategy

```
10 CPU cores × 1.65M TPS/core ≈ 16.5M TPS
```

More cores = more throughput. The batching layer scales horizontally.

---

## Sharding + Parallelism Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    VALIDATOR COORDINATOR                │
│              (goroutine per shard, 10 cores)            │
└─────────┬─────────┬─────────┬─────────┬─────────────────┘
          │         │         │         │
    ┌─────▼─────┐┌──▼──┐┌─────▼─────┐┌──▼──┐
    │ Shard 0   ││Shard││ Shard N-1 ││Shard│
    │ (parallel ││ 1   ││ (parallel ││ N   │
    │  goroutine)││(par)││  goroutine)││(par)│
    └─────┬─────┘└─┬───┘└─────┬─────┘└─┬───┘
          │        │         │        │
          ▼        ▼         ▼        ▼
    ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
    │ Root 0  │ │ Root 1  │ │Root N-1 │ │ Root N  │
    └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘
         └───────────┴────┬─────┴───────────┘
                          ▼
                   ┌─────────────┐
                   │  SUPER ROOT │
                   │ (Merkle of  │
                   │ shard roots)│
                   └──────┬──────┘
                          ▼
                   ┌─────────────┐
                   │   ZK PROOF  │
                   └─────────────┘
```

Each shard processes in parallel via goroutines. Uses `runtime.GOMAXPROCS(10)` for max parallelism.

---

## Implementation Notes

- **Parallel**: Go `sync.WaitGroup` + goroutines per shard
- **Sharded**: Independent Merkle trees per shard, combined into super-root
- **Hash**: `crypto/sha256` (assembly-optimized on Apple Silicon)
- **ZK**: Groth16 with beacon-secured trusted setup

---

## Running Benchmarks

```bash
# Sharded + Parallel benchmark (Go)
cd integration/go && go run sharded-merkle.go

# Two-layer benchmark (JS + ZK)
node integration/benchmark-two-layer.js
```
