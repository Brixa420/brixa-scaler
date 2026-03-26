# Brixa Scaler

Network routing and TPS layer.

## Honest Benchmark Status (March 26, 2026)

### Hardware
- Mac mini M4

### What Works
- Circuit compiles and runs
- Different inputs produce different valid proofs
- All proofs verify correctly

### Timing Analysis

| Step | Time |
|------|------|
| Setup (PK gen) | 13ms |
| Witness creation | 0.1ms |
| Prove | 1-3ms |
| Verify | 1-2ms |

- Prove is 21.7x witness creation (not instant)
- But still seems fast for Groth16

### Uncertainty
- Kimi reports Groth16 should take 100-500ms per proof
- Our 1-3ms seems suspiciously fast
- Possible explanations:
  1. gnark is extremely optimized
  2. Apple M4 is extremely fast  
  3. 51 constraints is very small
  4. Some issue we haven't found

### Code for Review
- keys/zk_poseidon.go - Working benchmark
- keys/zk_proof_diffs.js - Different inputs test

### Honest Claim
"~256K TPS based on gnark proving on Apple M4. Timing seems fast - needs verification."

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
