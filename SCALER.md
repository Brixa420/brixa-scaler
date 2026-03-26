# Brixa Scaler

Network routing and TPS layer.

## Architecture Components

### 1. Optimistic Game Loop
- **Purpose:** <10ms tick rate for gameplay
- **How:** Batching for commitment (not ZK proving)
- **TPS:** 100K+

### 2. Merkle State Roots
- **Purpose:** Every tick, cheap commitment
- **How:** SHA256 Merkle tree
- **TPS:** 25M (tested)

### 3. Periodic ZK Rollup
- **Purpose:** Every N ticks, prove state integrity
- **How:** Your 3,800 TPS proving (sufficient)
- **Latency:** 938ms acceptable for settlement

### 4. Agent Identity (ZK Credentials)
- **Purpose:** Identity verification, not per-action
- **Circuit:** Different from transaction proving
- **Infra:** Reuses same ZK infrastructure

## Layer Summary

| Component | Purpose | TPS | When |
|-----------|---------|-----|------|
| Optimistic loop | Gameplay speed | 100K+ | Every tick |
| Merkle roots | Cheap commitment | 25M | Every tick |
| ZK rollup | Settlement audit | 4K | Every N ticks |
| Agent ZK | Identity credentials | (same infra) | On-demand |

## Advantage
- Game loop never waits for ZK
- ZK proves periodic integrity, not real-time
- Same infrastructure handles both

## NOT a Blockchain
Brixa Scaler is NOT a blockchain. It is chain-agnostic.
