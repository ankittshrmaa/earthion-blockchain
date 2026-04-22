# 🌎 Earthion Blockchain

> A simple, understandable blockchain for everyone.

---

## 🤔 What is Earthion?

Imagine a **book** digital notethat everyone can read, but no one can tear pages out of or change what's written. That's essentially what a blockchain is!

### In Simple Words

Think of Earthion like a **share* whered Google Doc*:
- 📝 Everyone can see all past versions
- 🔒 No one can delete or change old entries
- ⛏️ Anyone can add new pages (mining)
- 💰 You can send "EIO" coins to others

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                    BLOCKCHAIN                             │
├─────────────────────────────────────────────────────────────┤
│  Block #1      Block #2      Block #3      Block #4        │
│  ┌─────┐      ┌─────┐      ┌─────┐      ┌─────┐          │
│  │ 📝 │ ───▶ │ 📝 │ ───▶ │ 📝 │ ───▶ │ 📝 │ ───▶    │
│  │     │      │     │      │     │      │     │            │
│  └─────┘      └─────┘      └─────┘      └─────┘          │
│     │           │           │           │                    │
│     └───────────┴───────────┴───────────┘                    │
│              All connected!                               │
└─────────────────────────────────────────────────────────────┘
```

Each block contains:
- **Transactions** - Like receipts showing who sent coins to whom
- **Hash** - A unique fingerprint of that block
- **Previous Hash** - Links to the block before it

---

## 🏗️ Architecture

### Three Parts Working Together

```
┌─────────────────────────────────────────────────────────────────────┐
│                         YOUR COMPUTER                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   ┌─────────────┐        ┌─────────────┐        ┌─────────────┐       │
│   │            │        │            │        │            │       │
│   │  FRONTEND  │ ────▶  │   Python   │ ────▶  │    Go     │       │
│   │  (React)  │        │  (FastAPI) │        │ Blockchain│       │
│   │            │        │            │        │            │       │
│   │  :5173    │        │  :8000    │        │  :9333   │       │
│   └────────────┘        └───────────┘        └──────────┘       │
│         │                   │                    │                   │
│         │        ┌───────┘                    │                   │
│         │        │                             │                   │
│         ▼        ▼                             ▼                   │
│   ┌──────────┐  ┌──────────┐         ┌──────────┐              │
│   │  User    │  │  Browser │         │ Network  │              │
│   │  sees   │  │ display  │         │ listens │              │
│   │  this   │◄─┘        │         │ for peers│              │
│   └─────────┘           └──────────┘         └──────────┘              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 1. Go Blockchain (Core)
The heart of the system - handles all the blockchain logic.

| What it does | Files |
|-------------|-------|
| Stores blocks | `core/blockchain.go` |
| Creates blocks | `core/block.go` |
| Handles transactions | `core/transaction.go` |
| Mining (PoW) | `core/pow.go` |
| Wallet management | `wallet/wallet.go` |
| Network sync | `p2p/` |

### 2. Python Backend (API)
Translates between user and blockchain.

| What it does | Files |
|-------------|-------|
| REST API | `main.py` |
| Blockchain client | `app/services/blockchain.py` |
| API endpoints | `app/api/` |

### 3. React Frontend (UI)
What you see in the browser.

| What it shows | Files |
|--------------|-------|
| Wallet display | `components/WalletCard.tsx` |
| Block list | `components/BlockList.tsx` |
| Mining button | `components/MiningControls.tsx` |
| Send coins | `components/SendTransaction.tsx` |

---

## 🚀 How to Run

### Prerequisites

| OS | What you need |
|-----|--------------|
| 🪟 Windows | WSL2 (Ubuntu) or Docker |
| 🍎 Mac | Terminal (included) |
| 🐧 Linux | Terminal (included) |

### Quick Start (All OS)

```bash
# 1. Go to the project folder
cd /home/user/earthion

# 2. Start the Go blockchain (Terminal 1)
cd earthion-blockchain-dev
PORT=9333 ./bin/server

# 3. Start Python API (Terminal 2)
cd earthion-backend
PYTHONPATH=. python3.13 -m uvicorn main:app --host 0.0.0.0

# 4. Start Frontend (Terminal 3)
cd earthion-backend/frontend
npm run dev
```

### Detailed Instructions

#### 🪟 Windows (using WSL2)

```powershell
# Install WSL2 if you don't have it:
wsl --install

# Then open Ubuntu and run:
cd /home/user/earthion
```

Or use **Docker**:

```powershell
# Install Docker Desktop

# Run everything in containers:
docker-compose up -d
```

---

#### 🍎 Mac

```bash
# Open Terminal

# Install Go (if not installed)
brew install go

# Install Python
brew install python@3.13

# Install Node.js
brew install node

# Then run the commands from Quick Start above
```

---

#### 🐧 Linux (Ubuntu/Debian)

```bash
# Open Terminal

# Update packages
sudo apt update && sudo apt upgrade -y

# Install Go
sudo apt install -y golang-go

# Install Python
sudo apt install -y python3 python3-pip python3-venv

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Clone and run
cd /home/user/earthion
```

---

## 📱 How to Use

### 1. Open the App

After starting all three servers, open your browser:

```
http://localhost:5173
```

### 2. What You'll See

```
┌──────────────────────────────────────────────────────────────────┐
│  🌎 Earthion                              Height: 5    ●Live │
├──────────────────────────────────────────────────────────────────┤
│                                                          │
│  ┌────────────────────┐  ┌──────────────────────────────┐  │
│  │ My Wallet           │  │ Recent Blocks                │  │
│  │                    │  │  #5  a1b2...  12:30  2tx │  │
│  │ Address: AbCd...   │  │  #4  9f8e...  12:25  1tx │  │
│  │                    │  │  #3  7d6c...  12:20  0tx │  │
│  │ Balance: 100 EIO  │  │  ...                      │  │
│  │                    │  └──────────────────────────┘  │
│  └────────────────────┘                               │
│                                                          │
│  ┌────────────────────┐  ┌──────────────────────────┐   │
│  │ Mining             │  │ Send EIO                 │   │
│  │                    │  │                         │   │
│  │ Height: 5          │  │ To: [___________]        │   │
│  │ Difficulty: 18    │  │ Amount: [___]          │   │
│  │ Reward: 50         │  │ [Send EIO]             │   │
│  │ [Mine Block]       │  │                         │   │
│  └────────────────────┘  └──────────────────────────┘   │
│                                                          │
└──────────────────────────────────────────────────────────────────┘
```

### 3. What You Can Do

| Action | How |
|--------|-----|
| **View Balance** | Automatically shown |
| **Mine Blocks** | Click "Mine Block" button |
| **Send Coins** | Enter address + amount, click Send |
| **View History** | See in "Recent Blocks" |

---

## 🔧 Troubleshooting

### Error: "Port already in use"

```bash
# Find what's using the port
lsof -i :5173  # or :8000 or :9333

# Kill it
kill -9 <PID>
```

### Error: "Connection failed"

1. Make sure all servers are running
2. Check the ports:
   - `:5173` - Frontend
   - `:8000` - Python API
   - `:9333` - Go Blockchain

### Error: "python: command not found"

```bash
# Use python3 instead
python3 -m uvicorn main:app
```

---

## 📝 Key Concepts (Plain English)

### Block
A "page" in the blockchain notebook. Contains transactions.

### Transaction
A record of coins being sent from one person to another.

### Mining
The process of adding new blocks. It requires computer work (proof of work).

### Hash
A unique fingerprint for each block. If you change anything, the hash changes.

### Wallet
Your personal account. Contains your address (like account number) and balance.

### UTXO (Unspent Transaction Output)
A fancy way of saying "coins you have that you can spend."

---

## 📂 Project Structure

```
earthion/
│
├── earthion-blockchain-dev/          # The core blockchain
│   ├── cmd/                         # Entry points
│   │   ├── main.go                   # CLI version
│   │   └── server/                   # HTTP server
│   ├── core/                        # Core logic
│   │   ├── blockchain.go             # The chain
│   │   ├── block.go                  # Blocks
│   │   ├── transaction.go            # Transactions
│   │   └── pow.go                    # Mining
│   ├── wallet/                       # Wallet
│   ├── p2p/                         # Network
│   └── storage/                      # Saving/loading
│
└── earthion-backend/                 # API + Frontend
    ├── main.py                      # FastAPI server
    ├── app/
    │   ├── api/                     # API endpoints
    │   ├── services/                 # Blockchain client
    │   └── config.py                 # Settings
    └── frontend/                     # React app
        ├── src/
        │   ├── components/            # UI parts
        │   └── services/             # API calls
        └── package.json
```

---

## 🙏 Credits

Built with:
- **Go** - Blockchain core
- **Python** - FastAPI
- **React + Vite** - Frontend
- **Tailwind CSS** - Styling

---

## 📜 License

MIT License - Use it however you want!

---

*Made with 💜 for anyone who wants to understand blockchain.*