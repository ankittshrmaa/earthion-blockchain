# 🌎 Earthion Blockchain

> A simple, understandable cryptocurrency like Bitcoin - built with Go, Python & React.

---

## 🤔 What is Earthion?

A **digital ledger** that everyone can read, but no one can change what's already written.

- 📝 Everyone can see all past transactions
- 🔒 No one can delete or alter old entries
- ⛏️ Anyone can add new blocks (mining)
- 💰 Send "EIO" coins to others

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                    BLOCKCHAIN                             │
├─────────────────────────────────────────────────────────────┤
│  Block #1      Block #2      Block #3      Block #4      │
│  ┌─────┐      ┌─────┐      ┌─────┐      ┌─────┐        │
│  │ 📝 │ ───▶ │ 📝 │ ───▶ │ 📝 │ ───▶ │ 📝 │ ───▶   │
│  └─────┘      └─────┘      └─────┘      └─────┘        │
│     │           │           │           │                    │
│     └───────────┴───────────┴───────────┘                    │
│              All connected!                               │
└─────────────────────────────────────────────────────────────┘
```

---

## ✨ Features

| Feature | Description |
|---------|------------|
| **UTXO Model** | Unspent Transaction Output (like Bitcoin) |
| **PoW Mining** | Proof of Work with dynamic difficulty |
| **Digital Signatures** | ECDSA (secp256k1) |
| **SegWit** | Segregated Witness support |
| **Schnorr Signatures** | Batch signature verification |
| **HD Wallet** | Hierarchical Deterministic wallet |
| **Merkle Proofs** | Compact proof verification |
| **Checkpoints** | Chain validation points |
| **Fork Handling** | Automatic reorg on longer valid chains |
| **P2P Network** | Peer discovery & sync |
| **Lightning** | Lightning Network support (experimental) |
| **Fee Market** | Dynamic transaction fees |

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      YOUR COMPUTER                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   ┌─────────────┐        ┌─────────────┐        ┌─────────────┐       │
│   │            │        │            │        │            │       │
│   │  FRONTEND  │ ────▶  │   Python   │ ────▶  │    Go     │       │
│   │  (React)  │        │  (FastAPI) │        │ Blockchain│       │
│   │            │        │            │        │            │       │
│   │  :5173    │        │  :8000    │        │  :8333   │       │
│   └────────────┘        └───────────┘        └──────────┘       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Components

| Component | Tech | Port | Purpose |
|-----------|------|------|--------|
| **Frontend** | React 19 + Vite | 5173 | User interface |
| **Backend** | Python FastAPI | 8000 | REST API bridge |
| **Blockchain** | Go | 8333 | Core logic, mining, wallet |

---

## 📁 Project Structure

```
earthion/
│
├── earthion-blockchain-dev/           # Go blockchain core
│   ├── cmd/
│   │   ├── main.go                # CLI entry point
│   │   └── server/               # HTTP server
│   ├── core/                     # Blockchain core
│   │   ├── block.go              # Block structure
│   │   ├── blockchain.go         # Chain management
│   │   ├── transaction.go        # UTXO transactions
│   │   ├── pow.go              # Proof of Work
│   │   ├── segwit.go           # SegWit support
│   │   ├── schnorr.go         # Schnorr signatures
│   │   ├── merkle_proofs.go   # Merkle proofs
│   │   ├── checkpoints.go     # Chain checkpoints
│   │   ├── fee.go           # Fee market
│   │   ├── mempool.go       # Transaction mempool
│   │   ├── validation.go    # Chain validation
│   │   └── utils.go        # Utilities
│   ├── wallet/                    # Wallet
│   │   └── wallet.go            # Wallet & signing
│   ├── hd/                      # HD Wallet
│   ├── crypto/                  # Cryptography
│   ├── p2p/                    # Peer-to-peer
│   │   ├── discovery.go        # Peer discovery
│   │   ├── sync.go            # Chain sync
│   │   ├── relay.go           # P2P relay
│   │   ├── peer.go           # Peer management
│   │   ├── security.go       # P2P security
│   │   └── message.go       # P2P messages
│   ├── http/                   # HTTP API
│   │   ├── server.go          # HTTP server
│   │   └── api.go            # API handlers
│   ├── lightning/              # Lightning Network
│   ├── cli/                   # CLI tool
│   └── storage/               # Persistence
│
└── earthion-backend/            # Python API + Frontend
    ├── main.py                 # FastAPI entry
    ├── requirements.txt       # Python deps
    ├── app/
    │   ├── api/              # API endpoints
    │   ├── services/         # Blockchain client
    │   ├── config.py        # Settings
    │   └── models.py       # Data models
    └── frontend/             # React app
        ├── src/
        │   ├── components/  # UI components
        │   ├── pages/      # Pages
        │   ├── services/   # API calls
        │   └── context/   # React context
        └── package.json
```

---

## 🐧 How to Run (Linux)

### Step 1: Install Dependencies

```bash
# Install Go
sudo apt update
sudo apt install -y golang-go

# Install Python
sudo apt install -y python3 python3-pip python3-venv

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Install npm packages (frontend)
cd earthion-backend/frontend
npm install
```

### Step 2: Build the Go Server

```bash
cd earthion-blockchain-dev
go mod tidy
go build -o bin/server ./cmd/server
```

### Step 3: Run All Three Parts

**Terminal 1 - Blockchain (Go):**
```bash
cd earthion-blockchain-dev
PORT=8333 ./bin/server
```

**Terminal 2 - API (Python):**
```bash
cd earthion-backend
PYTHONPATH=. python3.13 -m uvicorn main:app --host 0.0.0.0 --port 8000
```

**Terminal 3 - Frontend (Vite):**
```bash
cd earthion-backend/frontend
npm run dev
```

### Step 4: Open in Browser

```
http://localhost:5173
```

---

## 🚀 How to Run

### Prerequisites

| OS | Required |
|----|---------|
| 🐧 Linux | Go, Python 3.13+, Node.js 20+ |
| 🍎 Mac | Homebrew (go, python, node) |
| 🪟 Windows | WSL2 or Docker |

### Quick Start

```bash
# Terminal 1: Go blockchain
cd earthion-blockchain-dev
PORT=8333 ./bin/server

# Terminal 2: Python API
cd earthion-backend
PYTHONPATH=. python3.13 -m uvicorn main:app --host 0.0.0.0 --port 8000

# Terminal 3: Frontend
cd earthion-backend/frontend
npm run dev
```

### Open in Browser

```
http://localhost:5173
```

---

## 🖥️ API Endpoints

### Chain
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/chain/height` | GET | Get chain height |
| `/api/chain/validate` | GET | Validate chain |
| `/api/chain/utxo` | GET | Get UTXO set |

### Blocks
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/blocks` | GET | Get all blocks |
| `/api/blocks/{hash}` | GET | Get block by hash |
| `/api/blocks/index/{index}` | GET | Get block by index |

### Wallet
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/wallet/address` | GET | Get wallet address |
| `/api/wallet/balance` | GET | Get wallet balance |
| `/api/wallet/send` | POST | Send coins |

### Mining
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/mining/mine` | POST | Mine new block |
| `/api/mining/reward` | GET | Get mining reward |

---

## 🔧 Troubleshooting

### Port already in use
```bash
lsof -i :5173  # or :8000 or :8333
kill -9 <PID>
```

### Connection failed
- Check all 3 servers are running
- Ports: 5173 (frontend), 8000 (API), 8333 (blockchain)

### Python not found
```bash
python3 -m uvicorn main:app
```

---

## 📝 Key Concepts

| Concept | Description |
|---------|------------|
| **Block** | A "page" in the blockchain containing transactions |
| **Transaction** | Record of coins sent from A to B |
| **Mining** | Adding new blocks via Proof of Work |
| **Hash** | Unique fingerprint of a block |
| **Wallet** | Your account with address & balance |
| **UTXO** | Unspent coins you can spend |
| **SegWit** | Witness data separated from transaction |
| **Merkle Root** | Hash of all transaction hashes |

---

## 🙏 Credits

- **Go** - Blockchain core (btcd/btcec)
- **Python** - FastAPI, httpx
- **React** - Vite, Tailwind CSS

---

## 📜 License

MIT License

*Made with 💜 for blockchain education.*