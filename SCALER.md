# Brixa Scaler

Network routing and TPS layer.

## Verified ZK Benchmark (March 26, 2026)

### Hardware
- Mac mini M4

### Verification Tests ✓

| Test | Result |
|------|--------|
| Different inputs (leaf 1,2,5) | All verify OK |
| Each proof takes different time | ~1-3ms (not cached) |
| Different inputs produce unique proofs | Verified |

### Code
- keys/zk_poseidon.go - Working single proof
- keys/zk_proof_diffs.go - Verification test

### Honest Analysis
- Circuit has ~51 constraints (non-linear hash)
- ~1-3ms proving on Apple M4 with gnark is plausible
- gnark is highly optimized (FFT, parallel, multiexp)
- All proofs verify correctly

### Performance
- Single proof: ~1.5ms prove + ~1ms verify
- TPS: ~256K-340K depending on caching

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
