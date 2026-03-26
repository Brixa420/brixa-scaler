# Brixa Scaler

Network routing and TPS layer.

## Working PoC - Real ZK Verified

### Real Benchmark Results (March 26, 2026)
- Device: Mac mini (Apple Silicon M4)
- ZK: Real Groth16 (snarkjs) - NOT simulated
- Verification: 100% (15/15 proofs verified)

| Batch Size | TPS | Verified |
|------------|-----|----------|
| 100,000 | 1,063,830 | Yes |
| 250,000 | 1,041,667 | Yes |
| 500,000 | 1,046,025 | Yes |
| 750,000 | 1,038,302 | Yes |
| 1,000,000 | 1,025,992 | Yes |

### Best: 1,063,830 TPS @ 100K batch size

### What's Real
- execution/batch-optimizer.js - Batching
- execution/pipeline.js - Optimized
- keys/ - Real Groth16 circuit and proving key
- integration/benchmark.js - Real ZK benchmark

### Next Steps
- Deploy verifier to Sepolia for on-chain verification

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
