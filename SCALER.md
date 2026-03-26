# Brixa Scaler

Network routing and TPS layer.

## Current Status: PoC / Design Document

### Working (Validated)
- execution/batch-optimizer.js - Batching (11K txs)
- execution/pipeline.js - Pipeline architecture
- execution/interfaces.js - API definitions
- execution/gpu-prover.js - Optimized with parallel CPU workers

### Benchmark Results (March 26, 2026)
- Proving: 43 proofs/sec (21x improvement from parallel workers)
- Batching: 11K txs/batch
- Pipeline: ~5K TPS

### Aspirational
- GPU proving (needs NVIDIA hardware for 20x more)

## NOT a Blockchain

Brixa Scaler is NOT a blockchain. It is chain-agnostic.
