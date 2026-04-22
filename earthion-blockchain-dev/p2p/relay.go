package p2p

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"earthion/core"
)

// =============================================================================
// Transaction Relay
// =============================================================================

// MempoolManager manages the transaction memory pool
type MempoolManager struct {
	localNode *LocalNode
	peers     *PeerPool

	mu       sync.RWMutex
	txs      map[string]*core.Transaction // txid -> tx
	txExpiry time.Time

	// Callbacks
	onNewTX func(*core.Transaction)
}

// NewMempoolManager creates a new mempool manager
func NewMempoolManager(local *LocalNode, peers *PeerPool) *MempoolManager {
	return &MempoolManager{
		localNode: local,
		peers:    peers,
		txs:      make(map[string]*core.Transaction),
		txExpiry: time.Now().Add(3 * 24 * time.Hour), // Default 3-day expiry
	}
}

// AddTransaction adds a transaction to the mempool
func (mm *MempoolManager) AddTransaction(tx *core.Transaction) error {
	txid := string(tx.ID)

	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Check if already in mempool
	if _, exists := mm.txs[txid]; exists {
		return fmt.Errorf("transaction already in mempool")
	}

	// Verify transaction before accepting
	if !tx.Verify() {
		return fmt.Errorf("invalid transaction")
	}

	mm.txs[txid] = tx

	// Broadcast to other peers
	mm.broadcastTX(tx, nil)

	fmt.Printf("[mempool] Added tx %x\n", tx.ID[:8])

	return nil
}

// GetTransaction returns a transaction by txid
func (mm *MempoolManager) GetTransaction(txid string) (*core.Transaction, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	tx, ok := mm.txs[txid]
	return tx, ok
}

// GetTransactions returns all transactions in mempool
func (mm *MempoolManager) GetTransactions() []*core.Transaction {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	txs := make([]*core.Transaction, 0, len(mm.txs))
	for _, tx := range mm.txs {
		txs = append(txs, tx)
	}
	return txs
}

// RemoveTransaction removes a transaction from mempool
func (mm *MempoolManager) RemoveTransaction(txid string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	delete(mm.txs, txid)
}

// Count returns the number of transactions
func (mm *MempoolManager) Count() int {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return len(mm.txs)
}

// HandleTX handles an incoming transaction
func (mm *MempoolManager) HandleTX(peer *Peer, tx *core.Transaction) error {
	txid := string(tx.ID)

	// Check if already have it
	mm.mu.RLock()
	_, exists := mm.txs[txid]
	mm.mu.RUnlock()

	if exists {
		return nil // Already have it
	}

	// Verify
	if !tx.Verify() {
		// Could send reject message
		return fmt.Errorf("verify failed")
	}

	// Add to mempool
	mm.mu.Lock()
	mm.txs[txid] = tx
	mm.mu.Unlock()

	// Relay to other peers
	mm.broadcastTX(tx, peer)

	if mm.onNewTX != nil {
		mm.onNewTX(tx)
	}

	return nil
}

// broadcastTX broadcasts a transaction to peers
func (mm *MempoolManager) broadcastTX(tx *core.Transaction, from *Peer) {
	// Create inv message
	inv := NewInvMessage()
	inv.AddTx(tx.ID)

	msg := &Message{
		Type:    MsgInv,
		Payload: inv.Serialize(),
	}

	// Broadcast to all except sender
	mm.peers.Broadcast(msg, from)
}

// HandleInv handles inventory message
func (mm *MempoolManager) HandleInv(peer *Peer, inv *InvMessage) error {
	// Filter to only transactions
	var wanted [][]byte

	for _, v := range inv.Vectors {
		if v.Type == InvTypeTX {
			txid := hex.EncodeToString(v.Hash[:])
			
			// Check if we already have it
			mm.mu.RLock()
			_, have := mm.txs[txid]
			mm.mu.RUnlock()
			
			if !have {
				hash := make([]byte, 32)
				copy(hash, v.Hash[:])
				wanted = append(wanted, hash)
			}
		}
	}

	if len(wanted) == 0 {
		return nil
	}

	// Request transactions
	msg := &Message{
		Type:    MsgGetData,
		Payload: mm.serializeGetData(wanted),
	}

	select {
	case peer.MessageChan() <- msg:
	default:
	}

	return nil
}

func (mm *MempoolManager) serializeGetData(hashes [][]byte) []byte {
	// Simplified - use proper inventory vector format
	count := len(hashes)
	size := 1 + count*32
	buf := make([]byte, size)

	buf[0] = byte(count)
	offset := 1

	for _, hash := range hashes {
		copy(buf[offset:offset+32], hash)
		offset += 32
	}

	return buf
}

// Cleanup removes expired transactions
func (mm *MempoolManager) Cleanup() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// For now, just clear old transactions based on count limit
	// Could add timestamp tracking later
	if len(mm.txs) > 10000 {
		// Remove oldest half
		removed := 0
		for txid := range mm.txs {
			if removed < len(mm.txs)/2 {
				delete(mm.txs, txid)
				removed++
			} else {
				break
			}
		}
		fmt.Printf("[mempool] Cleaned up %d transactions\n", removed)
	}

	if len(mm.txs) > 10000 {
		fmt.Printf("[mempool] Cleaned up transactions\n")
	}
}

// SetOnNewTX sets the callback for new transactions
func (mm *MempoolManager) SetOnNewTX(f func(*core.Transaction)) {
	mm.onNewTX = f
}

// =============================================================================
// Block Relay
// =============================================================================

// BlockRelay manages block propagation
type BlockRelay struct {
	localNode *LocalNode
	peers    *PeerPool

	// Block cache (for recent blocks)
	mu    sync.RWMutex
	cache map[string]*core.Block // hash -> block
}

// NewBlockRelay creates a new block relay
func NewBlockRelay(local *LocalNode, peers *PeerPool) *BlockRelay {
	return &BlockRelay{
		localNode: local,
		peers:    peers,
		cache:    make(map[string]*core.Block),
	}
}

// HandleBlock handles an incoming block
func (br *BlockRelay) HandleBlock(peer *Peer, block *core.Block, prevBlock *core.Block) error {
	hash := string(hex.EncodeToString(block.Hash))

	// Check cache
	br.mu.RLock()
	_, exists := br.cache[hash]
	br.mu.RUnlock()

	if exists {
		return nil // Already have it
	}

	// Verify block using core validation
	if err := core.ValidateBlock(block, prevBlock); err != nil {
		return fmt.Errorf("verify: %w", err)
	}

	// Cache it
	br.mu.Lock()
	br.cache[hash] = block
	// Keep only last 100 blocks
	if len(br.cache) > 100 {
		// Remove oldest
	}
	br.mu.Unlock()

	// Relay to other peers
	br.relayBlock(block, peer)

	return nil
}

// AnnounceBlock announces a new block
func (br *BlockRelay) AnnounceBlock(block *core.Block) {
	br.relayBlock(block, nil)
}

// relayBlock relays a block to all peers except one
func (br *BlockRelay) relayBlock(block *core.Block, from *Peer) {
	// First send inv
	inv := NewInvMessage()
	inv.AddBlock(block.Hash)

	msg := &Message{
		Type:    MsgInv,
		Payload: inv.Serialize(),
	}

	br.peers.Broadcast(msg, from)

	// Then send the block data if peer requests
	// (Peer will request via GetData after seeing Inv)
}

// HandleInv handles inventory message
func (br *BlockRelay) HandleInv(peer *Peer, inv *InvMessage) error {
	var wanted [][]byte

	for _, v := range inv.Vectors {
		if v.Type == InvTypeBlock {
			// Check if we already have it
			br.mu.RLock()
			_, have := br.cache[hex.EncodeToString(v.Hash[:])]
			br.mu.RUnlock()

			if !have {
				hash := make([]byte, 32)
				copy(hash, v.Hash[:])
				wanted = append(wanted, hash)
			}
		}
	}

	if len(wanted) == 0 {
		return nil
	}

	// Request blocks
	msg := &Message{
		Type:    MsgGetData,
		Payload: serializeGetDataRequest(wanted),
	}

	select {
	case peer.MessageChan() <- msg:
	default:
	}

	return nil
}

// serializeGetDataRequest serializes a getdata request
func serializeGetDataRequest(hashes [][]byte) []byte {
	count := len(hashes)
	size := 1 + count*32
	buf := make([]byte, size)

	buf[0] = byte(count)
	offset := 1

	for _, hash := range hashes {
		copy(buf[offset:offset+32], hash)
		offset += 32
	}

	return buf
}

// GetBlock returns a cached block by hash
func (br *BlockRelay) GetBlock(hash string) (*core.Block, bool) {
	br.mu.RLock()
	defer br.mu.RUnlock()

	block, ok := br.cache[hash]
	return block, ok
}