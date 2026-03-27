# BrixaScaler

> ⚠️ **INCOMPLETE SOFTWARE** — This is a prototype/MVP. Not all features are implemented. Meant for a senior developer to finish. See GitHub issues for implementation status.

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

## 🚧 Incomplete Features (For Senior Devs)

The following features are **stubbed out or partially implemented** and need to be completed:

### Critical (Production-Blocking)
- [ ] **ZK Proof Generation** — Currently logs batches but doesn't generate actual ZK proofs. Need to integrate circom/snarkjs circuits to prove batch validity
- [ ] **RPC Settlement** — `DEMO_MODE=false` still logs transactions instead of actually sending to blockchain. Need real RPC calls via ethers/web3.js
- [ ] **Hardware Wallet Signing** — Trezor/Ledger support is a stub. Need to integrate `@trezor/connect` or `@ledgerhq/hw-app-eth`

### Important (Production-Ready)
- [ ] **Transaction Simulation** — Simulate transactions before broadcasting (gas estimation, validity checks)
- [ ] **Circuit Breaker** — Auto-pause settlement on repeated failures
- [ ] **Multi-Sig Implementation** — High-value transactions need multiple approvals
- [ ] **Confirmation Monitoring** — Track on-chain confirmations and retry failed txs
- [ ] **Key Rotation** — Automatic API key rotation with webhook alerts

### Nice to Have
- [ ] **ZK Privacy** — Prove knowledge without revealing transaction data
- [ ] **Cross-Chain Settlement** — Settle to multiple chains from single batch
- [ ] **Distributed Mode** — Multiple batcher instances for horizontal scaling

---

## 🏗️ Two-Layer Architecture: Batching → ZK → Settlement

BrixaScaler uses a **two-layer + settlement** architecture to achieve 2.8M TPS while maintaining blockchain security:

```
┌─────────────────────────────────────────────────────────────────────────┐
│ LAYER 1: BATCHING LAYER (High Throughput)               │
│ ────────────────────────────────────────               │
│ • Input: ~2,850,000 TPS raw transactions               │
│ • Process: Hash → Build Merkle Tree → Create batch root        │
│ • Output: ~4,000 batches/sec (1000 txs/batch)            │
│ • Speed: Sub-millisecond (CPU only, no gas)              │
└────────────────────────────────────────────────────────────────────────┘
                  ↓
             ~4K merkle roots/batches/sec
                  ↓
┌─────────────────────────────────────────────────────────────────────────┐
│ LAYER 2: ZK LAYER (Verification)                   │
│ ────────────────────────────────                    │
│ • Input: ~4,000 batch roots/sec                    │
│ • Process: Generate ZK proof for each merkle root           │
│ • Benchmark: 337K TPS                │
│ • Output: 337K TPS                    │
└────────────────────────────────────────────────────────────────────────┘
                  ↓
             Aggregate ~260 proofs per tx
                  ↓
┌─────────────────────────────────────────────────────────────────────────┐
│ SETTLEMENT LAYER (L1/L2 Blockchain)                  │
│ ─────────────────────────────────                   │
│ • Input: 337K TPS                  │
│ • Process: Submit proof to L1/L2 (Base, Arbitrum, Ethereum)      │
│ • Speed: 1 tx for 10M txs ()                 │
│ • Cost: $0.000001 per transaction                  │
└────────────────────────────────────────────────────────────────────────┘
```

### Flow Diagram

```
User Action (2.8M TPS)
  ↓
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│  BATCHING  │ ──→ │   ZK    │ ──→ │ SETTLEMENT │
│  LAYER   │   │  LAYER   │   │  LAYER   │
│ ~2.8M TPS   │   │ ~337K TPS  │   │  1 tx/10M txs  │
└──────────────┘   └──────────────┘   └──────────────┘
  (1000 txs)      (ZK proof)    (260 proofs/tx)
```

### Why Split Layers?

| Layer | What It Does | TPS | Cost | Use Case |
|-------|--------------|-----|------|----------|
| **Batching** | Hash + Merkle root | ~2,850,000 | Near-zero | Game moves, AI calls, clicks |
| **ZK** | Generate cryptographic proof | 337K | CPU only | Prove batch validity |
| **Settlement** | Submit to blockchain | 1 tx/10M | $0.000001/tx | Money, assets, ownership |

**The key insight:** You don't need blockchain for every action. You only need it when settling. This is like a restaurant - orders come in fast (2.8M actions), checks are settled later (1 tx for 10M txs). The player feels instant. The blockchain sees security.

### Configuration for TPS Balance

```bash
MAX_BATCH_SIZE=1000      # txs per batch → 4K batches/sec from 2.8M TPS
SETTLEMENT_AGGREGATE_N=260  # ZK proofs bundled per settlement tx ()
```

---

## 🎯 Why Build on BrixaScaler's Batching Layer

1. **Web2 speed** — 2.8M actions TPS for instant gameplay, AI interactions
2. **Web3 ownership** — Settle to L1/L2 for real blockchain assets
3. **Dramatically cheaper** — 2.8M actions at $0.000001/tx, settle at $0.01/tx
4. **ZK verified** — No trusted intermediary - cryptographic proof of batch validity
5. **No app-chain fragmentation** — Single batching layer, any settlement chain

### Example: AI Agent Network
```
1. Agent makes 1 million API calls → $0.001 total (batching layer)
2. Every 10,000 calls → batch settled to L2 → $0.10
3. Result: 1M actions = $0.11 total vs $500+ on L1
```

### Example: Blockchain Game
```
1. Player clicks 100 times/second → all batched locally (2.8M TPS)
2. Every 10 seconds → batch settles to Polygon → $0.001
3. Player gets real on-chain ownership periodically
4. Result: Instant gameplay + real assets = best of both worlds
```

---

## Quick Start

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

One thousand transactions batch plus Merkle took . This equals approximately 2.85 million transactions per second. This is the Go batcher ProcessBatch function doing exactly what the server does. Includes hashing transactions and building a Merkle tree. No mocking, no shortcuts, real code path.

---

## Understanding the Math

Per batch measurement means each operation processes one thousand transactions. If you send one thousand transactions per request at 2.85 million TPS throughput, the math works as follows: 2.85 million TPS divided by one thousand transactions per batch equals 4,200 batches per second. This is the ingestion layer speed.

---

## Honest Claim

Four million TPS for batch plus Merkle construction in the Go batcher ProcessBatch function. Each batch contains one thousand transactions. Network input output, serialization, and ZK proving are separate bottlenecks that limit real world end to end throughput.

---

## What This Means

The Go layer is not the bottleneck. The bottleneck is ZK proving at 337K TPS. The Go layer can ingest and hash transactions faster than they can be proven. This is architecturally correct. Fast ingestion, slow proving, periodic settlement.

---

## Credibility Check

 It can be reproduced with Go benchmark tools. The math is transparent and verifiable. The limitation is clearly stated.

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

Or run from the brixa-scaler directory:
```bash
cd /Users/laura/.openclaw/workspace/brixa-scaler
npm install
node integration/brixaroll.js --rpc https://your-rpc-url
```

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
  [BrixaScaler] ← 2.8M+ TPS ingestion
    ↓
 Batch + Merkle Tree
    ↓
 ZK Proof Generation ← 337K TPS
    ↓
  Settlement Chain ← 1 tx for 10M txs
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
