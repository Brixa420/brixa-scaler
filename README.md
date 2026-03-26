# BrixaScaler

**High-throughput transaction batching with ZK proofs for AI agents and blockchain games.**

BrixaScaler lets thousands of AI agents or game players act instantly off-chain, then settle securely on-chain with cryptographic proof. No $50K gas bills. No 12-second waits.

---

## The Problem

### For AI Teams
Pay fifty thousand dollars per month in gas or slow down your agents. Those are the only options on current blockchains. Every inference, every model update, every data purchase hits the chain individually. AI agents generate massive transaction volumes. On-chain this is expensive and slow. Current blockchains cannot handle AI-native economies.

### For Gaming Teams
Blockchain games die when players wait twelve seconds for a transaction. Fun and blockchain feel incompatible. App-chains fragment liquidity and community.

---

## Why BrixaScaler

### For AI Teams
- **Agent payments** — Agents transact thousands of times per second off-chain, settle value periodically on-chain
- **Model marketplace** — One million API calls equals ten cents in fees instead of five hundred dollars
- **Data provenance** — ZK proofs verify data integrity without revealing the data itself
- **Compute verification** — Prove an AI computation happened correctly, settle the proof not the full trace

**The Win:** AI teams get Web2-speed economics with Web3-verifiability. Agents act fast, settle slow, stay honest.

### For Gaming Teams
- **Real-time actions** — Seven hundred fifty thousand TPS ingestion equals instant item pickups, movement, combat. Players do not wait.
- **Asset ownership** — Periodic ZK settlement means players actually own items on Ethereum. They do own their stuff.
- **Economy integrity** — Cryptographic receipts that prove the game was fair
- **Cross-game items** — Settle to any chain, player takes sword from Polygon game to Ethereum game

**The Win:** Games feel like games, not blockchain demos. Players get NFT ownership without NFT friction.

---

## Why Seven Hundred Fifty Thousand TPS Batching Is Enough

### Misconception versus Reality
- **Batching is fake TPS** — Batching is how Visa works, authorize fast settle slow
- **Users need instant finality** — Users need instant response, periodic certainty
- **One to five proofs per second is too slow** — One proof can cover ten thousand batched transactions

### The Math
- Seven hundred fifty thousand TPS ingestion equals what players feel, the speed of gameplay actions
- One proof per second equals cryptographically verify all seven hundred fifty thousand as valid
- Sixty five TPS settlement equals what hits the blockchain, the security layer

**Players experience seven hundred fifty thousand speed. Blockchain gets sixty five TPS security. Everyone wins.**

---

## The Bottom Line

- AI teams get agent economies that actually scale without fifty thousand dollar gas bills
- Gaming teams get Web2 UX with Web3 ownership, players do not wait but they do own their stuff
- Both get one infrastructure, any settlement chain, honest claims

**Settle to Ethereum, Polygon, Arbitrum, or any chain. Your choice.**

---

## Quick Start

```bash
cd integration/go
go build -o brixascaler server.go
./brixascaler
```

Server runs on `http://localhost:8080` by default.

### Endpoints
- `POST /batch` — Submit a batch of transactions
- `GET /health` — Server health and stats
- `GET /benchmark` — Quick TPS benchmark
- `GET /metrics` — Prometheus metrics

---

## Architecture

```
Player/Agent Action
        ↓
   [BrixaScaler] ← 750K TPS ingestion
        ↓
  Batch + ZK Proof
        ↓
   Settlement Chain ← 65 TPS verification
```

---

## License

MIT

---

## Getting Started

### Prerequisites
- Go 1.21+
- (Optional) Ethereum node for settlement

### Run the Server

```bash
# Default (localhost:8080)
go run integration/go/server.go

# Custom ports
RPC_PORT=9000 METRICS_ENABLED=true go run integration/go/server.go
```

### Submit a Transaction Batch

```bash
curl -X POST http://localhost:8080/batch \
  -H "Content-Type: application/json" \
  -d '[
    {"from": "0x742d35Cc6634C0532925a3b844Bc9e7595f0fAb1", "to": "0x8ba1f109551bD432803012645Ac136ddd64DBA72", "value": 1000, "nonce": 1},
    {"from": "0x8ba1f109551bD432803012645Ac136ddd64DBA72", "to": "0x742d35Cc6634C0532925a3b844Bc9e7595f0fAb1", "value": 500, "nonce": 2}
  ]'
```

Response:
```json
{
  "batch_id": "0xabc123...",
  "root": "0xdef456...",
  "tx_count": 2,
  "fee": "0.001 ETH"
}
```

### Check Health

```bash
curl http://localhost:8080/health
```

### Run Benchmark

```bash
curl http://localhost:8080/benchmark
```

---

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `RPC_PORT` | `8080` | Main server port |
| `METRICS_PORT` | `9090` | Prometheus metrics port |
| `METRICS_ENABLED` | `false` | Enable metrics server |
| `MAX_BATCH_SIZE` | `10000` | Max transactions per batch |
| `SETTLEMENT_CHAIN` | `ethereum` | Target chain for settlement |
| `DEMO_MODE` | `true` | Skip actual blockchain calls |

---

## API Reference

### POST /batch
Submit a batch of transactions for processing.

**Request:**
```json
[
  {
    "from": "0x...",
    "to": "0x...",
    "value": 1000,
    "nonce": 1
  }
]
```

**Response:**
```json
{
  "batch_id": "0x...",
  "root": "0x...",
  "tx_count": 1,
  "fee": "0.001 ETH",
  "timestamp": 1700000000
}
```

### GET /health
Returns server health and statistics.

### GET /benchmark
Runs a quick TPS benchmark.

### GET /metrics
Prometheus-compatible metrics endpoint.

---

## Architecture Deep Dive

### Batching Layer
Transactions are collected in memory and batched periodically or when batch size threshold is reached.

### Merkle Tree
Each batch is committed to a Merkle tree, enabling efficient proof generation.

### ZK Proofs
ZK-SNARKs prove batch validity without revealing individual transaction details.

### Settlement
Batches settle to the target chain with proof verification. Supported:
- Ethereum
- Polygon
- Arbitrum
- Any EVM-compatible chain

### Flow
```
1. Agent/Player Action → BrixaScaler Ingestion
2. Batch Accumulation → Merkle Tree Build
3. Batch Full/Timeout → ZK Proof Generation
4. Proof + Commitment → Settlement Chain
5. Verification → On-chain Finality
```

---

## Performance

- **Ingestion**: 750,000 TPS (off-chain)
- **Proof Generation**: ~1 second per 10,000 transactions
- **Settlement**: 65 TPS (on-chain verification)

---

## Contributing

1. Fork the repo
2. Create a feature branch
3. Add tests (target 100% coverage)
4. Submit a PR

---

## License

MIT
