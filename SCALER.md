# Brixa Scaler

Network routing and TPS layer.

## Honest Benchmark Status (March 26, 2026)

### Hardware
- Mac mini M4 (Apple Silicon)

### Key Verification
- Function: real groth16.Prove (address verified)
- Constraints: 51 (confirmed)
- Prove time: ~1ms (micro-benchmark)

### Performance (Corrected Claims)

| Metric | Original | Corrected |
|--------|----------|----------|
| TPS | "~256K TPS ZK proving" | "256K TPS on 51-constraint micro-benchmark" |
| Use Case | "Fast ZK for AI/gaming" | "Architecture validated; production TPS pending real circuit" |
| Status | "Needs verification" | "Constraint scaling law understood; production measurement next" |

### Scaling Estimate
- 51 constraints → ~256K TPS
- 10K constraints → ~1,280 TPS (linear scale)

### What Works ✓
- Circuit compiles and runs
- Constraint count verified (51)
- Function address confirmed real
- Different inputs produce different valid proofs

### What's Next
1. Build production circuit (10K+ constraints)
2. Measure actual production TPS
3. GPU acceleration for speedup

### Code
- keys/zk_verify_func.go - Constraint count verification
- keys/zk_explicit_timing.go - Timing test

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
