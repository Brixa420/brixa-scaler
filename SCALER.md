# Brixa Scaler

Network routing and TPS layer.

## Current Status (March 26, 2026)

### Working

| Component | Status | Speed |
|-----------|--------|-------|
| Merkle tree (SHA256) | ✅ Works | ~1.1M TPS |
| ZK Verify | ✅ Works | 16ms, 63/sec |

### Not Working

**ZK Prove** - Requires trusted setup that takes too long
- Phase 2 preparation: hours
- Circuit setup + contribution: more hours

### Path Forward

To get ZK proving working:

1. **Trusted setup** (takes ~4+ hours):
   ```bash
   snarkjs ptn bn128 20 powersoftau_0000.ptau  # 1-2 min
   snarkjs pt2 powersoftau_0000.ptau final.ptau  # 2-4 HOURS
   snarkjs g16s circuit.r1cs final.ptau key_0000.zkey  # 30 min
   snarkjs zkc key_0000.zkey final.zkey  # 30 min
   ```

2. **Or use external service** - Generate proof on server with pre-built keys

3. **Or simpler circuit** - Non-Merkle ZK for simpler verification

### Current Numbers

- Merkle (100K): 90ms → 1.1M TPS
- ZK verify: 16ms → 63 verifications/sec
- ZK prove: Needs new keys

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
