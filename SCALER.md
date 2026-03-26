# Brixa Scaler

Network routing and TPS layer.

## Recent Updates (March 26, 2026)

### Critical Interfaces
- `execution/interfaces.js` - Standardized API between Scaler/Node Engine and settlement

#### 1. Transaction Ingestion (BatcherInput)
Any tx format:
- txData: Opaque payload (any format)
- intent: "payment" | "compute" | "storage" (for sharding)
- priority: Ordering hint (0-100)

#### 2. Proof Output (ProofBundle)
Any settlement layer verifies this:
- merkleRoot, proof (SNARK/STARK), publicInputs, metadata
- Hardware: "cpu" | "cuda" | "opencl"

#### 3. Settlement Config (SettlementConfig)
Configurable finality:
- mode: "time" | "batchSize" | "manual"
- destination: "ethereum" | "polygon" | "custom" | "none"
- compression: "none" | "recursive" | "aggregate"

Presets: fast, balanced, secure, throughput

## NOT a Blockchain

Brixa Scaler is NOT a blockchain. It is chain-agnostic.
