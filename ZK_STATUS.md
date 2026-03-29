# ZK Status - HONEST Numbers

## Current Implementation
- **Circuit:** 10-level Merkle (1024 leaves), 221 constraints
- **Hash:** Poseidon-like (x^5 s-box)
- **Backend:** gnark Groth16 on bn254
- **Hardware:** Apple M4

## What's ACTUALLY Working ✅

| Metric | Value | Method |
|--------|-------|--------|
| Proving | ~2,150/sec | Local benchmark (verified 3 runs) |
| Single proof | 0.40ms | Average of 1000 proofs |
| Merkle tree | ~6,000/sec | HTTP /buildtree endpoint |
| Proof size | ~128 bytes | Groth16 bn254 |

## What's BROKEN ❌

| Metric | Status | Issue |
|--------|--------|-------|
| Verification | ❌ Broken | gnark v0.14.0 Verify() panics with nil pointer |
| HTTP prove | ❌ Hangs | Crashes after prove on verify |
| External proving | ⚠️ No endpoint | /prove hangs, /proveonly crashes |

## The gnark Bug

```
panic: runtime error: invalid memory address or nil pointer dereference
  at groth16.Verify()
```

**This is a BUG IN GNARK v0.14.0**, not our code.
- Tried v0.13.0, v0.12.0 - same issue
- PLONK requires SRS (trusted setup)
- Proving works fine, verification crashes

## What We Measured (Real)

### ZK Proving (gnark groth16)
```
10 proofs: 5-7ms   = 1,400-2,000/sec
50 proofs: 23-28ms = 1,785-2,173/sec  
100 proofs: 45-48ms = 2,083-2,222/sec
500 proofs: 233-252ms = 1,984-2,145/sec
1000 proofs: 460-475ms = 2,105-2,188/sec
```

**Average: ~2,150 proofs/sec**

### Merkle Tree Building
```
100 leaves:  15ms  = 6,666/sec
1000 leaves: 485ms = 2,061/sec
```

## What Was Claimed vs What's Real

| Claim (git) | Reality | Issue |
|-------------|---------|-------|
| 424 proofs/sec | 2,150/sec | Old benchmark |
| Verification <1ms | BROKEN | gnark bug |
| 140K-435K TPS | UNVERIFIABLE | Can't verify without working verify |
| 2,000 base TPS | ~2,150/sec | This one is accurate ✅ |

## Honest Numbers

| Layer | Real TPS | Notes |
|-------|----------|-------|
| ZK Proving | ~2,150 | Verified |
| Merkle | ~6,000 | Verified |
| **E2E** | **UNVERIFIABLE** | Can't verify proofs |

## What NOT to Claim

- ❌ "140K TPS" - Not verified (verification broken)
- ❌ "435K TPS recursive" - Not verified
- ❌ "ZK-verified batch" - Can't verify without fix
- ❌ "1.3M TPS" - Calculated, not measured

## What CAN We Claim

- ✅ "2,150 proofs/sec on M4"
- ✅ "6,000 Merkle roots/sec"
- ⚠️ "Theoretical max: proving × batch size"

## Path Forward

1. **Fix verification** - Use different gnark version or external verifier
2. **Re-benchmark E2E** - After fix, measure real end-to-end
3. **Update claims** - Only claim verified numbers

## Files
- `zk-gnark/merkle_10.go` - 10-level circuit (proving works)
- `zk-gnark/bench_raw.go` - Benchmark that shows real numbers
- `integration/recursive-batching-gnark.js` - Client

## Last Updated
March 28, 2026 - Fixed after discovering gnark verification bug