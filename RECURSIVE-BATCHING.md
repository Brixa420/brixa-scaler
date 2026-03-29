# Recursive Batching - REMOVED

As of March 28, 2026, recursive batching has been **removed from the system**.

## Why Removed
- Verification was broken (couldn't verify proofs)
- Could not test if recursive batching actually worked
- Claimed numbers were inflated and unverified

## What's Left
- Full system benchmark: ~850 TPS
- ZK Prove: ~2,100/sec
- ZK Verify: ~1,500/sec
- Merkle: ~1M+ leaves/sec

## Files Removed
- `integration/recursive-batching.js`
- `integration/recursive-batching-gnark.js`
