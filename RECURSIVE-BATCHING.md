# ZK Recursive Batching - Status

## The Concept

Recursive batching means:
1. **Level 1**: Batch 16 txs → 1 batch proof
2. **Level 2**: Aggregate 8 batch proofs → 1 super proof
3. **Result**: 1 super proof = 128 txs validated!

## Verification Comparison

| Approach | # of On-Chain Verifications | Txs per Verify |
|----------|----------------------------|----------------|
| Non-recursive | 8 (for 128 txs) | 16 |
| **Recursive** | **1** (for 128 txs) | **128** |

**Savings: 87.5% reduction in verification costs!**

## Current Status

gnark proving works at ~3 proofs/sec (330ms/proof) for 4-tx batches. The recursive batching would aggregate multiple batch proofs into one super proof.

## Implementation Notes

gnark v0.14.0 doesn't support native recursive proofs (verifying a proof inside a circuit). Options:
1. **Nova/IVC** - Use the Nova library for incremental computation
2. **Simulated aggregation** - Hash multiple proofs together, prove knowledge
3. **PLONK recursion** - Use PLONK with recursion-friendly setup

The current batch_verifier.go on port 4111 generates individual batch proofs. To add recursive aggregation, we'd need a custom aggregation circuit that verifies multiple batch roots in one constraint system.

## Files
- `zk-gnark/batch_verifier.go` - Current batch prover (port 4111)
- `zk-gnark/fixed_verify.go` - Verification benchmark