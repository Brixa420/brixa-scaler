# Brixa Scaler

Network routing and TPS layer.

## Honest Status (March 26, 2026)

### What Actually Works

**Merkle Tree Batching (Real)**
- Building SHA256 Merkle tree: ~90ms for 100K transactions
- TPS: ~1M (just hashing, no ZK)

**ZK Verification (Real)**
- snarkjs.groth16.verify() on pre-generated proof: ~514ms
- Can verify 1.9 proofs/sec

### What Doesn't Work Yet

**ZK Proving**
- Cannot generate new proofs - circuit requires correct Merkle path
- Would need to: build tree → extract path → prove → verify
- This is the hard part

### Verified (Real)
- keys/circuit compiles
- keys/proof.json verifies successfully  
- keys/verification_key.json is valid

### What's Missing for Full ZK
1. Build actual Merkle tree with all txs
2. Extract proof path for each batch
3. Generate circuit inputs
4. Run snarkjs.fullProve()
5. Deploy verifier to Sepolia

### Honest Numbers
| Component | Real | Notes |
|-----------|------|-------|
| Merkle tree | 90ms/100K | SHA256 only |
| ZK Prove | FAILS | Need Merkle path inputs |
| ZK Verify | 514ms | Pre-generated proof |
| TPS | ~1M | Merkle building only |

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
