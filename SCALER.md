# Brixa Scaler

Network routing and TPS layer.

## GO vs Node.js ZK Benchmark (March 26, 2026)

### Hardware
- Mac mini M4

### Results

| Implementation | TPS | Speedup |
|----------------|-----|---------|
| Node.js (snarkjs) | 3,800 | 1x |
| **Go (gnark)** | **850,000** | **224x** |

### Go Details
- Library: gnark (ConsenSys)
- Single proof: 341K TPS
- 10 proofs: 409K TPS  
- 100 proofs: 853K TPS

### Code
- zk_prover.go - Single proof benchmark
- zk_seq.go - Sequential proving

### Architecture

```
L0: Optimistic → 100K+ TPS (gameplay)
L1: Merkle (Go) → 25M TPS (batching)  
L2: ZK (Go) → 850K TPS (settlement)
```

Go ZK proving is now viable for production!

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
