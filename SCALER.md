# Brixa Scaler

**Truthful documentation - what has been tested, what hasn't.**

## What This Is

- A **proof-of-concept** ZK proving system for batch verification
- **NOT a blockchain** - no consensus, no blocks, no chain
- **NOT deployed anywhere** - local testing only

## What Has Been TESTED (Real Results)

### Environment
- **Hardware:** Apple M4 Mac mini
- **Software:** Go + gnark v0.14.0
- **Test date:** March 26, 2026

### Benchmark Results

| Config | Constraints | Parallel | Latency | TPS |
|--------|-------------|----------|---------|-----|
| 6 levels, 5K batch | 155K | 4 | 1.0s | **19,467** |
| 8 levels, 5K batch | 205K | 4 | 1.3s | 15,143 |
| 10 levels, 10K batch | 510K | 1 | 1.6s | 13,656 |
| 20 levels, 10K batch | 1.01M | 1 | 1.6s | 6,339 |
| 32 levels, 10K batch | 1.61M | 1 | 3.1s | 3,219 |

### What This Measures
- ZK proof generation (witness + prove + verify)
- Single-machine, local testing only
- Batch of 5,000-10,000 hash verifications per proof
- NOT real transaction processing

## What Has NOT Been Tested

- ❌ Real network throughput
- ❌ Multiple machines
- ❌ Real transactions (only hash verification)
- ❌ Production deployment
- ❌ GPU acceleration (not available in gnark)
- ❌ Latency under load
- ❌ Actual Visa/merchant integration

## Known Limitations

1. **CPU-only:** gnark has no Metal GPU support (M4)
2. **Toy circuit:** Uses simple addition-based hash, not real cryptography
3. **Isolated testing:** No network simulation
4. **Single machine:** Horizontal scaling untested

## Honest Assessment

- ✓ Circuit compiles and produces valid proofs
- ✓ Verification passes
- ✓ Batching improves throughput
- ✗ No real TPS in production
- ✗ Not a blockchain
- ✗ Untested at scale

## Code Location

`keys/zk_6_levels_4.go` - Best tested configuration (19,467 TPS)
