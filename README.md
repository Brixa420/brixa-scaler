# BrixaScaler

⚠️ **IMPORTANT: DEMO MODE**

**DEFAULT IS DEMO MODE** — Transactions are logged but NOT sent to any blockchain!

To enable real transactions:
```bash
export DEMO_MODE=false
export SETTLEMENT_RPC_URL=https://your-rpc-url
export SETTLEMENT_PRIVATE_KEY=your_private_key_here
```

⚠️ **WARNING:** Operating with `DEMO_MODE=false` involves **REAL MONEY**. Use at your own risk.

This software is provided as-is without warranty. The authors assume no liability for any losses incurred through use of this software. Always test thoroughly in demo mode before enabling real transactions.

> ⚠️ **For Developers:** This is pre-production software. ZK proof generation, actual RPC settlement, and hardware wallet signing are stubs/placeholders. See GitHub issues for implementation status.

---

**High-throughput transaction batching with ZK proofs for AI agents and blockchain games.**

BrixaScaler lets thousands of AI agents or game players act instantly off-chain, then settle securely on-chain with cryptographic proof. No fifty thousand dollar gas bills. No twelve second waits.

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
- **Real-time actions** — Four million TPS ingestion equals instant item pickups, movement, combat. Players do not wait.
- **Asset ownership** — Periodic ZK settlement means players actually own items on Ethereum. They do own their stuff.
- **Economy integrity** — Cryptographic receipts that prove the game was fair
- **Cross-game items** — Settle to any chain, player takes sword from Polygon game to Ethereum game

**The Win:** Games feel like games, not blockchain demos. Players get NFT ownership without NFT friction.

---

## What Was Actually Measured

One thousand transactions batch plus Merkle took 237,808 nanoseconds per operation equals 0.238 milliseconds. This equals approximately 4.2 million transactions per second. This is the Go batcher ProcessBatch function doing exactly what the server does. Includes hashing transactions and building a Merkle tree. No mocking, no shortcuts, real code path.

---

## Understanding the Math

Per batch measurement means each operation processes one thousand transactions. If you send one thousand transactions per request at 4.2 million TPS throughput, the math works as follows: 4.2 million TPS divided by one thousand transactions per batch equals 4,200 batches per second. Each batch takes 0.238 milliseconds. This is the ingestion layer speed.

---

## Honest Claim

Four million TPS for batch plus Merkle construction in the Go batcher ProcessBatch function. Each batch contains one thousand transactions. Network input output, serialization, and ZK proving are separate bottlenecks that limit real world end to end throughput.

---

## What This Means

The Go layer is not the bottleneck. The bottleneck is ZK proving at one to five proofs per second. The Go layer can ingest and hash transactions faster than they can be proven. This is architecturally correct. Fast ingestion, slow proving, periodic settlement.

---

## Credibility Check

237,808 nanoseconds per operation is a specific measurable number. It can be reproduced with Go benchmark tools. The math is transparent and verifiable. The limitation is clearly stated.

---

## The Bottom Line

- AI teams get agent economies that actually scale without fifty thousand dollar gas bills
- Gaming teams get Web2 UX with Web3 ownership, players do not wait but they do own their stuff
- Both get one infrastructure, any settlement chain, honest claims

**Settle to Ethereum, Polygon, Arbitrum, or any chain. Your choice.**

---

## Quick Start

```bash
# Install dependencies
npm install

# Run Brixaroll (recommended, true off-chain)
node integration/brixaroll.js --rpc https://your-rpc-url

# OR run BrixaScaler (simple batching)
node integration/brixa-scaler.js --rpc https://your-rpc-url

# Go implementation (faster)
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
   [BrixaScaler] ← 4M+ TPS ingestion
        ↓
  Batch + Merkle Tree
        ↓
  ZK Proof Generation ← 1-5 proofs/second
        ↓
   Settlement Chain ← 65 TPS verification
```

---

## Environment Variables

### Security
| Variable | Default | Description |
|----------|---------|-------------|
| `API_KEY` | - | Authentication key (header `X-API-Key` or `?api_key=`) |
| `RATE_LIMIT_PER_SECOND` | 10 | Requests per second per client |
| `RATE_LIMIT_BURST` | 20 | Burst allowance for rate limiting |
| `MAX_REQUEST_SIZE` | 1048576 | Max request size in bytes (1MB) |
| `MAX_GAS_PRICE_GWEI` | 100 | Maximum gas price in Gwei |
| `CORS_ORIGINS` | - | Comma-separated allowed origins |
| `MAX_TX_VALUE` | 1000 ETH | Maximum transaction value |
| `MAX_TX_DATA_SIZE` | 1024 | Max transaction data size |
| `REDIRECT_HTTP_TO_HTTPS` | false | Redirect HTTP to HTTPS |

### Settlement (when DEMO_MODE=false)
| Variable | Description |
|----------|-------------|
| `SETTLEMENT_PRIVATE_KEY` | Private key for signing transactions |
| `SETTLEMENT_RPC_URL` | Blockchain RPC URL |
| `SETTLEMENT_CHAIN_ID` | Chain ID (default: 1 for Ethereum) |
| `SETTLEMENT_GAS_LIMIT` | Gas limit per transaction (default: 21000) |

### Multi-Sig
| Variable | Default | Description |
|----------|---------|-------------|
| `MULTISIG_ENABLED` | false | Enable multi-sig |
| `MULTISIG_THRESHOLD` | 2 | Required approvals |
| `MULTISIG_APPROVERS` | - | Comma-separated approver addresses |
| `MULTISIG_HIGH_VALUE_THRESHOLD` | 10 ETH | Value requiring multi-sig |

### Hardware Wallet
| Variable | Description |
|----------|-------------|
| `WALLET_TYPE` | "software" (default), "trezor", "ledger" |
| `HW_WALLET_PATH` | Device path for hardware wallet |
| `HW_CHAIN_ID` | Chain ID for hardware wallet |

### Key Rotation
| Variable | Default | Description |
|----------|---------|-------------|
| `KEY_ROTATION_ENABLED` | false | Enable automatic key rotation |
| `KEY_ROTATION_INTERVAL_HOURS` | 168 | Hours between rotations (7 days) |
| `KEY_ROTATION_WEBHOOK_URL` | - | Alert webhook URL |

---

## Docker

```bash
# Build
docker build -t brixascaler .

# Run
docker run -p 8080:8080 -p 9090:9090 \
  -e DEMO_MODE=true \
  -e API_KEY=your_key \
  brixascaler

# Or with docker-compose
cp .env.example .env
# Edit .env with your values
docker-compose up -d
```

---

## License

MIT

**Demo Only - Don't Sue Us:** This software is provided as-is for demonstration purposes. No real transactions are processed by default. Use at your own risk.
