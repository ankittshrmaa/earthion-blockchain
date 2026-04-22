package http

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"earthion/core"
	"earthion/wallet"
)

// Handler handles HTTP API requests
type Handler struct {
	bc  *core.Blockchain
	wallet *wallet.Wallet
}

// NewHandler creates a new API handler
func NewHandler(bc *core.Blockchain, wal *wallet.Wallet) *Handler {
	return &Handler{
		bc:      bc,
		wallet:  wal,
	}
}

// =============================================================================
// Chain Endpoints
// =============================================================================

// ChainHeight returns the current chain height
func (h *Handler) ChainHeight(w http.ResponseWriter, r *http.Request) {
	height := h.bc.ChainHeight()
	json.NewEncoder(w).Encode(map[string]int{"height": height})
}

// Validate validates the entire blockchain
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	valid := h.bc.Validate()
	json.NewEncoder(w).Encode(map[string]bool{"valid": valid})
}

// UTXO returns all UTXOs
func (h *Handler) UTXO(w http.ResponseWriter, r *http.Request) {
	utxo := h.bc.UTXOIndex()
	utxos := make([]map[string]interface{}, 0)
	for key, out := range utxo {
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			continue
		}
		utxos = append(utxos, map[string]interface{}{
			"txid":   parts[0],
			"outIdx": parts[1],
			"value":  out.Value,
			"pubkey": hex.EncodeToString(out.PubKey),
		})
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"utxo": utxos, "count": len(utxos)})
}

// =============================================================================
// Block Endpoints
// =============================================================================

// GetBlocks returns all blocks (with optional limit)
func (h *Handler) GetBlocks(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if limit > 100 {
		limit = 100
	}

	blocks := h.bc.Blocks
	if len(blocks) > limit {
		blocks = blocks[len(blocks)-limit:]
	}

	response := make([]map[string]interface{}, len(blocks))
	for i, b := range blocks {
		response[i] = map[string]interface{}{
			"index":       b.Index,
			"timestamp":   b.Timestamp,
			"hash":       hex.EncodeToString(b.Hash),
			"prevHash":   hex.EncodeToString(b.PrevHash),
			"merkleRoot": hex.EncodeToString(b.MerkleRoot),
			"nonce":      b.Nonce,
			"difficulty": b.Difficulty,
			"txCount":   len(b.Transactions),
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"blocks": response, "count": len(response)})
}

// GetBlockByHash returns a block by hash
func (h *Handler) GetBlockByHash(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimPrefix(r.URL.Path, "/api/blocks/")
	if hash == "" {
		http.Error(w, "missing hash", http.StatusBadRequest)
		return
	}

	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		http.Error(w, "invalid hash", http.StatusBadRequest)
		return
	}

	block := h.bc.GetBlock(hashBytes)
	if block == nil {
		http.Error(w, "block not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"index":       block.Index,
		"timestamp":   block.Timestamp,
		"hash":       hex.EncodeToString(block.Hash),
		"prevHash":   hex.EncodeToString(block.PrevHash),
		"merkleRoot": hex.EncodeToString(block.MerkleRoot),
		"nonce":      block.Nonce,
		"difficulty": block.Difficulty,
		"txCount":   len(block.Transactions),
		"transactions": func() []map[string]interface{} {
			txs := make([]map[string]interface{}, len(block.Transactions))
			for i, tx := range block.Transactions {
				txs[i] = map[string]interface{}{
					"id":     hex.EncodeToString(tx.ID),
					"inputs": len(tx.Inputs),
					"outputs": len(tx.Outputs),
				}
			}
			return txs
		}(),
	}
	json.NewEncoder(w).Encode(response)
}

// GetBlockByIndex returns a block by index
func (h *Handler) GetBlockByIndex(w http.ResponseWriter, r *http.Request) {
	var index int
	_, err := fmt.Sscanf(r.URL.Path, "/api/blocks/index/%d", &index)
	if err != nil {
		http.Error(w, "invalid index", http.StatusBadRequest)
		return
	}

	block := h.bc.GetBlockByIndex(index)
	if block == nil {
		http.Error(w, "block not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"index":       block.Index,
		"hash":       hex.EncodeToString(block.Hash),
		"prevHash":   hex.EncodeToString(block.PrevHash),
		"merkleRoot": hex.EncodeToString(block.MerkleRoot),
		"nonce":      block.Nonce,
		"difficulty": block.Difficulty,
		"txCount":   len(block.Transactions),
	}
	json.NewEncoder(w).Encode(response)
}

// =============================================================================
// Wallet Endpoints
// =============================================================================

// GetAddress returns the wallet address
func (h *Handler) GetAddress(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"address": h.wallet.AddressHex()})
}

// GetBalance returns the wallet balance
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	balance := h.bc.GetBalance(h.wallet.GetRawAddress())
	json.NewEncoder(w).Encode(map[string]int{"balance": balance})
}

// Send handles sending coins
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		To     string `json:"to"`
		Amount int    `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	toAddr, err := hex.DecodeString(req.To)
	if err != nil {
		http.Error(w, "invalid address", http.StatusBadRequest)
		return
	}

	// Create transaction
	tx, err := core.NewTransaction(
		h.wallet,
		toAddr,
		req.Amount,
		h.bc,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tx == nil {
		http.Error(w, "failed to create transaction", http.StatusInternalServerError)
		return
	}

	// Add to blockchain
	h.bc.AddBlock([]*core.Transaction{tx})

	json.NewEncoder(w).Encode(map[string]string{
		"txid": hex.EncodeToString(tx.ID),
	})
}

// =============================================================================
// Mining Endpoints
// =============================================================================

// Mine mines new blocks
func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create coinbase transaction
	height := h.bc.ChainHeight()
	cbTx := core.CoinbaseTx(h.wallet.GetRawAddress(), core.GetBlockReward(height), height, nil)
	if cbTx == nil {
		http.Error(w, "failed to create coinbase", http.StatusInternalServerError)
		return
	}

	// Mine block
	h.bc.AddBlock([]*core.Transaction{cbTx})

	block := h.bc.LastBlock()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"hash":   hex.EncodeToString(block.Hash),
		"index":  block.Index,
		"nonce":  block.Nonce,
		"reward": core.GetBlockReward(block.Index),
	})
}

// GetReward returns the current mining reward
func (h *Handler) GetReward(w http.ResponseWriter, r *http.Request) {
	height := h.bc.ChainHeight()
	reward := core.GetBlockReward(height)
	json.NewEncoder(w).Encode(map[string]int{"reward": reward, "height": height})
}

// =============================================================================
// Stats Endpoints
// =============================================================================

// Stats returns blockchain statistics
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	height := h.bc.ChainHeight()
	totalWork := h.bc.TotalWork()

	// Calculate total supply
	supply := 0
	for _, b := range h.bc.Blocks {
		for _, tx := range b.Transactions {
			if tx.IsCoinbase() && len(tx.Outputs) > 0 {
				supply += tx.Outputs[0].Value
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"height":        height,
		"difficulty":    core.CurrentDifficulty(h.bc.Blocks),
		"totalWork":     totalWork,
		"totalSupply":   supply,
		"lastBlock":    hex.EncodeToString(h.bc.LastBlock().Hash),
	})
}

// Health returns health status
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	valid := h.bc.Validate()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"height": h.bc.ChainHeight(),
		"valid":  valid,
	})
}