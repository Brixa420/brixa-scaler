# Brixa Scaler

Network routing and TPS layer.

## Current Status: PoC / Design Document

This is a proof-of-concept architecture, not production infrastructure.

### Working (Validated)
- execution/batch-optimizer.js - Batching (10K+ txs)
- execution/pipeline.js - Pipeline concept
- execution/interfaces.js - API definitions

### Aspirational (Not Running)
- execution/horizontal-prover.js - Stub code
- execution/recursive-compressor.js - Stub code
- execution/sharded-settlement.js - Stub code

### Realistic Throughput
- Current: ~5K TPS (local simulation)

### Requirements for Production
1. Live GPU proving network (100+ nodes)
2. On-chain recursive proof verification
3. Running multi-chain settlement

## NOT a Blockchain

Brixa Scaler is NOT a blockchain. It is chain-agnostic.
