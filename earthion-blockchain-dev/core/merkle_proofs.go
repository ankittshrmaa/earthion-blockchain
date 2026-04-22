package core

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"earthion/crypto"
)

// =============================================================================
// Compact Merkle Proofs (BIP-37) - Light Client Support
// =============================================================================

// FilterType represents the type of bloom filter
type FilterType int

const (
	FilterTypeNone FilterType = iota
	FilterTypeTXIDMatch
	FilterTypePubKeyEquals
)

// =============================================================================
// Bloom Filter (BIP-37)
// =============================================================================

// BloomFilter represents a Bloom filter for SPV clients
type BloomFilter struct {
	data        []byte           // Filter data
	tweak       uint32           // Random tweak
	flags       BloomUpdateFlags // Update flag
	size        uint32           // Size in bytes
	hashFuncs   uint32           // Number of hash functions
}

// BloomUpdateFlags represents bloom filter update flags
type BloomUpdateFlags byte

const (
	BloomUpdateNone BloomUpdateFlags = iota // Don't add outputs
	BloomUpdateAll                           // Add all outputs to filter
	BloomUpdateMask                          // Mask for update flags
)

// NewBloomFilter creates a new Bloom filter
func NewBloomFilter(size uint32, hashFuncs uint32, tweak uint32) *BloomFilter {
	data := make([]byte, size)
	return &BloomFilter{
		data:      data,
		tweak:     tweak,
		flags:     BloomUpdateAll,
		size:      size,
		hashFuncs: hashFuncs,
	}
}

// Insert inserts data into the filter
func (bf *BloomFilter) Insert(data []byte) {
	for i := uint32(0); i < bf.hashFuncs; i++ {
		idx := bf.hash(i, data)
		bf.data[idx/8] |= (1 << (idx % 8))
	}
}

// Contains checks if data might be in the filter
func (bf *BloomFilter) Contains(data []byte) bool {
	for i := uint32(0); i < bf.hashFuncs; i++ {
		idx := bf.hash(i, data)
		if (bf.data[idx/8] & (1 << (idx % 8))) == 0 {
			return false
		}
	}
	return true
}

// hash calculates the i'th hash function
func (bf *BloomFilter) hash(i uint32, data []byte) uint32 {
	h := sipHash24(data, uint64(bf.tweak+uint32(i)))
	return uint32(h % uint64(bf.size*8))
}

// sipHash24 is a fast non-cryptographic hash (simplified)
func sipHash24(data []byte, k uint64) uint64 {
	// Simplified SipHash-2-4 implementation
	// For production, use proper SipHash implementation
	
	v0 := k ^ 0x736f6d6570736575
	v1 := k ^ 0x646f72616e646f6d
	v2 := k ^ 0x6c7967656e657261
	v3 := k ^ 0x74656462797465
	
	// Process 8-byte chunks
	for len(data) >= 8 {
		m := binary.LittleEndian.Uint64(data)
		v3 ^= m
		v0 ^= m
		v0 += v3
		v3 = rotl64(v3, 32)
		v1 ^= v2
		v2 = rotl64(v2, 17)
		v0 ^= v1
		v1 += v2
		v2 = rotl64(v2, 13)
		data = data[8:]
	}
	
	// Finalization
	v3 ^= 0xff
	for i := 0; i < 4; i++ {
		v0 ^= v1
		v2 ^= v3
		v0 += v1
		v2 += v3
		v3 = rotl64(v3, 16)
		v1 = rotl64(v1, 13)
		v0 = rotl64(v0, 32)
	}
	
	return v0 ^ v1 ^ v2 ^ v3
}

func rotl64(x uint64, r uint8) uint64 {
	return (x << r) | (x >> (64 - r))
}

// =============================================================================
// Compact Merkle Tree (BIP-37)
// =============================================================================

// CompactMerkleTree represents a compact Merkle tree for light clients
type CompactMerkleTree struct {
	// Tree structure
	Hashes     [][]byte      // All hashes in tree
	Branches   [][][]byte   // Authentication path for each leaf
	TopBits    uint8        // Number of top nodes
	NumTX      uint32       // Number of transactions
	
	// Partial merkle branch
	PartialBranches [][]byte // Selected merkle branches
	PartialIndices []uint32  // Index of selected hashes
}

// NewCompactMerkleTree creates a new compact merkle tree
func NewCompactMerkleTree() *CompactMerkleTree {
	return &CompactMerkleTree{
		Hashes:          make([][]byte, 0),
		Branches:       make([][][]byte, 0),
		PartialBranches: make([][]byte, 0),
		PartialIndices:  make([]uint32, 0),
	}
}

// Build builds the compact tree from transactions
func (cmt *CompactMerkleTree) Build(transactions []*Transaction) {
	cmt.NumTX = uint32(len(transactions))
	
	// Calculate tree height
	height := 0
	for (1 << height) < len(transactions) {
		height++
	}
	
	// Start with transaction hashes as leaves
	cmt.Hashes = make([][]byte, len(transactions))
	for i, tx := range transactions {
		cmt.Hashes[i] = tx.ID
	}
	
	// Build tree bottom-up
	level := cmt.Hashes
	for len(level) > 1 {
		var nextLevel [][]byte
		
		for i := 0; i < len(level); i += 2 {
			var left, right []byte
			left = level[i]
			if i+1 < len(level) {
				right = level[i+1]
			} else {
				right = left
			}
			
			combined := make([]byte, 64)
			copy(combined[:32], left)
			copy(combined[32:], right)
			parent := crypto.DoubleHash(combined)
			nextLevel = append(nextLevel, parent)
		}
		
		level = nextLevel
		cmt.Hashes = append(cmt.Hashes, level...)
	}
	
	// Store branches for each transaction
	cmt.Branches = make([][][]byte, len(transactions))
	for i := range transactions {
		cmt.Branches[i] = cmt.buildBranch(uint32(i), uint32(height))
	}
}

// buildBranch builds the authentication path for a leaf
func (cmt *CompactMerkleTree) buildBranch(index uint32, height uint32) [][]byte {
	branch := make([][]byte, 0)
	
	levelSize := cmt.NumTX
	currentIdx := index
	
	for level := uint32(0); level < height; level++ {
		var siblingIdx uint32
		if currentIdx%2 == 0 {
			siblingIdx = currentIdx + 1
		} else {
			siblingIdx = currentIdx - 1
		}
		
		if siblingIdx < levelSize {
			siblingHash := cmt.Hashes[int(levelSize-1)+int(siblingIdx)]
			if siblingHash != nil {
				branch = append(branch, siblingHash)
			}
		}
		
		currentIdx = currentIdx / 2
		levelSize = (levelSize + 1) / 2
	}
	
	return branch
}

// GetPartialMerkleBranch returns partial branches for given indices
func (cmt *CompactMerkleTree) GetPartialMerkleBranch(indices []uint32) [][]byte {
	branches := make([][]byte, 0)
	for _, idx := range indices {
		if int(idx) < len(cmt.Branches) {
			branches = append(branches, cmt.Branches[idx]...)
		}
	}
	return branches
}

// SerializePartialMerkle serializes partial merkle branch (BIP-37)
// Format: 1-byte flags + merkle width + num hashes + hash array + indices
func (cmt *CompactMerkleTree) SerializePartialMerkle(indices []uint32) []byte {
	var buf bytes.Buffer
	
	// Flags (partial tree)
	buf.WriteByte(0x01) // Partial merkle
	
	// Number of transactions
	var numTX [4]byte
	binary.LittleEndian.PutUint32(numTX[:], cmt.NumTX)
	buf.Write(numTX[:])
	
	// Number of hashes in partial tree
	hashes := cmt.GetPartialMerkleBranch(indices)
	var numHashes [4]byte
	binary.LittleEndian.PutUint32(numHashes[:], uint32(len(hashes)))
	buf.Write(numHashes[:])
	
	// Hashes
	for _, h := range hashes {
		buf.Write(h)
	}
	
	// Flags for each transaction (1 if in tree, 0 if not)
	// Simplified: all selected
	flagBytes := make([]byte, (len(indices)+7)/8)
	for i := range indices {
		flagBytes[i/8] |= (1 << (i % 8))
	}
	buf.Write(flagBytes)
	
	return buf.Bytes()
}

// VerifyMerkleProof verifies a merkle proof
func (cmt *CompactMerkleTree) VerifyMerkleProof(txHash []byte, proof []byte, merkleRoot []byte) bool {
	// Parse proof and verify
	// Simplified: rebuild path to root
	
	if len(proof) < 32 {
		return false
	}
	
	// For now, rebuild from full tree
	// Check if txHash is in our tree
	for i, h := range cmt.Hashes[:cmt.NumTX] {
		if bytes.Equal(h, txHash) {
			// Build path to root
			currentHash := txHash
			idx := uint32(i)
			
			// Get branch
			if int(idx) >= len(cmt.Branches) {
				return false
			}
			
			branch := cmt.Branches[idx]
			for _, sibling := range branch {
				combined := make([]byte, 64)
				if idx%2 == 0 {
					copy(combined[:32], currentHash)
					copy(combined[32:], sibling)
				} else {
					copy(combined[:32], sibling)
					copy(combined[32:], currentHash)
				}
				currentHash = crypto.DoubleHash(combined)
				idx = idx / 2
			}
			
			return bytes.Equal(currentHash, merkleRoot)
		}
	}
	
	_ = proof // proof used for future verification
	return false
}

// =============================================================================
// Partial Merkle Block (BIP-37)
// =============================================================================

// PartialMerkleBlock represents a partial merkle block for SPV
type PartialMerkleBlock struct {
	Header       BlockHeader    // Block header
	Transactions uint32         // Number of transactions
	Hashes       [][]byte       // Selected hashes
	Flags        []byte         // Match flags
}

// NewPartialMerkleBlock creates a new partial merkle block
func NewPartialMerkleBlock() *PartialMerkleBlock {
	return &PartialMerkleBlock{
		Hashes: make([][]byte, 0),
		Flags:  make([]byte, 0),
	}
}

// ParseMerkleBlock parses a merkle block from header + merkle data
func ParseMerkleBlock(blockData []byte) (*PartialMerkleBlock, error) {
	if len(blockData) < 80 {
		return nil, fmt.Errorf("data too short for header")
	}
	
	pb := &PartialMerkleBlock{}
	
	// Parse header
	header, err := DeserializeBlockHeader(blockData[:80])
	if err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}
	pb.Header = header
	
	offset := 80
	
	// Transactions count
	if offset+4 > len(blockData) {
		return nil, fmt.Errorf("data too short for tx count")
	}
	pb.Transactions = binary.LittleEndian.Uint32(blockData[offset : offset+4])
	offset += 4
	
	// Number of hashes
	if offset+4 > len(blockData) {
		return nil, fmt.Errorf("data too short for hash count")
	}
	numHashes := binary.LittleEndian.Uint32(blockData[offset : offset+4])
	offset += 4
	
	// Read hashes
	pb.Hashes = make([][]byte, numHashes)
	for i := uint32(0); i < numHashes; i++ {
		if offset+32 > len(blockData) {
			return nil, fmt.Errorf("data too short for hash")
		}
		pb.Hashes[i] = blockData[offset : offset+32]
		offset += 32
	}
	
	// Flags
	if offset < len(blockData) {
		pb.Flags = blockData[offset:]
	}
	
	return pb, nil
}

// FilterMatches checks if transaction matches the filter
func (pm *PartialMerkleBlock) FilterMatches(filter *BloomFilter, txHashes [][]byte) []int {
	matches := make([]int, 0)
	
	for i, txHash := range txHashes {
		if filter.Contains(txHash) {
			matches = append(matches, i)
		}
	}
	
	return matches
}

// =============================================================================
// SPV Client (Simplified)
// =============================================================================

// SPVClient represents a Simplified Payment Verification client
type SPVClient struct {
	Chain          interface{ GetBestHash() []byte; GetBestHeight() int }
	Filter         *BloomFilter
	PartialBlocks  map[string]*PartialMerkleBlock
}

// NewSPVClient creates a new SPV client
func NewSPVClient(chain interface{ GetBestHash() []byte; GetBestHeight() int }) *SPVClient {
	return &SPVClient{
		Chain:         chain,
		Filter:        NewBloomFilter(30000, 10, 0), // Default params
		PartialBlocks: make(map[string]*PartialMerkleBlock),
	}
}

// SetBloomFilter sets the bloom filter
func (spv *SPVClient) SetBloomFilter(filter *BloomFilter) {
	spv.Filter = filter
}

// AddToFilter adds data to the bloom filter
func (spv *SPVClient) AddToFilter(data []byte) {
	spv.Filter.Insert(data)
}

// ProcessBlock processes a partial merkle block
func (spv *SPVClient) ProcessBlock(p merkleBlock, header BlockHeader) error {
	hash := header.Hash
	if hash == nil {
		hash = crypto.DoubleHash(header.Serialize())
	}
	
	spv.PartialBlocks[string(hash)] = &PartialMerkleBlock{
		Header:       header,
		Transactions: p.numTX,
		Hashes:       p.hashes,
		Flags:        p.flags,
	}
	
	// Verify header (check PoW and chain work)
	// In production, would verify difficulty
	
	return nil
}

type merkleBlock struct {
	numTX  uint32
	hashes [][]byte
	flags  []byte
}

// GetTransaction gets a transaction from partial block
func (spv *SPVClient) GetTransaction(blockHash []byte, txHash []byte) *Transaction {
	pb, ok := spv.PartialBlocks[string(blockHash)]
	if !ok {
		return nil
	}
	
	// Check if txHash matches one of our partial hashes
	// In production, would reconstruct full merkle tree
	// For now, just check if in hashes list
	for _, h := range pb.Hashes {
		if bytes.Equal(h, txHash) {
			// Would fetch full tx from network
			return nil
		}
	}
	
	return nil
}

// =============================================================================
// Merkle Trie (Alternative - for future use)
// =============================================================================

// MerkleTrie represents a Merkle Patricia Trie for state
type MerkleTrie struct {
	root   []byte
	nodes  map[string]*TrieNode
}

// TrieNode represents a node in the trie
type TrieNode struct {
	Children map[string][]byte  // Partial path -> hash
	Value    []byte
}

// NewMerkleTrie creates a new Merkle trie
func NewMerkleTrie() *MerkleTrie {
	return &MerkleTrie{
		nodes: make(map[string]*TrieNode),
	}
}

// Insert inserts a key-value pair
func (mt *MerkleTrie) Insert(key, value []byte) {
	path := encodePath(key)
	// Simplified: would implement full trie logic
	_ = path
	_ = value
}

// Get retrieves a value
func (mt *MerkleTrie) Get(key []byte) []byte {
	path := encodePath(key)
	// Simplified
	_ = path
	return nil
}

// Root returns the merkle root
func (mt *MerkleTrie) Root() []byte {
	return mt.root
}

func encodePath(key []byte) []byte {
	// Path encoding for Merkle trie
	return crypto.Hash(key)
}