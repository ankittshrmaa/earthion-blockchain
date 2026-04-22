package crypto

import (
	"bytes"
)

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Hash  []byte
}

// MerkleTree builds and manages a Merkle tree
type MerkleTree struct {
	Root *MerkleNode
}

// NewMerkleTree creates a Merkle tree from transaction hashes
func NewMerkleTree(txHashes [][]byte) *MerkleTree {
	if len(txHashes) == 0 {
		return nil
	}

	// If only one tx, create root with its hash (duplicate for left/right)
	if len(txHashes) == 1 {
		txHashes = append(txHashes, txHashes[0])
	}

	var nodes []*MerkleNode

	// Create leaf nodes (transaction hashes)
	for _, txHash := range txHashes {
		nodes = append(nodes, &MerkleNode{Hash: txHash})
	}

	// Build tree bottom-up
	for len(nodes) > 1 {
		var newLevel []*MerkleNode

		for i := 0; i < len(nodes); i += 2 {
			left := nodes[i]

			var right *MerkleNode
			if i+1 < len(nodes) {
				right = nodes[i+1]
			} else {
				// Duplicate last node if odd number
				right = left
			}

			// Hash(left || right)
			combined := append(left.Hash, right.Hash...)
			hash := DoubleHash(combined)

			newLevel = append(newLevel, &MerkleNode{
				Left:  left,
				Right: right,
				Hash:  hash,
			})
		}

		nodes = newLevel
	}

	return &MerkleTree{Root: nodes[0]}
}

// RootHash returns the Merkle root
func (mt *MerkleTree) RootHash() []byte {
	if mt == nil || mt.Root == nil {
		return nil
	}
	return mt.Root.Hash
}

// MerkleProof returns the Merkle proof for a transaction at index
func (mt *MerkleTree) MerkleProof(index int) [][]byte {
	if mt == nil || mt.Root == nil {
		return nil
	}

	var proof [][]byte
	var currentIndex int = index
	node := mt.Root

	for node.Left != nil || node.Right != nil {
		var siblingHash []byte

		if node.Left != nil && currentIndex%2 == 0 {
			// We're on the left, sibling is right
			siblingHash = node.Right.Hash
		} else if node.Right != nil {
			// We're on the right, sibling is left
			siblingHash = node.Left.Hash
		} else {
			break
		}

		proof = append(proof, siblingHash)
		currentIndex /= 2
		node = node.Left
		if node == nil {
			node = node.Right
		}
	}

	return proof
}

// VerifyMerkleProof verifies a Merkle proof
func VerifyMerkleProof(txHash, merkleRoot []byte, proof [][]byte, index int) bool {
	currentHash := txHash

	for _, siblingHash := range proof {
		var combined []byte
		if index%2 == 0 {
			combined = append(currentHash, siblingHash...)
		} else {
			combined = append(siblingHash, currentHash...)
		}
		currentHash = DoubleHash(combined)
		index /= 2
	}

	return bytes.Equal(currentHash, merkleRoot)
}

// BuildMerkleRoot computes Merkle root from transaction hashes (convenience function)
func BuildMerkleRoot(txHashes []byte) []byte {
	return DoubleHash(txHashes)
}