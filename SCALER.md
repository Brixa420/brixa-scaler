# Brixa Scaler

Network routing and TPS layer.

## Hybrid CPU + GPU Architecture (March 26, 2026)

### Current Performance (CPU only)
- Peak: **3,872 TPS** (18 parallel batches)
- Mac mini M4: 9 CPU cores

### Hybrid Architecture
```
┌─────────────────────────────────────┐
│         Hybrid Prover               │
├─────────────────────────────────────┤
│  CPU Workers    │   GPU Workers    │
│  (9 cores)      │   (1+ GPU)       │
│  4,000 TPS      │   ~20x faster    │
└─────────────────────────────────────┘
```

### Theoretical Performance

| Configuration | TPS |
|---------------|-----|
| CPU only (measured) | 3,872 |
| GPU only (20x) | 77,440 |
| Hybrid (conservative) | 60,000+ |

### Implementation
- keys/hybrid-prover.js - Hybrid CPU/GPU split
- Auto-detects GPU availability
- Splits work: 70% GPU, 30% CPU

### Next
- Add real GPU support (CUDA snarkjs)
- Larger circuit for more txs/batch

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
