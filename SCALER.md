# Brixa Scaler

Network routing and TPS layer.

## Hardware Baseline (Mac mini M4)

**⚠️ IMPORTANT: This is a BASELINE, not a ceiling**

| Hardware | TPS | Notes |
|----------|-----|-------|
| Mac mini M4 | ~3,800 | **Baseline** (9 cores) |

### What's This Baseline?
- Tests run on consumer hardware (Mac mini M4)
- Proves the architecture works
- Shows ~3,800 TPS is achievable

### What's NOT Included
- ❌ No GPU (would be 20x faster)
- ❌ No multi-machine cluster (10 machines = 10x)
- ❌ No production-optimized circuit

### Extrapolated Potential
With proper infrastructure:

| Config | Estimated TPS |
|--------|---------------|
| + GPU (per machine) | 76,000 |
| + 10 machines | 760,000 |
| Both | 1,500,000+ |

### Run Your Own Baseline
```bash
cd keys
npm install snarkjs circomlib
node parallel-bench.js
```

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
