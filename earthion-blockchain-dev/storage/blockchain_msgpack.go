package storage

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/vmihailenco/msgpack/v5"

	"earthion/core"
)

// BlockMsgPack is a msgpack-friendly version of Block
type BlockMsgPack struct {
	Index        int                         `msgpack:"index"`
	Timestamp    int64                       `msgpack:"timestamp"`
	PrevHash     string                      `msgpack:"prevHash"`
	MerkleRoot   string                      `msgpack:"merkleRoot"`
	Hash         string                      `msgpack:"hash"`
	Nonce        int                         `msgpack:"nonce"`
	Difficulty   uint32                      `msgpack:"difficulty"`
	Transactions []core.TransactionMsgPack  `msgpack:"transactions"`
}

// TransactionMsgPack is a msgpack-friendly version of Transaction
type TransactionMsgPack struct {
	ID       string               `msgpack:"id"`
	Inputs  []core.TXInputMsgPack `msgpack:"inputs"`
	Outputs []core.TXOutputMsgPack `msgpack:"outputs"`
}

// SaveBlockchainMsgPack persists blockchain to file using MessagePack
// MessagePack is more efficient than JSON: faster parsing, smaller size, binary format
func SaveBlockchainMsgPack(bc *core.Blockchain, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Convert to msgpack-friendly format
	blocksMsgPack := make([]BlockMsgPack, len(bc.Blocks))
	for i, block := range bc.Blocks {
		blocksMsgPack[i] = blockToMsgPack(block)
	}

	// Encode using msgpack - much more efficient than JSON
	encoder := msgpack.NewEncoder(file)
	return encoder.Encode(blocksMsgPack)
}

// LoadBlockchainMsgPack reads blockchain from MessagePack file
func LoadBlockchainMsgPack(filename string) (*core.Blockchain, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode using msgpack
	var blocksMsgPack []BlockMsgPack
	decoder := msgpack.NewDecoder(file)
	err = decoder.Decode(&blocksMsgPack)
	if err != nil {
		return nil, fmt.Errorf("msgpack decode error: %w", err)
	}

	// Convert from msgpack format
	blocks := make([]*core.Block, len(blocksMsgPack))
	for i, bm := range blocksMsgPack {
		b := blockFromMsgPack(bm)
		if b == nil || len(b.Hash) == 0 {
			return nil, fmt.Errorf("invalid block data at index %d", i)
		}
		blocks[i] = b
	}

	// Create blockchain using constructor
	bc := core.NewBlockchain()
	bc.SetFilename(filename)
	bc.Blocks = blocks
	bc.RebuildIndex()

	return bc, nil
}

func blockToMsgPack(b *core.Block) BlockMsgPack {
	bj := BlockMsgPack{
		Index:      b.Index,
		Timestamp:  b.TimestampInt(),
		PrevHash:   hex.EncodeToString(b.PrevHash),
		MerkleRoot: hex.EncodeToString(b.MerkleRoot),
		Hash:       hex.EncodeToString(b.Hash),
		Nonce:      b.Nonce,
		Difficulty: b.Difficulty,
	}

	bj.Transactions = make([]core.TransactionMsgPack, len(b.Transactions))
	for i, tx := range b.Transactions {
		bj.Transactions[i] = tx.ToMsgPack()
	}

	return bj
}

func blockFromMsgPack(bj BlockMsgPack) *core.Block {
	block := &core.Block{}

	block.Index = bj.Index
	block.BlockHeader.Timestamp = bj.Timestamp
	block.Nonce = bj.Nonce
	block.BlockHeader.Difficulty = bj.Difficulty

	var err error
	block.PrevHash, err = hex.DecodeString(bj.PrevHash)
	if err != nil {
		return nil
	}
	block.MerkleRoot, err = hex.DecodeString(bj.MerkleRoot)
	if err != nil {
		return nil
	}
	block.Hash, err = hex.DecodeString(bj.Hash)
	if err != nil {
		return nil
	}

	block.Transactions = make([]*core.Transaction, len(bj.Transactions))
	for i, tm := range bj.Transactions {
		block.Transactions[i] = core.TransactionFromMsgPack(tm)
		if block.Transactions[i] == nil {
			return nil
		}
	}

	return block
}

// AutoDetectLoad attempts to load blockchain, trying both MessagePack and JSON formats
// This provides backward compatibility while using the more efficient MessagePack
func AutoDetectLoad(filename string) (*core.Blockchain, error) {
	// Try MessagePack first (newer, more efficient)
	bc, err := LoadBlockchainMsgPack(filename)
	if err == nil {
		return bc, nil
	}

	// Fall back to JSON (for backward compatibility with existing blockchain.dat)
	return LoadBlockchain(filename)
}

// AutoDetectSave saves using MessagePack format (more efficient)
func AutoDetectSave(bc *core.Blockchain, filename string) error {
	return SaveBlockchainMsgPack(bc, filename)
}