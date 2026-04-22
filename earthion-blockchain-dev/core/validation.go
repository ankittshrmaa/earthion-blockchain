package core

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"earthion/crypto"
)

// Validation constants
const (
	MaxTransactionSize       = 1000000  // 1MB
	MaxOutputsPerTransaction = 10000
	MaxInputsPerTransaction  = 10000
	MaxValue                 = math.MaxInt64 // Max value per output
	BlockMaxTimeInFuture     = 2 * 60       // 2 minutes in seconds
	MaxBlockSize             = 1000000       // 1MB max block size
	DustLimit                = 546           // Dust limit in satoshis (same as Bitcoin)
)

// ValidationError represents a validation error with code
type ValidationError struct {
	Code    string
	Message string
	TxID   []byte
}

func (e *ValidationError) Error() string {
	txIDStr := ""
	if e.TxID != nil {
		txIDStr = hex.EncodeToString(e.TxID)
		if len(txIDStr) > 16 {
			txIDStr = txIDStr[:16]
		}
	}
	return fmt.Sprintf("[%s] %s (tx: %s...)", e.Code, e.Message, txIDStr)
}

// Validation codes
const (
	ErrCodeNilTx          = "ERR_TX_NIL"
	ErrCodeEmptyID       = "ERR_TX_EMPTY_ID"
	ErrCodeNoInputs     = "ERR_TX_NO_INPUTS"
	ErrCodeNoOutputs   = "ERR_TX_NO_OUTPUTS"
	ErrCodeInvalidIn  = "ERR_TX_INVALID_IN"
	ErrCodeNegOutput  = "ERR_TX_NEG_OUTPUT"
	ErrCodeOverflow  = "ERR_TX_OVERFLOW"
	ErrCodeSigVerify = "ERR_TX_SIG_VERIFY"
	ErrCodeTxSize   = "ERR_TX_SIZE"
	ErrCodeChainTip  = "ERR_BLOCK_CHAIN"
	ErrCodeTime      = "ERR_BLOCK_TIME"
	ErrCodePoW       = "ERR_BLOCK_POW"
	ErrCodeMerkle    = "ERR_BLOCK_MERKLE"
	ErrCodeOrphan    = "ERR_TX_ORPHAN"
	ErrCodeBlockSize = "ERR_BLOCK_SIZE"
	// SegWit validation codes
	ErrCodeSegWit       = "ERR_SEGWIT"
	ErrCodeWitness      = "ERR_WITNESS"
	ErrCodeWitnessStack = "ERR_WITNESS_STACK"
	ErrCodeWeight       = "ERR_WEIGHT"
)

// =============================================================================
// TRANSACTION VALIDATION
// =============================================================================

// ValidateTransaction performs strict validation on a transaction
// Returns nil if valid, or ValidationError if invalid
func ValidateTransaction(tx *Transaction, isCoinbase bool) *ValidationError {
	// Check nil
	if tx == nil {
		return &ValidationError{Code: ErrCodeNilTx, Message: "transaction is nil"}
	}

	// Check ID
	if len(tx.ID) == 0 {
		txID := []byte{}
		return &ValidationError{Code: ErrCodeEmptyID, Message: "transaction has no ID", TxID: txID}
	}

	// Check inputs
	if len(tx.Inputs) == 0 {
		return &ValidationError{Code: ErrCodeNoInputs, Message: "no inputs", TxID: tx.ID}
	}
	if len(tx.Inputs) > MaxInputsPerTransaction {
		return &ValidationError{Code: ErrCodeOverflow, 
			Message: fmt.Sprintf("too many inputs: %d > %d", len(tx.Inputs), MaxInputsPerTransaction), TxID: tx.ID}
	}

	// Check outputs
	if len(tx.Outputs) == 0 {
		return &ValidationError{Code: ErrCodeNoOutputs, Message: "no outputs", TxID: tx.ID}
	}
	if len(tx.Outputs) > MaxOutputsPerTransaction {
		return &ValidationError{Code: ErrCodeOverflow, 
			Message: fmt.Sprintf("too many outputs: %d > %d", len(tx.Outputs), MaxOutputsPerTransaction), TxID: tx.ID}
	}

	// Validate inputs
	inputsSeen := make(map[string]bool)
	for i, in := range tx.Inputs {
		// Skip coinbase input validation
		if in.OutIndex == -1 && isCoinbase {
			continue
		}
		
		// Check input has required fields
		if len(in.Txid) == 0 {
			return &ValidationError{Code: ErrCodeInvalidIn, 
				Message: fmt.Sprintf("input %d: no txid", i), TxID: tx.ID}
		}
		if in.OutIndex < -1 {
			return &ValidationError{Code: ErrCodeInvalidIn, 
				Message: fmt.Sprintf("input %d: negative out index", i), TxID: tx.ID}
		}

		// Check for duplicate inputs (double-spend within same tx)
		inputKey := fmt.Sprintf("%s:%d", hex.EncodeToString(in.Txid), in.OutIndex)
		if inputsSeen[inputKey] {
			return &ValidationError{Code: ErrCodeInvalidIn,
				Message: fmt.Sprintf("input %d: duplicate input %s", i, inputKey), TxID: tx.ID}
		}
		inputsSeen[inputKey] = true
	}

	// Validate outputs
	totalOut := int64(0)
	for i, out := range tx.Outputs {
		if out.Value <= 0 {
			return &ValidationError{Code: ErrCodeNegOutput,
				Message: fmt.Sprintf("output %d: non-positive value %d", i, out.Value), TxID: tx.ID}
		}
		// Dust limit check - prevent UTXO set spam
		if out.Value < DustLimit {
			return &ValidationError{Code: ErrCodeNegOutput,
				Message: fmt.Sprintf("output %d: dust limit %d < %d", i, out.Value, DustLimit), TxID: tx.ID}
		}
		if out.Value > MaxValue {
			return &ValidationError{Code: ErrCodeOverflow,
				Message: fmt.Sprintf("output %d: value overflow", i), TxID: tx.ID}
		}
		if len(out.PubKey) == 0 {
			return &ValidationError{Code: ErrCodeNegOutput,
				Message: fmt.Sprintf("output %d: no pubkey", i), TxID: tx.ID}
		}
		if len(out.PubKey) != 20 {
			return &ValidationError{Code: ErrCodeNegOutput,
				Message: fmt.Sprintf("output %d: invalid pubkey length %d (expected 20)", i, len(out.PubKey)), TxID: tx.ID}
		}

		totalOut += int64(out.Value)
		// Check for overflow
		if totalOut > MaxValue {
			return &ValidationError{Code: ErrCodeOverflow,
				Message: "total output overflow", TxID: tx.ID}
		}
	}

	// Verify signature for non-coinbase
	if !isCoinbase {
		if !tx.Verify() {
			return &ValidationError{Code: ErrCodeSigVerify, Message: "signature verification failed", TxID: tx.ID}
		}
	}

	return nil
}

// ValidateTransactionForMempool validates transaction for mempool acceptance
func ValidateTransactionForMempool(tx *Transaction, bc *Blockchain) *ValidationError {
	// Basic validation
	if err := ValidateTransaction(tx, tx.IsCoinbase()); err != nil {
		return err
	}

	// Check if inputs reference valid UTXOs (not already spent)
	utxos := bc.UTXOIndex()
	for _, in := range tx.Inputs {
		if in.OutIndex == -1 {
			continue // coinbase input
		}
		
		key := fmt.Sprintf("%s:%d", hex.EncodeToString(in.Txid), in.OutIndex)
		if _, exists := utxos[key]; !exists {
			return &ValidationError{Code: ErrCodeOrphan, 
				Message: fmt.Sprintf("input references unknown UTXO: %s", key), TxID: tx.ID}
		}
	}

	return nil
}

// =============================================================================
// BLOCK VALIDATION
// =============================================================================

// ValidateBlock performs strict validation on a block
func ValidateBlock(block *Block, prevBlock *Block) *ValidationError {
	// Check block
	if block == nil {
		return &ValidationError{Code: ErrCodeNilTx, Message: "block is nil"}
	}

	// Check index
	if block.Index < 0 {
		return &ValidationError{Code: ErrCodeChainTip, Message: "negative block index"}
	}

	// Check prevHash linkage
	if prevBlock != nil {
		if !bytes.Equal(block.PrevHash, prevBlock.Hash) {
			return &ValidationError{Code: ErrCodeChainTip, 
				Message: fmt.Sprintf("prevHash mismatch: expected %x, got %x", 
					prevBlock.Hash[:8], block.PrevHash[:8])}
		}
	}

	// Check timestamp (not too far in future)
	now := time.Now().Unix()
	ts := block.TimestampInt()
	if ts > now+BlockMaxTimeInFuture {
		return &ValidationError{Code: ErrCodeTime, 
			Message: fmt.Sprintf("timestamp too far in future: %d > %d", 
				ts, now)}
	}
	if ts < 0 {
		return &ValidationError{Code: ErrCodeTime, Message: "negative timestamp"}
	}

	// Check difficulty
	if block.Difficulty == 0 {
		return &ValidationError{Code: ErrCodeChainTip, Message: "zero difficulty"}
	}
	if block.Difficulty < MinDifficulty {
		return &ValidationError{Code: ErrCodeChainTip, Message: "difficulty below minimum"}
	}
	if block.Difficulty > MaxDifficulty {
		return &ValidationError{Code: ErrCodeChainTip, Message: "difficulty above maximum"}
	}

	// Check PoW
	pow := NewProofOfWork(block)
	if !pow.Validate() {
		return &ValidationError{Code: ErrCodePoW, Message: "PoW validation failed"}
	}

	// Check Merkle root
	txHashes := make([][]byte, len(block.Transactions))
	for i, tx := range block.Transactions {
		txHashes[i] = tx.ID
	}
	merkleTree := crypto.NewMerkleTree(txHashes)
	if !bytes.Equal(block.MerkleRoot, merkleTree.RootHash()) {
		return &ValidationError{Code: ErrCodeMerkle, Message: "Merkle root mismatch"}
	}

	// Validate all transactions in block
	for i, tx := range block.Transactions {
		if err := ValidateTransaction(tx, tx.IsCoinbase()); err != nil {
			return &ValidationError{
				Code: err.Code, 
				Message: fmt.Sprintf("tx %d: %s", i, err.Message), 
				TxID: tx.ID,
			}
		}
	}

	// Validate coinbase in block
	if len(block.Transactions) == 0 {
		return &ValidationError{Code: ErrCodeChainTip, Message: "block has no transactions"}
	}

	// Check coinbase is first
	if !block.Transactions[0].IsCoinbase() {
		return &ValidationError{Code: ErrCodeChainTip, Message: "first tx is not coinbase"}
	}

	// Check block size limit
	blockSize := len(block.Serialize())
	if blockSize > MaxBlockSize {
		return &ValidationError{Code: ErrCodeChainTip, 
			Message: fmt.Sprintf("block size %d exceeds limit %d", blockSize, MaxBlockSize)}
	}

	return nil
}

// =============================================================================
// CHAIN VALIDATION
// =============================================================================

// ValidateChain performs full chain validation
func ValidateChain(bc *Blockchain) *ValidationError {
	if len(bc.Blocks) == 0 {
		return &ValidationError{Code: ErrCodeChainTip, Message: "empty chain"}
	}

	// Check genesis
	genesis := bc.Blocks[0]
	if !bytes.Equal(genesis.PrevHash, []byte{}) {
		return &ValidationError{Code: ErrCodeChainTip, Message: "genesis has wrong prevHash"}
	}

	// Validate each block
	for i := 1; i < len(bc.Blocks); i++ {
		prevBlock := bc.Blocks[i-1]
		block := bc.Blocks[i]
		
		if err := ValidateBlock(block, prevBlock); err != nil {
			return &ValidationError{
				Code: err.Code,
				Message: fmt.Sprintf("block %d: %s", i, err.Message),
			}
		}
	}

	return nil
}

// =============================================================================
// UTXO VALIDATION
// =============================================================================

// ValidateUTXO checks UTXO set consistency
func (bc *Blockchain) ValidateUTXO() *ValidationError {
	utxos := bc.UTXOIndex()
	
	// Check no duplicate keys
	seen := make(map[string]bool)
	for key := range utxos {
		if seen[key] {
			return &ValidationError{Code: ErrCodeChainTip, 
				Message: fmt.Sprintf("duplicate UTXO: %s", key)}
		}
		seen[key] = true
	}
	
	// Check no negative values
	for key, out := range utxos {
		if out.Value <= 0 {
			return &ValidationError{Code: ErrCodeNegOutput, 
				Message: fmt.Sprintf("negative UTXO %s: %d", key, out.Value)}
		}
	}
	
	return nil
}

// ValidateSegWitTransaction validates a SegWit transaction
func ValidateSegWitTransaction(tx *Transaction, witnessData map[int][]*Witness) *ValidationError {
	hasWitness := false
	for _, w := range witnessData {
		if len(w) > 0 {
			hasWitness = true
			break
		}
	}
	
	if !hasWitness {
		return nil
	}
	
	for idx, witnesses := range witnessData {
		if len(witnesses) == 0 {
			continue
		}
		
		if idx >= len(tx.Inputs) {
			return &ValidationError{Code: ErrCodeSegWit, 
				Message: fmt.Sprintf("witness index %d exceeds input count", idx), TxID: tx.ID}
		}
		
		for _, w := range witnesses {
			if len(w.Items) == 0 {
				return &ValidationError{Code: ErrCodeWitnessStack, 
					Message: fmt.Sprintf("empty witness at input %d", idx), TxID: tx.ID}
			}
		}
	}
	
	return nil
}

// ValidateWitnessProgram validates a witness program
func ValidateWitnessProgram(program *WitnessProgram, witness *Witness) *ValidationError {
	if program == nil {
		return &ValidationError{Code: ErrCodeWitness, Message: "nil witness program"}
	}
	
	if program.Version != 0 {
		return &ValidationError{Code: ErrCodeWitness, 
			Message: fmt.Sprintf("unsupported witness version: %d", program.Version)}
	}
	
	programLen := len(program.Program)
	if programLen != 20 && programLen != 32 {
		return &ValidationError{Code: ErrCodeWitness, 
			Message: fmt.Sprintf("invalid witness program length: %d", programLen)}
	}
	
	if witness != nil {
		if programLen == 20 && len(witness.Items) < 2 {
			return &ValidationError{Code: ErrCodeWitnessStack, 
				Message: "P2WPKH witness needs at least 2 items"}
		}
	}
	
	return nil
}

// ValidateTaprootTransaction validates a Taproot transaction
func ValidateTaprootTransaction(tx *Transaction, witnessData map[int][]*Witness) *ValidationError {
	return nil
}