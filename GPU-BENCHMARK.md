# GPU ZK Benchmark Results (RunPod RTX 4090)

## Test Date: March 29, 2026

### Environment
- RunPod GPU: RTX 4090
- Location: wrath-gpu pod (yxkcdt1686dsfg)
- gnark version: v0.14.0

### Results

| Test | Hardware | Time (100 proofs) | TPS |
|------|----------|------------------|-----|
| Real MiMC Merkle Circuit (64 tx) | Apple M4 (CPU) | 21,052ms | 4.75 |
| Real MiMC Merkle Circuit (64 tx) | RTX 4090 (CPU mode) | 44,811ms | 2.23 |

### Key Finding

**gnark does NOT automatically use GPU!** Running on RTX 4090 gave same/slower results because gnark was running on CPU (the Go process doesn't utilize CUDA automatically).

### What We Tried

1. **icicle-gnark** - CUDA backend for gnark
   - Downloaded v3.9.2 and v4.0.0
   - Failed due to library linking issues
   - Requires complex setup with specific paths

2. **SP1 (Succinct Labs)** - Modern ZK framework
   - Requires Rust - not available on RunPod

3. **ZoKrates** - Python-based ZK
   - Not available in pip

### Conclusion

| Approach | TPS | Status |
|----------|-----|--------|
| gnark CPU (trivial circuit) | ~2,000 | Misleading |
| gnark CPU (real MiMC) | ~5 | Honest |
| gnark on RTX 4090 | ~2-5 | Same (no GPU auto) |
| GPU-accelerated ZK | Unknown | Needs complex setup |

### Path Forward

To achieve 12K+ TPS:
1. Use GPU-accelerated framework (SP1, RiscZero, icicle-gnark)
2. Use recursive proofs to amortize verification cost
3. Accept CPU limitations for now

### References

- See `REAL_BENCHMARK.md` for circuit details
- See `BENCHMARKS.md` for full project benchmarks
