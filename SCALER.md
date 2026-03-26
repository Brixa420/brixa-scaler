# Brixa Scaler

Network routing and TPS layer.

## Honest Status (March 26, 2026)

### What Actually Works

**Merkle Tree (Real)**
- SHA256 Merkle tree for 100K txs: ~90ms
- TPS: ~1.1M (just hashing)

**ZK Verification (Real, Verified)**
- snarkjs.groth16.verify(): 16ms average
- 63 verifications per second
- 10/10 successful verifications

### What Doesn't Work Yet

**ZK Proving (Fails)**
- snarkjs.groth16.fullProve() fails with "Assert Failed" on circuit
- The circuit (batch_merkle) expects specific Merkle path inputs
- Need correct leaf + root + pathElements + pathIndices
- The input.json was generated with different/broken inputs

### Honest Benchmark

| Component | Time | Works | Notes |
|-----------|------|-------|-------|
| Merkle 100K | 90ms | Yes | SHA256 hashing |
| ZK Prove | FAILS | No | Circuit input validation fails |
| ZK Verify | 16ms | Yes | 63 verifications/sec |

### To Get ZK Proving Working

1. Build actual Merkle tree from batch transactions
2. Extract correct path elements for the specific leaf
3. Format as field elements (BigInt in BN254)
4. Run snarkjs.groth16.fullProve()

The circuit is a Merkle verifier - it's not "broken", just needs correct inputs from a real tree.

### Next Step
Deploy verifier to Sepolia to verify the existing proof on-chain.

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
