package storage

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"earthion/core"
)

const ChainFile = "blockchain.dat"

// BlockJSON is a JSON-friendly version of Block
type BlockJSON struct {
	Index        int                    `json:"index"`
	Timestamp   int64                  `json:"timestamp"`
	PrevHash    string                `json:"prevHash"`
	MerkleRoot  string                `json:"merkleRoot"`
	Hash        string                `json:"hash"`
	Nonce       int                   `json:"nonce"`
	Difficulty uint32                `json:"difficulty"`
	Transactions []core.TransactionJSON `json:"transactions"`
}

// SaveBlockchain persists blockchain to file
func SaveBlockchain(bc *core.Blockchain, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Convert to JSON-friendly format
	blocksJSON := make([]BlockJSON, len(bc.Blocks))
	for i, block := range bc.Blocks {
		blocksJSON[i] = blockToJSON(block)
	}

	encoder := json.NewEncoder(file)
	return encoder.Encode(blocksJSON)
}

// LoadBlockchain reads blockchain from file
func LoadBlockchain(filename string) (*core.Blockchain, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Load blocks
	var blocksJSON []BlockJSON
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&blocksJSON)
	if err != nil {
		return nil, err
	}

	// Convert from JSON format - preserve exact values
	blocks := make([]*core.Block, len(blocksJSON))
	for i, bj := range blocksJSON {
		b := blockFromJSON(bj)
		// Validate block has required fields
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

func blockToJSON(b *core.Block) BlockJSON {
	bj := BlockJSON{
		Index:        b.Index,
		Timestamp:   b.TimestampInt(),
		PrevHash:    hex.EncodeToString(b.PrevHash),
		MerkleRoot:  hex.EncodeToString(b.MerkleRoot),
		Hash:        hex.EncodeToString(b.Hash),
		Nonce:       b.Nonce,
		Difficulty: b.Difficulty,
	}

	bj.Transactions = make([]core.TransactionJSON, len(b.Transactions))
	for i, tx := range b.Transactions {
		bj.Transactions[i] = tx.ToJSON()
	}

	return bj
}

func blockFromJSON(bj BlockJSON) *core.Block {
	block := &core.Block{}

	block.Index = bj.Index
	block.BlockHeader.Timestamp = bj.Timestamp
	block.Nonce = bj.Nonce
	block.BlockHeader.Difficulty = bj.Difficulty

	// Validate and decode hashes with proper error handling
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

	// Convert transactions
	block.Transactions = make([]*core.Transaction, len(bj.Transactions))
	for i, tj := range bj.Transactions {
		block.Transactions[i] = core.TransactionFromJSON(tj)
		if block.Transactions[i] == nil {
			return nil
		}
	}

	return block
}