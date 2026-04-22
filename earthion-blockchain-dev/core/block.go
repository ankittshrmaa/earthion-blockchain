package core

import (
	"bytes"
	"encoding/gob"
	"time"

	"earthion/crypto"
)

// BlockHeader represents just the block header (for sync)
type BlockHeader struct {
	Index       int
	Timestamp   int64
	PrevHash    []byte
	MerkleRoot  []byte
	Hash       []byte
	Nonce      int
	Difficulty uint32
}

// Block represents a full block
type Block struct {
	BlockHeader
	Transactions []*Transaction
}

// GetHeader returns the block header
func (b *Block) GetHeader() BlockHeader {
	return b.BlockHeader
}

// GetBestHeight returns the block height
func (h BlockHeader) GetBestHeight() int {
	return h.Index
}

// GetBestHash returns the block hash
func (h BlockHeader) GetBestHash() []byte {
	return h.Hash
}

// =============================================================================
// Serialization
// =============================================================================

// Serialize serializes the block (full)
func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)
	_ = encoder.Encode(b)
	return res.Bytes()
}

// DeserializeBlock deserializes a block from bytes
func DeserializeBlock(data []byte) (*Block, error) {
	block := &Block{}
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(block)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// Timestamp returns the block timestamp as time.Time
func (b *Block) Timestamp() time.Time {
	return time.Unix(b.BlockHeader.Timestamp, 0)
}

// TimestampInt returns the block timestamp as int64
func (b *Block) TimestampInt() int64 {
	return b.BlockHeader.Timestamp
}

// SerializeHeader serializes just the header
func (h BlockHeader) Serialize() []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	_ = encoder.Encode(h)
	return buf.Bytes()
}

// DeserializeBlockHeader deserializes a block header
func DeserializeBlockHeader(data []byte) (BlockHeader, error) {
	h := BlockHeader{}
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&h)
	if err != nil {
		return BlockHeader{}, err
	}
	return h, nil
}

// =============================================================================
// Block Creation
// =============================================================================

// Create new block with transactions
// prevBlocks is needed to calculate dynamic difficulty
func NewBlock(txs []*Transaction, prevHash []byte, index int, prevBlocks []*Block) *Block {
	// Calculate difficulty based on previous blocks
	var difficulty uint32 = InitialDifficulty
	if prevBlocks != nil && len(prevBlocks) > 0 {
		difficulty = CurrentDifficulty(prevBlocks)
	}

	// Enforce difficulty bounds (safety check)
	if difficulty < MinDifficulty {
		difficulty = MinDifficulty
	}
	if difficulty > MaxDifficulty {
		difficulty = MaxDifficulty
	}

	// Build transaction hashes
	txHashes := make([][]byte, len(txs))
	for i, tx := range txs {
		txHashes[i] = tx.ID
	}

	// Calculate Merkle root
	var merkleRoot []byte
	if len(txHashes) > 0 {
		tree := crypto.NewMerkleTree(txHashes)
		merkleRoot = tree.RootHash()
	} else {
		// Empty block - use double hash of empty bytes
		merkleRoot = crypto.DoubleHash([]byte{})
	}

	block := &Block{
		BlockHeader: BlockHeader{
			Index:       index,
			Timestamp:   time.Now().Unix(),
			PrevHash:   prevHash,
			MerkleRoot: merkleRoot,
			Difficulty: difficulty,
		},
		Transactions: txs,
	}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.BlockHeader.Nonce = nonce
	block.BlockHeader.Hash = hash

	return block
}

// GenesisBlock creates the first block in chain
func GenesisBlock() *Block {
	return NewBlock([]*Transaction{}, []byte{}, 0, nil)
}