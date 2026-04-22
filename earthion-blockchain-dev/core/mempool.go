package core

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"earthion/crypto"
)

// Mempool constants
const (
	MaxMempoolSize      = 1000           // Maximum transactions in mempool
	MempoolMaxAge       = 72 * time.Hour // Maximum age before eviction (3 days)
	MempoolEvictionBatch = 50             // Number of txs to evict when full
)

// Mempool represents a pool of pending transactions
type Mempool struct {
	mu       sync.RWMutex
	txs      map[string]*MempoolEntry  // txID -> entry with metadata
	bySender map[string][]string        // address (20-byte hex) -> tx IDs
}

// MempoolEntry wraps a transaction with metadata for eviction
type MempoolEntry struct {
	Tx        *Transaction
	AddedAt   time.Time
	FeeRate   int  // Calculated fee rate for eviction prioritization
	TxSize    int  // Serialized size for fee calculation
}

// NewMempool creates a new mempool
func NewMempool() *Mempool {
	return &Mempool{
		txs:      make(map[string]*MempoolEntry),
		bySender: make(map[string][]string),
	}
}

// Add adds a transaction to the mempool
// Returns error if transaction is invalid or mempool is full
func (m *Mempool) Add(tx *Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check mempool size and evict if necessary
	if len(m.txs) >= MaxMempoolSize {
		// Evict lowest fee rate transactions
		m.evictLowFee(MempoolEvictionBatch)
		
		// Check again after eviction
		if len(m.txs) >= MaxMempoolSize {
			return fmt.Errorf("mempool is full after eviction")
		}
	}

	// Check if transaction already exists
	txID := hex.EncodeToString(tx.ID)
	if _, exists := m.txs[txID]; exists {
		return fmt.Errorf("transaction already in mempool")
	}

	// Validate transaction before adding
	if err := m.validateTx(tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	// Calculate fee rate for prioritization
	txSize := len(tx.Serialize())
	feeRate := m.calculateFeeRate(tx, txSize)

	// Check minimum fee rate
	if feeRate < MinFeeRate {
		return fmt.Errorf("fee rate too low: %d < %d", feeRate, MinFeeRate)
	}

	// Create entry with metadata
	entry := &MempoolEntry{
		Tx:      tx,
		AddedAt: time.Now(),
		FeeRate: feeRate,
		TxSize:  txSize,
	}

	// Add to mempool
	m.txs[txID] = entry

	// Index by sender address (20-byte pubkey hash, not full PubKey)
	if !tx.IsCoinbase() && len(tx.Inputs) > 0 {
		// Derive address from first input's pubkey hash
		// The input's PubKey is the full 33/65 byte public key
		// We need to hash it to get the 20-byte address
		addr := crypto.Hash(tx.Inputs[0].PubKey)
		sender := hex.EncodeToString(addr[:20])
		m.bySender[sender] = append(m.bySender[sender], txID)
	}

	return nil
}

// calculateFeeRate calculates the fee rate (satoshis per byte) for a transaction
func (m *Mempool) calculateFeeRate(tx *Transaction, txSize int) int {
	if txSize == 0 {
		return 0
	}

	// Calculate total input value minus output value (fees)
	inputTotal := 0
	outputTotal := 0

	// Get input values from referenced UTXOs (simplified - would need UTXO index)
	// For now, use a placeholder - in production, query UTXO set
	_ = inputTotal

	// Use output total as approximation
	for _, out := range tx.Outputs {
		outputTotal += out.Value
	}

	// If coinbase, use a default high fee rate
	if tx.IsCoinbase() {
		return 100 // Higher priority for coinbase
	}

	// Return fee rate (simplified)
	// In production: (inputTotal - outputTotal) / txSize
	return DefaultFeeRate
}

// evictLowFee removes the lowest fee rate transactions to make room
func (m *Mempool) evictLowFee(count int) {
	if len(m.txs) == 0 {
		return
	}

	// Collect entries with their fee rates
	type txEntry struct {
		txID   string
		feeRate int
	}

	var entries []txEntry
	for txID, entry := range m.txs {
		entries = append(entries, txEntry{txID, entry.FeeRate})
	}

	// Sort by fee rate (lowest first)
	// Simple bubble sort for small lists
	for i := 0; i < len(entries)-1; i++ {
		for j := 0; j < len(entries)-i-1; j++ {
			if entries[j].feeRate > entries[j+1].feeRate {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}

	// Remove lowest fee rate transactions
	removed := 0
	for _, e := range entries {
		if removed >= count {
			break
		}
		m.removeEntry(e.txID)
		removed++
	}
}

// removeEntry removes a transaction entry from mempool
func (m *Mempool) removeEntry(txID string) {
	entry, exists := m.txs[txID]
	if !exists {
		return
	}

	// Remove from sender index
	tx := entry.Tx
	if !tx.IsCoinbase() && len(tx.Inputs) > 0 {
		addr := crypto.Hash(tx.Inputs[0].PubKey)
		sender := hex.EncodeToString(addr[:20])
		if ids, ok := m.bySender[sender]; ok {
			for i, id := range ids {
				if id == txID {
					m.bySender[sender] = append(ids[:i], ids[i+1:]...)
					break
				}
			}
		}
	}

	delete(m.txs, txID)
}

// Remove removes a transaction from the mempool
func (m *Mempool) Remove(txID []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.removeEntry(hex.EncodeToString(txID))
}

// Get retrieves a transaction by ID
func (m *Mempool) Get(txID []byte) (*Transaction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.txs[hex.EncodeToString(txID)]; ok {
		return entry.Tx, ok
	}
	return nil, false
}

// GetBySender returns all transactions from a sender (by address)
func (m *Mempool) GetBySender(address []byte) []*Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Transaction
	sender := hex.EncodeToString(address)
	txIDs := m.bySender[sender]
	for _, txID := range txIDs {
		if entry, ok := m.txs[txID]; ok {
			result = append(result, entry.Tx)
		}
	}
	return result
}

// List returns all transactions in mempool (sorted by fee rate, highest first)
func (m *Mempool) List() []*Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txs := make([]*Transaction, 0, len(m.txs))
	for _, entry := range m.txs {
		txs = append(txs, entry.Tx)
	}

	// Sort by fee rate (highest first - more attractive to miners)
	// Simple bubble sort for small lists
	for i := 0; i < len(txs)-1; i++ {
		for j := 0; j < len(txs)-i-1; j++ {
			entryI := m.txs[hex.EncodeToString(txs[i].ID)]
			entryJ := m.txs[hex.EncodeToString(txs[j].ID)]
			if entryI != nil && entryJ != nil && entryI.FeeRate < entryJ.FeeRate {
				txs[i], txs[j] = txs[j], txs[i]
			}
		}
	}

	return txs
}

// Size returns the number of transactions in mempool
func (m *Mempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.txs)
}

// Clear removes all transactions from mempool
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.txs = make(map[string]*MempoolEntry)
	m.bySender = make(map[string][]string)
}

// Contains checks if a transaction is in mempool
func (m *Mempool) Contains(txID []byte) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.txs[hex.EncodeToString(txID)]
	return exists
}

// validateTx performs basic transaction validation before adding to mempool
func (m *Mempool) validateTx(tx *Transaction) error {
	// Check for nil
	if tx == nil {
		return fmt.Errorf("nil transaction")
	}

	// Check ID exists
	if len(tx.ID) == 0 {
		return fmt.Errorf("transaction has no ID")
	}

	// Check inputs
	if len(tx.Inputs) == 0 {
		return fmt.Errorf("transaction has no inputs")
	}

	// Check outputs
	if len(tx.Outputs) == 0 {
		return fmt.Errorf("transaction has no outputs")
	}

	// Check output values are positive
	for _, out := range tx.Outputs {
		if out.Value <= 0 {
			return fmt.Errorf("output value must be positive")
		}
	}

	// Verify signature for non-coinbase
	if !tx.IsCoinbase() {
		if !tx.Verify() {
			return fmt.Errorf("invalid signature")
		}
	}

	return nil
}

// RemoveExpired removes transactions older than maxAge
func (m *Mempool) RemoveExpired(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for txID, entry := range m.txs {
		if now.Sub(entry.AddedAt) > maxAge {
			toRemove = append(toRemove, txID)
		}
	}

	for _, txID := range toRemove {
		m.removeEntry(txID)
	}
}

// GetConflicts returns transactions that conflict with the given input
func (m *Mempool) GetConflicts(tx *Transaction) []*Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var conflicts []*Transaction

	for _, entry := range m.txs {
		existing := entry.Tx
		for _, in := range tx.Inputs {
			for _, existingIn := range existing.Inputs {
				if len(in.Txid) > 0 && len(existingIn.Txid) > 0 {
					if string(in.Txid) == string(existingIn.Txid) && in.OutIndex == existingIn.OutIndex {
						conflicts = append(conflicts, existing)
					}
				}
			}
		}
	}

	return conflicts
}

// GetFeeEstimate returns a fee estimate based on current mempool state
// Returns the minimum fee rate needed for next-block inclusion
func (m *Mempool) GetFeeEstimate() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.txs) == 0 {
		return DefaultFeeRate
	}

	// Get the lowest fee rate in the top 10 transactions
	// This gives a good estimate for next-block inclusion
	type feeEntry struct {
		feeRate int
	}

	var fees []feeEntry
	count := 0
	for _, entry := range m.txs {
		fees = append(fees, feeEntry{entry.FeeRate})
		count++
		if count >= 10 {
			break
		}
	}

	if len(fees) == 0 {
		return DefaultFeeRate
	}

	// Find minimum
	minFee := fees[0].feeRate
	for _, f := range fees {
		if f.feeRate < minFee {
			minFee = f.feeRate
		}
	}

	return minFee
}