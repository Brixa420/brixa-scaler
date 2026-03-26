# Brixa Scaler

Network routing and TPS layer.

## Working ZK Pipeline (March 26, 2026)

We built a new circuit from scratch with working ZK:

### Components Working
| Component | Status | Speed |
|-----------|--------|-------|
| Circuit (batch_merkle.circom) | ✅ Compiled | - |
| Trusted setup | ✅ Done | - |
| Witness generation | ✅ Works | 250ms |
| ZK Prove | ✅ Works | 358ms |
| ZK Verify | ✅ Works | 340ms |

### Full Pipeline
```
1. Build Merkle tree from transactions
2. Extract proof path
3. Generate witness (snarkjs wc)
4. Generate proof (snarkjs g16p)  
5. Verify (snarkjs g16v)
```

### Files Created
- batch_merkle.circom - Circuit source (simple addition-based)
- batch_merkle.r1cs, .wasm, .zkey - Compiled
- batch_vk.json - Verification key
- proof_test.json - Working proof example

### Next Steps
- Replace simple addition hash with Poseidon
- Scale to more levels
- Add to benchmark

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
