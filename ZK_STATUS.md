# ZK Status - Real Numbers

## Working Components ✅

| Component | Rate | Status |
|-----------|------|--------|
| ZK Proving | ~2,150/sec | ✅ Working |
| ZK Verify | ~2,500/sec | ✅ Working |
| Merkle Tree | ~6,000/sec | ✅ Working |
| snarkjs (JS) | ~17/sec | ⚠️ Slower than gnark |

## Full System Benchmark (Go gnark)

The complete flow: Input → Merkle → Prove → Verify

| Batch Size | Time | TPS |
|------------|------|-----|
| 10 txs | 11ms | 909 |
| 100 txs | 114ms | 877 |
| 1000 txs | 1160ms | 862 |

**Average: ~850 TPS** (full prove + verify)

Note: This is with trivial A*B=C circuit (2 constraints). Real circuits will be slower.

## snarkjs vs gnark Comparison

| Backend | Verify Speed | Notes |
|---------|-------------|-------|
| Go gnark | ~850 TPS | **Much faster** - native Go |
| snarkjs (JS) | ~17 TPS | WASM overhead |

**gnark is ~50x faster** than snarkjs for verification!

## Comparison: Different Versions

Current: `github.com/consensys/gnark v0.14.0`

Available versions:
- v0.13.0, v0.12.0, v0.11.0, v0.10.0, v0.9.0, etc.

v0.14.0 is the latest stable - no need to downgrade.

## History

- March 28, 20:58 - Verified gnark is working at ~850 TPS
- March 28, 20:55 - snarkjs benchmark shows only ~17/sec (too slow)
- March 28, 20:25 - Verification fixed!
- Before: gnark Verify() panics with nil pointer
- Fix: Use separate PublicCircuit struct with Define() method

## Files

- `zk-gnark/fixed_verify.go` - Working verification benchmark
- `zk-gnark/batch_verifier.go` - HTTP server for proof generation
- `zk-snarkjs/verifier.js` - External snarkjs verifier (slower)