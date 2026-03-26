# Brixa Scaler

Network routing and TPS layer.

## Honest Status (March 26, 2026)

### What Works

| Component | Status | Speed |
|-----------|--------|-------|
| Merkle tree (SHA256) | ✅ Real | ~1M TPS |
| ZK Verify | ✅ Real | 63/sec, 16ms |

### What's Broken

**ZK Proving** - The circuit expects specific inputs from the original trusted setup
- proof.json was generated with inputs we don't have
- snarkjs.groth16.fullProve() fails: "Assert Failed" on MerkleTree line 21
- The circuit needs: leaf + root + valid pathElements + valid pathIndices
- We can verify the existing proof but cannot generate new ones

### To Fix ZK Proving

Option 1: Recompile circuit with new trusted setup
```bash
# New ptau
snarkjs ptn bn128 20 powersoftau_0000.ptau
# Compile circuit  
circom batch_merkle.circom --r1cs --wasm --sym
# New zkey
snarkjs groth16 setup batch_merkle.r1cs powersoftau_0000.ptau batch_merkle_0000.zkey
# Contribute  
snarkjs zkc batch_merkle_0000.zkey batch_merkle_final.zkey
```

Option 2: Use PLONK (no trusted setup required)
```bash
snarkjs pk setup batch_merkle.r1cs powersoftau.ptau batch_merkle.zkey
snarkjs pkp batch_merkle.zkey witness.wtns proof.json public.json
```

### Current Numbers

- Merkle build (100K): 90ms → 1.1M TPS
- ZK verify: 16ms → 63/sec
- ZK prove: FAILS

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
