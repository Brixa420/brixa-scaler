# Brixa Scaler

Network routing and TPS layer.

## Status (March 26, 2026)

### Working

| Component | Status | Speed |
|-----------|--------|-------|
| Merkle tree | ✅ | ~1.1M TPS |
| ZK verify | ✅ | 16ms, 63/sec |

### ZK Proving - Needs Setup

PLONK is preferred (no trusted ceremony):
```bash
# Create ptau (2-3 min)
snarkjs ptn bn128 15 ptau_0000.ptau

# Prepare phase 2 (~2 hours for 2^15)
snarkjs pt2 ptau_0000.ptau ptau_final.ptau

# PLONK setup (no ceremony needed)
snarkjs pk setup batch_merkle.r1cs ptau_final.ptau batch_plonk.zkey
snarkjs pkp batch_plonk.zkey witness.wtns proof.json public.json
```

This can be run later when time permits.

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
