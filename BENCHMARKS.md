# BrixaScaler Benchmark Results

## Measured Performance (Apple M4 10-core)

### Layer 1: Transaction Batching - MERKLE TREE

#### Single-Shard (baseline)
| Batch Size | Time | Throughput |
|------------|------|------------|
| 1,000 | 97µs | 10.3M TPS |
| 10,000 | 963µs | 10.4M TPS |
| 100,000 | 9.3ms | 10.7M TPS |
| 1,000,000 | 83.8ms | 11.9M TPS |

#### Sharded + Parallel (10 cores)
| Batch Size | Shards | Time | Throughput | Improvement |
|------------|--------|------|------------|-------------|
| 1,000,000 | 10 | 44.5ms | 22.5M TPS | 1.9x |
| 5,000,000 | 10 | 206.6ms | 24.2M TPS | 2.0x |
| 10,000,000 | 10 | 426ms | 23.5M TPS | 2.0x |
| 10,000,000 | 20 | 394ms | 25.4M TPS | 2.1x |

*Method: SHA256 hash + Merkle tree construction (Go, 10 parallel goroutines)*

---

### Key Findings

| Mode | Peak TPS |
|------|----------|
| Single-shard | **11.9M TPS** |
| Sharded (10) | **25.4M TPS** |
| **Speedup** | **2.1x** |

---

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

| Layer | Peak Throughput | Implementation |
|-------|-----------------|-----------------|
| Batching (sharded) | **25.4M TPS** | 10 parallel goroutines |
| ZK Proving | 2.6 TPS | Single Groth16 prover |

**Solution: Validator Network** - N validators = N × 2.6 TPS proving capacity.

---

## Sharding Architecture

```
┌─────────────────────────────────────────────────────────┐
│              VALIDATOR COORDINATOR                       │
│         (goroutine per shard, 10 cores)                 │
└─────────┬─────────┬─────────┬─────────┬─────────────────┘
          │         │         │         │
    ┌─────▼─────┐┌──▼──┐┌─────▼─────┐┌──▼──┐
    │ Shard 0   ││Shard││ Shard 9  ││ ... │
    │(parallel) ││ 1   ││(parallel) ││     │
    └─────┬─────┘└─┬───┘└─────┬─────┘└─┬───┘
          │        │         │        │
          ▼        ▼         ▼        ▼
    ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
    │ Root 0  │ │ Root 1  │ │ Root 9  │ │   ...   │
    └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘
         └───────────┴────┬─────┴───────────┘
                          ▼
                   ┌─────────────┐
                   │  SUPER ROOT │
                   │(Merkle of   │
                   │ shard roots)│
                   └──────┬──────┘
                          ▼
                   ┌─────────────┐
                   │   ZK PROOF   │
                   │ (validator)  │
                   └─────────────┘
```

---

## Running Benchmarks

```bash
# Merkle tree benchmark (Go)
cd integration/go && go run benchmark_merkle.go

# Sharded merkle (existing)
go run sharded-merkle.go

# Two-layer (JS + ZK)
node ../benchmark-two-layer.js
```
