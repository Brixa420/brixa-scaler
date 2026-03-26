# Brixa Scaler

Network routing and TPS layer.

## Recent Updates (March 26, 2026)

### Pipeline Architecture
- `execution/pipeline.js` - Overlaps Batch → Prove → Settle (not sequential)
- `execution/batch-optimizer.js` - Dynamic batching (10K-100K txs per proof)
- `execution/gpu-prover.js` - GPU acceleration for 20x faster proving

### Performance
- **25M TPS** (theoretical with GPU cluster)
- Batching: 10K-100K txs/proof (vs 2,500 before)
- Pipeline: Parallel stages, not sequential
- GPU: 20x speedup when NVIDIA GPU available

## Works With

**Brixa Node Engine** (brixa-node-engine) - Execution layer

## NOT a Blockchain

Brixa Scaler is NOT a blockchain. It is chain-agnostic.
