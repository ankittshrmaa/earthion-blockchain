package core

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"earthion/crypto"
)

type Blockchain struct {
	Blocks       []*Block
	filename     string       // for auto-save
	orphaned     []*Block    // Orphaned blocks (for fork resolution)
	altChains    [][]*Block  // Alternative chains for fork handling

	// Index maps for O(1) lookups
	blockIndex  map[string]int  // hash -> block index
	txIndex     map[string]*Transaction  // txID -> transaction

	// UTXO Index for O(1) balance lookups
	// Key: "txID:outIdx", Value: TXOutput
	utxoIndex map[string]TXOutput

	// Address balance index for O(1) balance lookup
	// Key: 20-byte address, Value: balance
	addressBalance map[string]int
}

func NewBlockchain() *Blockchain {
	bc := &Blockchain{
		Blocks:    []*Block{GenesisBlock()},
		filename:  "",
		orphaned:  nil,
		altChains: nil,
	}
	// Initialize indices
	bc.blockIndex = make(map[string]int)
	bc.txIndex = make(map[string]*Transaction)
	bc.utxoIndex = make(map[string]TXOutput)
	bc.addressBalance = make(map[string]int)

	// Index genesis block and build initial UTXO set
	bc.rebuildIndex()
	bc.rebuildUTXOIndex()

	return bc
}

// SetFilename enables auto-save to the specified file
func (bc *Blockchain) SetFilename(filename string) {
	bc.filename = filename
}

func (bc *Blockchain) AddBlock(txs []*Transaction) {
	prev := bc.Blocks[len(bc.Blocks)-1]
	// Pass bc.Blocks for dynamic difficulty calculation
	newBlock := NewBlock(txs, prev.Hash, prev.Index+1, bc.Blocks)
	bc.Blocks = append(bc.Blocks, newBlock)

	// Update indices
	bc.updateIndex(newBlock)
	bc.updateUTXOIndex(newBlock)

	// Auto-save if filename set (using gob for fast internal serialization)
	if bc.filename != "" {
		if err := bc.saveToFile(bc.filename); err != nil {
			log.Printf("Auto-save failed: %v", err)
		}
	}
}

// rebuildIndex rebuilds all indices from blocks
func (bc *Blockchain) rebuildIndex() {
	bc.blockIndex = make(map[string]int)
	bc.txIndex = make(map[string]*Transaction)

	for i, block := range bc.Blocks {
		hash := hex.EncodeToString(block.Hash)
		bc.blockIndex[hash] = i

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
			bc.txIndex[txID] = tx
		}
	}

	// Also rebuild UTXO index
	bc.rebuildUTXOIndex()
}

// updateIndex adds a new block to indices
func (bc *Blockchain) updateIndex(block *Block) {
	hash := hex.EncodeToString(block.Hash)
	bc.blockIndex[hash] = block.Index

	for _, tx := range block.Transactions {
		txID := hex.EncodeToString(tx.ID)
		bc.txIndex[txID] = tx
	}
}

// rebuildUTXOIndex rebuilds the UTXO index from all blocks
// This is O(n) where n is total transactions in chain
func (bc *Blockchain) rebuildUTXOIndex() {
	bc.utxoIndex = make(map[string]TXOutput)
	bc.addressBalance = make(map[string]int)

	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

			// Add all outputs as potential UTXOs
			for outIdx, out := range tx.Outputs {
				key := fmt.Sprintf("%s:%d", txID, outIdx)
				bc.utxoIndex[key] = out

				// Update address balance
				addr := hex.EncodeToString(out.PubKey)
				bc.addressBalance[addr] += out.Value
			}

			// Remove spent inputs from UTXO set
			for _, in := range tx.Inputs {
				if in.OutIndex == -1 {
					continue // coinbase has no valid input
				}
				inTxID := hex.EncodeToString(in.Txid)
				key := fmt.Sprintf("%s:%d", inTxID, in.OutIndex)

				// Get the spent output to update balance
				if spentOut, exists := bc.utxoIndex[key]; exists {
					spentAddr := hex.EncodeToString(spentOut.PubKey)
					bc.addressBalance[spentAddr] -= spentOut.Value
				}

				// Remove from UTXO set
				delete(bc.utxoIndex, key)
			}
		}
	}
}

// updateUTXOIndex updates the UTXO index with a new block
func (bc *Blockchain) updateUTXOIndex(block *Block) {
	for _, tx := range block.Transactions {
		txID := hex.EncodeToString(tx.ID)

		// Add new outputs to UTXO set
		for outIdx, out := range tx.Outputs {
			key := fmt.Sprintf("%s:%d", txID, outIdx)
			bc.utxoIndex[key] = out

			// Update address balance
			addr := hex.EncodeToString(out.PubKey)
			bc.addressBalance[addr] += out.Value
		}

		// Remove spent inputs from UTXO set
		for _, in := range tx.Inputs {
			if in.OutIndex == -1 {
				continue
			}
			inTxID := hex.EncodeToString(in.Txid)
			key := fmt.Sprintf("%s:%d", inTxID, in.OutIndex)

			// Get the spent output to update balance
			if spentOut, exists := bc.utxoIndex[key]; exists {
				spentAddr := hex.EncodeToString(spentOut.PubKey)
				bc.addressBalance[spentAddr] -= spentOut.Value
			}

			// Remove from UTXO set
			delete(bc.utxoIndex, key)
		}
	}
}

// saveToFile persists blockchain to file (internal - uses gob for speed within Go)
// For external storage/exchange, use storage.SaveBlockchainMsgPack for MessagePack
func (bc *Blockchain) saveToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(bc)
}

// saveToFileMsgPack persists blockchain to file using MessagePack (more efficient)
// For now, uses gob - can be upgraded to msgpack by adding storage package properly later
func (bc *Blockchain) saveToFileMsgPack(filename string) error {
	return bc.saveToFile(filename)
}

// Validate checks the entire blockchain for integrity
func (bc *Blockchain) Validate() bool {
	if len(bc.Blocks) == 0 {
		return false
	}

	genesis := bc.Blocks[0]
	if !bytes.Equal(genesis.PrevHash, []byte{}) {
		log.Println("Genesis block has incorrect prevHash")
		return false
	}

	// Track spent outputs to detect double-spends
	spentOutputs := make(map[string]bool)

	for i := 1; i < len(bc.Blocks); i++ {
		block := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		if !bytes.Equal(block.PrevHash, prevBlock.Hash) {
			log.Printf("Block %d: prevHash doesn't match previous block hash\n", i)
			log.Printf("  block.PrevHash: %x\n", block.PrevHash[:8])
			log.Printf("  prevBlock.Hash: %x\n", prevBlock.Hash[:8])
			return false
		}

		pow := NewProofOfWork(block)
		if !pow.Validate() {
			// Debug: show hash comparison
			data := pow.prepareData(block.Nonce)
			calcHash := crypto.DoubleHash(data)
			log.Printf("Block %d: PoW validation failed\n", i)
			log.Printf("  stored Hash:   %x\n", block.Hash[:8])
			log.Printf("  calculated:  %x\n", calcHash[:8])
			log.Printf("  Nonce: %d, Difficulty: %d\n", block.Nonce, block.Difficulty)
			return false
		}

		// Validate Merkle root matches transactions
		txHashes := make([][]byte, len(block.Transactions))
		for j, tx := range block.Transactions {
			txHashes[j] = tx.ID
		}
		merkleTree := crypto.NewMerkleTree(txHashes)
		if !bytes.Equal(block.MerkleRoot, merkleTree.RootHash()) {
			log.Printf("Block %d: Merkle root mismatch\n", i)
			return false
		}

		for j, tx := range block.Transactions {
			if tx.IsCoinbase() {
				continue
			}

			// Check for double-spend within this transaction
			inputsSeen := make(map[string]bool)
			for _, in := range tx.Inputs {
				if in.OutIndex == -1 {
					continue
				}
				inTxID := hex.EncodeToString(in.Txid)
				key := fmt.Sprintf("%s:%d", inTxID, in.OutIndex)
				
				if inputsSeen[key] {
					log.Printf("Block %d: Transaction %d has double-spend input\n", i, j)
					return false
				}
				inputsSeen[key] = true

				// Check against previously spent outputs in the chain
				if spentOutputs[key] {
					log.Printf("Block %d: Transaction %d attempts to spend already-spent output\n", i, j)
					return false
				}
				spentOutputs[key] = true
			}

			if !tx.Verify() {
				log.Printf("Block %d: Transaction %d verification failed\n", i, j)
				return false
			}
		}
	}

	return true
}

func (bc *Blockchain) GetBlock(blockHash []byte) *Block {
	// Try index first (O(1))
	hash := hex.EncodeToString(blockHash)
	if idx, ok := bc.blockIndex[hash]; ok {
		return bc.Blocks[idx]
	}

	// Fallback to linear search
	for _, block := range bc.Blocks {
		if bytes.Equal(block.Hash, blockHash) {
			return block
		}
	}
	return nil
}

// GetBlockByIndex returns a block by its index (O(1))
func (bc *Blockchain) GetBlockByIndex(index int) *Block {
	if index >= 0 && index < len(bc.Blocks) {
		return bc.Blocks[index]
	}
	return nil
}

func (bc *Blockchain) FindTransaction(txID []byte) *Transaction {
	// Try index first (O(1))
	txIDHex := hex.EncodeToString(txID)
	if tx, ok := bc.txIndex[txIDHex]; ok {
		return tx
	}

	// Fallback to linear search
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, txID) {
				return tx
			}
		}
	}
	return nil
}

// UTXOIndex returns the precomputed UTXO index map in O(1) time
// Key: "txID:outIdx", Value: TXOutput
func (bc *Blockchain) UTXOIndex() map[string]TXOutput {
	return bc.utxoIndex
}

// GetBalance returns the balance for a given address in O(1) time
// Uses the precomputed addressBalance index
// Note: address should be the raw 20-byte pubkey hash (not Base58 encoded)
func (bc *Blockchain) GetBalance(address []byte) int {
	addr := hex.EncodeToString(address)
	return bc.addressBalance[addr]
}

// GetBalanceFromUTXO calculates balance by iterating UTXO set (for verification)
func (bc *Blockchain) GetBalanceFromUTXO(address []byte) int {
	balance := 0
	for _, out := range bc.utxoIndex {
		if bytes.Equal(out.PubKey, address) {
			balance += out.Value
		}
	}
	return balance
}

// =============================================================================
// FORK HANDLING
// =============================================================================

// TotalWork calculates the total PoW done on the chain
func (bc *Blockchain) TotalWork() int {
	work := 0
	for _, block := range bc.Blocks {
		// Each block contributes its difficulty in work
		work += int(block.Difficulty)
	}
	return work
}

// AddOrphanedBlock adds a block that doesn't connect to main chain
func (bc *Blockchain) AddOrphanedBlock(block *Block) {
	bc.orphaned = append(bc.orphaned, block)
}

// GetOrphanedBlocks returns all orphaned blocks
func (bc *Blockchain) GetOrphanedBlocks() []*Block {
	return bc.orphaned
}

// AddAlternativeChain stores an alternative chain (fork)
func (bc *Blockchain) AddAlternativeChain(blocks []*Block) {
	bc.altChains = append(bc.altChains, blocks)
}

// GetAlternativeChains returns all stored alternative chains
func (bc *Blockchain) GetAlternativeChains() [][]*Block {
	return bc.altChains
}

// AttemptReorg checks for longer valid chains and reorg if needed
// Returns true if reorganization occurred
func (bc *Blockchain) AttemptReorg() bool {
	// Check if any alternative chain is longer and valid
	for _, altChain := range bc.altChains {
		if len(altChain) <= len(bc.Blocks) {
			continue // Not longer
		}
		
		// Validate the alternative chain
		valid := true
		for _, block := range altChain {
			pow := NewProofOfWork(block)
			if !pow.Validate() {
				valid = false
				break
			}
		}
		
		if valid {
			log.Printf("Reorganization: switching to longer chain (%d -> %d blocks)\n", 
				len(bc.Blocks), len(altChain))
			
			// Store current chain as orphaned before switching
			bc.orphaned = append(bc.orphaned, bc.Blocks...)
			
			// Switch to new chain
			bc.Blocks = altChain
			bc.altChains = nil // Clear alternatives after reorg
			
			// CRITICAL: Rebuild indices after reorganization
			bc.rebuildIndex()
			
			return true
		}
	}
	
	return false
}

// ResolveForks checks all orphaned blocks for valid new chain tips
// Returns number of blocks incorporated
func (bc *Blockchain) ResolveForks() int {
	incorporated := 0
	currentTip := bc.Blocks[len(bc.Blocks)-1]
	
	// Try to connect orphaned blocks
	var remaining []*Block
	for _, block := range bc.orphaned {
		if bytes.Equal(block.PrevHash, currentTip.Hash) {
			// Can connect!
			bc.Blocks = append(bc.Blocks, block)
			currentTip = block
			incorporated++
		} else {
			remaining = append(remaining, block)
		}
	}
	
	bc.orphaned = remaining
	
	// Rebuild indices if any blocks were incorporated
	if incorporated > 0 {
		bc.rebuildIndex()
	}
	
	return incorporated
}

// ChainHeight returns the current chain length
func (bc *Blockchain) ChainHeight() int {
	return len(bc.Blocks)
}

// LastBlock returns the tip of the chain
func (bc *Blockchain) LastBlock() *Block {
	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

// RebuildIndex rebuilds all indices (used after loading from disk)
// This rebuilds both the block/transaction index AND the UTXO index
// Critical: Both indices must be rebuilt together to ensure correct chain state
func (bc *Blockchain) RebuildIndex() {
	bc.rebuildIndex()
	bc.rebuildUTXOIndex()
}
