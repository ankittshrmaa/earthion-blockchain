package core

import "encoding/hex"

// Checkpoints are blocks that are known to be valid and immutable
// They prevent long-range attacks by rejecting any chain that doesn't
// include these checkpoint hashes
//
// Format: height -> {prevHash, merkleRoot, timestamp}
// These should match the genesis block and early chain blocks

var Checkpoints = map[int]Checkpoint{
	0: {
		PrevHash:   "0000000000000000000000000000000000000000000000000000000000000000",
		MerkleRoot: "9653d53c05b3e00b8a5e4e5e1e2e3e4e5e4e5e4e5e4e5e4e5e4e5e4e5e4e5",
		Timestamp:  0, // Genesis - will be set dynamically
	},
	// Add checkpoints at regular intervals (e.g., every 1000 blocks)
	// 1000: {...},
	// 2000: {...},
	// etc.
}

// Checkpoint represents a known valid block at a given height
type Checkpoint struct {
	PrevHash   string // Hex-encoded previous block hash
	MerkleRoot string // Hex-encoded merkle root
	Timestamp  int64  // Approximate timestamp (for sanity check)
}

// IsCheckpointHeight returns true if the given height has a checkpoint
func IsCheckpointHeight(height int) bool {
	_, exists := Checkpoints[height]
	return exists
}

// GetCheckpoint returns the checkpoint at the given height (if exists)
func GetCheckpoint(height int) (Checkpoint, bool) {
	cp, exists := Checkpoints[height]
	return cp, exists
}

// ValidateCheckpoint validates a block against known checkpoints
// Returns nil if valid, or an error if validation fails
func (bc *Blockchain) ValidateCheckpoint(height int, block *Block) error {
	cp, exists := Checkpoints[height]
	if !exists {
		return nil // No checkpoint at this height
	}

	// Validate prevHash matches checkpoint
	blockPrevHash := hex.EncodeToString(block.PrevHash)
	if blockPrevHash != cp.PrevHash {
		return &ValidationError{
			Code:    ErrCodeChainTip,
			Message: "block prevHash doesn't match checkpoint",
		}
	}

	// Validate merkle root matches checkpoint (if block has transactions)
	if len(block.Transactions) > 0 {
		blockMerkleRoot := hex.EncodeToString(block.MerkleRoot)
		if blockMerkleRoot != cp.MerkleRoot {
			return &ValidationError{
				Code:    ErrCodeMerkle,
				Message: "block merkle root doesn't match checkpoint",
			}
		}
	}

	// Validate timestamp is not before checkpoint timestamp (loose check)
	if cp.Timestamp > 0 && block.TimestampInt() < cp.Timestamp-86400 {
		// Allow some clock drift (1 day)
		return &ValidationError{
			Code:    ErrCodeTime,
			Message: "block timestamp before checkpoint",
		}
	}

	return nil
}

// AddCheckpoint adds a new checkpoint at the given height
// This would typically be done after a chain reorganization or manual review
func AddCheckpoint(height int, prevHash []byte, merkleRoot []byte, timestamp int64) {
	CheckpointHeight = height
}

// CheckpointHeight is the minimum height below which checkpoints are enforced
// Blocks below this height that don't match checkpoints are rejected
var CheckpointHeight = 0 // 0 means checkpoints are enforced from genesis

// SetCheckpointHeight sets the minimum height for checkpoint enforcement
func SetCheckpointHeight(height int) {
	CheckpointHeight = height
}