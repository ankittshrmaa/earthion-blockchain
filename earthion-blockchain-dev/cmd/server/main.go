package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"earthion/core"
	"earthion/p2p"
	"earthion/storage"
	"earthion/wallet"
)

const (
	WalletFile = "wallet.dat"
	ChainFile  = "blockchain.dat"
	DefaultPort = "8333"
)

var (
	bc          *core.Blockchain
	wal        *wallet.Wallet
	p2pServer  *p2p.Server
	localNode  *p2p.LocalNode
	mempool    *core.Mempool      // Transaction mempool
	miningMu   sync.Mutex         // Prevents race conditions in mining
)

// API Response types
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type BlockJSON struct {
	Index        int                `json:"index"`
	Timestamp    int64              `json:"timestamp"`
	PrevHash     string             `json:"prevHash"`
	MerkleRoot   string             `json:"merkleRoot"`
	Hash         string             `json:"hash"`
	Nonce        int                `json:"nonce"`
	Difficulty   uint32             `json:"difficulty"`
	Transactions []TransactionJSON `json:"transactions"`
}

type TransactionJSON struct {
	ID        string             `json:"id"`
	Inputs    []TXInputJSON      `json:"inputs"`
	Outputs   []TXOutputJSON     `json:"outputs"`
}

type TXInputJSON struct {
	Txid     string `json:"txid"`
	OutIndex int    `json:"outIndex"`
	Signature string `json:"signature"`
	PubKey   string `json:"pubKey"`
}

type TXOutputJSON struct {
	Value  int    `json:"value"`
	PubKey string `json:"pubKey"`
}

type UTXOJSON struct {
	Key   string `json:"key"`
	Value TXOutputJSON `json:"value"`
}

type StatsJSON struct {
	Height          int    `json:"height"`
	Difficulty      uint32 `json:"difficulty"`
	TotalWork       int    `json:"totalWork"`
	CurrentReward   int    `json:"currentReward"`
	TotalMined     int    `json:"totalMined"`
	MaxSupply      int    `json:"maxSupply"`
}

// Helper functions
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func blockToJSON(b *core.Block) BlockJSON {
	bj := BlockJSON{
		Index:      b.Index,
		Timestamp:  b.TimestampInt(),
		PrevHash:   hex.EncodeToString(b.PrevHash),
		MerkleRoot: hex.EncodeToString(b.MerkleRoot),
		Hash:       hex.EncodeToString(b.Hash),
		Nonce:      b.Nonce,
		Difficulty: b.Difficulty,
	}

	bj.Transactions = make([]TransactionJSON, len(b.Transactions))
	for i, tx := range b.Transactions {
		bj.Transactions[i] = txToJSON(tx)
	}

	return bj
}

func txToJSON(tx *core.Transaction) TransactionJSON {
	tj := TransactionJSON{
		ID: hex.EncodeToString(tx.ID),
	}

	tj.Inputs = make([]TXInputJSON, len(tx.Inputs))
	for i, in := range tx.Inputs {
		tj.Inputs[i] = TXInputJSON{
			Txid:      hex.EncodeToString(in.Txid),
			OutIndex:  in.OutIndex,
			Signature: hex.EncodeToString(in.Signature),
			PubKey:    hex.EncodeToString(in.PubKey),
		}
	}

	tj.Outputs = make([]TXOutputJSON, len(tx.Outputs))
	for i, out := range tx.Outputs {
		tj.Outputs[i] = TXOutputJSON{
			Value:  out.Value,
			PubKey: hex.EncodeToString(out.PubKey),
		}
	}

	return tj
}

// Handlers

// GET /api/blocks - Get all blocks
func handleGetBlocks(w http.ResponseWriter, r *http.Request) {
	blocks := make([]BlockJSON, len(bc.Blocks))
	for i, b := range bc.Blocks {
		blocks[i] = blockToJSON(b)
	}
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    blocks,
	})
}

// GET /api/blocks/:hash - Get block by hash
func handleGetBlockByHash(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if hash == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "hash required"})
		return
	}

	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid hash format"})
		return
	}

	block := bc.GetBlock(hashBytes)
	if block == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "block not found"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    blockToJSON(block),
	})
}

// GET /api/blocks/index/:n - Get block by index
func handleGetBlockByIndex(w http.ResponseWriter, r *http.Request) {
	var index int
	_, err := fmt.Sscanf(r.PathValue("index"), "%d", &index)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid index"})
		return
	}

	block := bc.GetBlockByIndex(index)
	if block == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "block not found"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    blockToJSON(block),
	})
}

// GET /api/balance/:address - Get balance for address
func handleGetBalance(w http.ResponseWriter, r *http.Request) {
	address := r.PathValue("address")
	if address == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "address required"})
		return
	}

	addressBytes, err := hex.DecodeString(address)
	if err != nil || len(addressBytes) != 20 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid address (need 20 bytes)"})
		return
	}

	balance := bc.GetBalance(addressBytes)
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]int{"balance": balance},
	})
}

// GET /api/utxo - Get all UTXOs
func handleGetUTXO(w http.ResponseWriter, r *http.Request) {
	utxoMap := bc.UTXOIndex()
	utxos := make([]UTXOJSON, 0, len(utxoMap))

	for key, out := range utxoMap {
		utxos = append(utxos, UTXOJSON{
			Key: key,
			Value: TXOutputJSON{
				Value:  out.Value,
				PubKey: hex.EncodeToString(out.PubKey),
			},
		})
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    utxos,
	})
}

// GET /api/mempool - Get all transactions in mempool
func handleGetMempool(w http.ResponseWriter, r *http.Request) {
	txs := mempool.List()
	txList := make([]TransactionJSON, len(txs))

	for i, tx := range txs {
		txList[i] = txToJSON(tx)
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"transactions": txList,
			"count":       len(txList),
			"totalFees":   calculateMempoolFees(),
		},
	})
}

// calculateMempoolFees calculates total fees in mempool
func calculateMempoolFees() int {
	total := 0
	for _, tx := range mempool.List() {
		fee := core.GetTxFee(tx, bc)
		if fee > 0 {
			total += fee
		}
	}
	return total
}

// GET /api/transaction/:txid - Get transaction by ID
func handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	txid := r.PathValue("txid")
	if txid == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "txid required"})
		return
	}

	txidBytes, err := hex.DecodeString(txid)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid txid format"})
		return
	}

	tx := bc.FindTransaction(txidBytes)
	if tx == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "transaction not found"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    txToJSON(tx),
	})
}

// POST /api/transaction - Submit new transaction to mempool
func handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		To     string `json:"to"`
		Amount int    `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	// Validate inputs
	if len(req.To) != 40 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid address (need 40 hex chars)"})
		return
	}
	if req.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "amount must be positive"})
		return
	}

	toBytes, err := hex.DecodeString(req.To)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid to address"})
		return
	}

	tx, err := core.NewTransaction(wal, toBytes, req.Amount, bc)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}

	// Add transaction to mempool instead of directly to blockchain
	if err := mempool.Add(tx); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: fmt.Sprintf("failed to add to mempool: %v", err)})
		return
	}

	log.Printf("Transaction %s added to mempool (fee: %d)", hex.EncodeToString(tx.ID)[:8], core.GetTxFee(tx, bc))

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    txToJSON(tx),
	})
}

// POST /api/mine - Mine a new block
func handleMine(w http.ResponseWriter, r *http.Request) {
	// Lock to prevent race conditions in mining
	miningMu.Lock()
	defer miningMu.Unlock()

	// Get previous block hash for coinbase uniqueness
	var prevBlockHash []byte
	if len(bc.Blocks) > 0 {
		prevBlockHash = bc.Blocks[len(bc.Blocks)-1].Hash
	}

	// Get block reward with halving
	blockIndex := len(bc.Blocks)
	reward := core.GetBlockReward(blockIndex)

	// Get transactions from mempool
	memPoolTxs := mempool.List()

	// Calculate total fees from mempool transactions
	totalFees := 0
	for _, tx := range memPoolTxs {
		fee := core.GetTxFee(tx, bc)
		if fee > 0 {
			totalFees += fee
		}
	}

	// Add fees to reward
	totalReward := reward + totalFees

	// Create coinbase transaction with reward + fees
	coinbase := core.CoinbaseTx(wal.GetRawAddress(), totalReward, blockIndex, prevBlockHash)

	// Prepare all transactions: coinbase + mempool transactions
	allTxs := []*core.Transaction{coinbase}
	allTxs = append(allTxs, memPoolTxs...)

	// Add block to chain (triggers PoW)
	bc.AddBlock(allTxs)

	// Remove mined transactions from mempool
	for _, tx := range memPoolTxs {
		mempool.Remove(tx.ID)
	}

	// Get the new block
	newBlock := bc.LastBlock()

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    blockToJSON(newBlock),
	})
}

// GET /api/validate - Validate chain
func handleValidate(w http.ResponseWriter, r *http.Request) {
	valid := bc.Validate()
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]bool{"valid": valid},
	})
}

// GET /api/stats - Get chain statistics
func handleStats(w http.ResponseWriter, r *http.Request) {
	height := bc.ChainHeight()
	var diff uint32
	if height > 0 {
		diff = bc.LastBlock().Difficulty
	}

	reward := core.GetBlockReward(height)
	totalMined := core.GetTotalMined(height - 1)
	maxSupply := core.GetMaxSupply()

	stats := StatsJSON{
		Height:         height,
		Difficulty:     diff,
		TotalWork:      bc.TotalWork(),
		CurrentReward:  reward,
		TotalMined:     totalMined,
		MaxSupply:      maxSupply,
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// GET /api/wallet/address - Get wallet address
func handleGetWalletAddress(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"address": wal.AddressHex(),
			"raw":     hex.EncodeToString(wal.GetRawAddress()),
		},
	})
}

// Root endpoint
func handleRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"service": "Earthion Blockchain API", "version": "1.0.0"},
	})
}

// Health check
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"status": "healthy"},
	})
}

// Initialize blockchain and wallet
func initialize() {
	// Load or create wallet
	var err error
	wal, err = wallet.LoadWallet(WalletFile)
	if err != nil {
		wal, err = wallet.NewWallet()
		if err != nil {
			log.Fatalf("Failed to create wallet: %v", err)
		}
		if err := wal.Save(WalletFile); err != nil {
			log.Fatalf("Failed to save wallet: %v", err)
		}
		log.Println("New wallet created!")
	}

	// Load or create blockchain - auto-detect format (JSON or MessagePack)
	bc, err = storage.AutoDetectLoad(ChainFile)
	if err != nil {
		if os.IsNotExist(err) {
			bc = core.NewBlockchain()
			log.Println("New blockchain created!")
		} else {
			log.Printf("Error loading chain: %v, creating new...\n", err)
			bc = core.NewBlockchain()
		}
	}

	// Enable auto-save
	bc.SetFilename(ChainFile)

	// Initialize mempool
	mempool = core.NewMempool()

	log.Printf("Wallet loaded: %s", wal.AddressHex())
	log.Printf("Chain height: %d blocks", bc.ChainHeight())
	log.Printf("Mempool initialized")
}

func main() {
	initialize()

	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	// Register handlers
	mux := http.NewServeMux()

	// Root
	mux.HandleFunc("/", handleRoot)

	// Health
	mux.HandleFunc("/health", handleHealth)

	// Blocks
	mux.HandleFunc("GET /api/blocks", handleGetBlocks)
	mux.HandleFunc("GET /api/blocks/{hash}", handleGetBlockByHash)
	mux.HandleFunc("GET /api/blocks/index/{index}", handleGetBlockByIndex)

	// Wallet & Balance
	mux.HandleFunc("GET /api/wallet/address", handleGetWalletAddress)
	mux.HandleFunc("GET /api/balance/{address}", handleGetBalance)

	// P2P
	mux.HandleFunc("GET /api/peers", handlePeers)
	mux.HandleFunc("POST /api/connect", handleConnectPeer)

	// UTXO
	mux.HandleFunc("GET /api/utxo", handleGetUTXO)

	// Mempool
	mux.HandleFunc("GET /api/mempool", handleGetMempool)

	// Transactions
	mux.HandleFunc("GET /api/transaction/{txid}", handleGetTransaction)
	mux.HandleFunc("POST /api/transaction", handleCreateTransaction)

	// Mining
	mux.HandleFunc("POST /api/mine", handleMine)

	// Chain
	mux.HandleFunc("GET /api/validate", handleValidate)
	mux.HandleFunc("GET /api/stats", handleStats)

	// CLI compatibility - create a basic CLI wrapper for backward compatibility
	cliHandler := func(w http.ResponseWriter, r *http.Request) {
		// Simple CLI passthrough for testing
		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    map[string]string{"mode": "API server"},
		})
	}
	mux.HandleFunc("/cli", cliHandler)

	// Auto-save on shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down, saving blockchain...")
		if err := storage.SaveBlockchain(bc, ChainFile); err != nil {
			log.Printf("Error saving chain: %v", err)
		} else {
			log.Println("Blockchain saved!")
		}
		os.Exit(0)
	}()

	// Start P2P server
	p2pCfg := p2p.DefaultConfig()
	p2pCfg.ListenAddr = ":" + p2pPort()
	
	localNode, _ = p2p.NewLocalNode()
	p2pServer = p2p.NewServer(p2pCfg, &chainAdapter{bc})
	
	if err := p2pServer.Start(localNode); err != nil {
		log.Printf("P2P server failed: %v", err)
	} else {
		log.Printf("P2P server listening on %s", p2pServer.Addr())
	}

	// Connect to bootstrap nodes
	go connectToPeers()

	log.Printf("Starting Earthion API server on port %s...", port)
	log.Printf("Endpoints:")
	log.Printf("  GET  /health              - Health check")
	log.Printf("  GET  /api/blocks          - Get all blocks")
	log.Printf("  GET  /api/blocks/:hash    - Get block by hash")
	log.Printf("  GET  /api/blocks/index/:n - Get block by index")
	log.Printf("  GET  /api/wallet/address  - Get wallet address")
	log.Printf("  GET  /api/balance/:addr   - Get balance")
	log.Printf("  GET  /api/utxo            - Get UTXO set")
	log.Printf("  GET  /api/transaction/:txid - Get transaction")
	log.Printf("  POST /api/transaction     - Create transaction")
	log.Printf("  POST /api/mine            - Mine block")
	log.Printf("  GET  /api/validate        - Validate chain")
	log.Printf("  GET  /api/stats           - Chain statistics")
	log.Printf("  GET  /api/peers           - P2P peers")
	log.Printf("  POST /api/connect         - Connect to peer")

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// =============================================================================
// P2P Integration
// =============================================================================

func p2pPort() string {
	if p := os.Getenv("P2P_PORT"); p != "" {
		return p
	}
	return "8333"
}

type chainAdapter struct {
	bc *core.Blockchain
}

func (ca *chainAdapter) GetBestHash() []byte {
	if ca.bc == nil {
		return nil
	}
	return ca.bc.LastBlock().Hash
}

func (ca *chainAdapter) GetBestHeight() int {
	if ca.bc == nil {
		return 0
	}
	return ca.bc.ChainHeight()
}

func (ca *chainAdapter) GetBlock(hash []byte) (*core.Block, error) {
	block := ca.bc.GetBlock(hash)
	if block == nil {
		return nil, fmt.Errorf("block not found")
	}
	return block, nil
}

func (ca *chainAdapter) AddBlock(block *core.Block) error {
	// AddBlock expects []*Transaction, extract from block
	ca.bc.AddBlock(block.Transactions)
	return nil
}

func (ca *chainAdapter) GetBlockHash(height int) ([]byte, error) {
	block := ca.bc.GetBlockByIndex(height)
	if block == nil {
		return nil, fmt.Errorf("block not found")
	}
	return block.Hash, nil
}

func (ca *chainAdapter) GetBlockHeight(hash []byte) (int, error) {
	block := ca.bc.GetBlock(hash)
	if block == nil {
		return 0, fmt.Errorf("block not found")
	}
	return block.Index, nil
}

func connectToPeers() {
	peers := []string{"127.0.0.1:8334", "127.0.0.1:8335"}
	for _, addr := range peers {
		if err := p2pServer.ConnectToPeer(addr, p2p.NodeID{}); err != nil {
			log.Printf("Failed to connect to %s: %v", addr, err)
		} else {
			log.Printf("Connected to peer: %s", addr)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// =============================================================================
// API Handlers
// =============================================================================

func handlePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	peers := p2pServer.Peers().List()
	type peerInfo struct {
		ID         string `json:"id"`
		Addr       string `json:"addr"`
		Height    int    `json:"height"`
		Direction string `json:"direction"`
	}

	var list []peerInfo
	for _, p := range peers {
		list = append(list, peerInfo{
			ID:         p.NodeIDString()[:8],
			Addr:       p.RemoteAddress(),
			Height:    p.BestHeight,
			Direction: "outbound",
		})
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: list})
}

func handleConnectPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Addr string `json:"addr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := p2pServer.ConnectToPeer(req.Addr, p2p.NodeID{}); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true})
}
