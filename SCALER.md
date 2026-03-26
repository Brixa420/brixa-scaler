# Brixa Scaler

Network routing and TPS layer.

## Full Architecture (March 26, 2026)

### Layer 1: Horizontal Proving Layer
- `execution/horizontal-prover.js` - 100+ GPU nodes
- 50 proofs/sec per prover × 100 provers = 5,000 proofs/sec
- 100K txs/proof = **500M TPS** proving bandwidth

### Layer 2: Recursive Proof Compression
- `execution/recursive-compressor.js` - Halo2/Nova style
- 1,000 child proofs → 1 parent proof
- 500M → **500K settlement units/sec**

### Layer 3: Sharded Settlement
- `execution/sharded-settlement.js` - 10 parallel chains
- 50K TPS per shard × 10 shards = **500K TPS finality**
- Unified state via recursive bridges

### Critical Interfaces
- `execution/interfaces.js` - BatcherInput, ProofBundle, SettlementConfig

## NOT a Blockchain

Brixa Scaler is NOT a blockchain. It is chain-agnostic.
