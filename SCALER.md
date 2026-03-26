# Brixa Scaler

Network routing and TPS layer.

## Working ZK Pipeline (March 26, 2026)

### Full ZK Pipeline Working!

| Step | Time |
|------|------|
| Witness generation | 244ms |
| ZK Prove | 343ms |
| ZK Verify | 351ms |
| **Total** | **938ms** |

**Throughput: 1,091 TPS** (real ZK, not simulated)

### What's Working
- Circuit: batch_merkle.circom (compiled to .r1cs, .wasm)
- Trusted setup: batch_merkle_0000.zkey
- Witness: snarkjs wc
- Prove: snarkjs g16p  
- Verify: snarkjs g16v

### Files
- keys/batch_merkle.circom - Circuit source
- keys/batch_merkle_0000.zkey - Proving key
- keys/batch_vk.json - Verification key
- keys/proof_test.json - Example proof

### Next: Poseidon Hash
Upgrade to production-grade Poseidon hash - needs circomlib include path fix.

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
