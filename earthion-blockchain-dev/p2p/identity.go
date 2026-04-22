package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"

	"earthion/crypto"
)

// ProtocolVersion is the current P2P protocol version
const ProtocolVersion = 1

// NodeID is a unique identifier for a node in the network
type NodeID [32]byte

// String returns the hex representation of the NodeID
func (n NodeID) String() string {
	return hex.EncodeToString(n[:])
}

// LocalNode holds this node's identity and keys
type LocalNode struct {
	ID        NodeID
	PrivKey   *btcec.PrivateKey
	PubKey    *btcec.PublicKey
	StartTime time.Time
}

// NewLocalNode generates a new node identity with a fresh keypair
func NewLocalNode() (*LocalNode, error) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	_, pubKey := btcec.PrivKeyFromBytes(privKey.Serialize())

	// Generate node ID from public key hash
	nodeID := generateNodeID(pubKey.SerializeCompressed())

	return &LocalNode{
		ID:        nodeID,
		PrivKey:   privKey,
		PubKey:    pubKey,
		StartTime: time.Now(),
	}, nil
}

// generateNodeID creates a deterministic node ID from public key
func generateNodeID(pubKey []byte) NodeID {
	hash := crypto.DoubleHash(pubKey)
	var id NodeID
	copy(id[:], hash[:32])
	return id
}

// Sign signs data with the node's private key (returns signature bytes)
func (n *LocalNode) Sign(data []byte) []byte {
	sig := ecdsa.Sign(n.PrivKey, data)
	return sig.Serialize()
}

// Verify checks a signature against the node's public key
func (n *LocalNode) Verify(data []byte, sigBytes []byte) bool {
	sig, err := ecdsa.ParseDERSignature(sigBytes)
	if err != nil {
		return false
	}
	return sig.Verify(data, n.PubKey)
}

// GetAddr returns the node's address for networking (placeholder)
func (n *LocalNode) GetAddr() string {
	// In production, this would return the actual network address
	return n.ID.String()[:16]
}

// =============================================================================
// Key Exchange (ECDH) for encrypted communication
// =============================================================================

// PerformECDH performs ECDH key exchange with a peer's public key
// Returns a shared secret that can be used for symmetric encryption
// Note: Simplified implementation - in production use proper ECDH
func (local *LocalNode) PerformECDH(peerPubKey []byte) ([]byte, error) {
	// Simplified: just hash both keys together for shared secret derivation
	// In production, use proper ECDH with library
	combined := append(local.PubKey.SerializeCompressed(), peerPubKey...)
	return crypto.Hash(combined), nil
}

// Encrypt encrypts data using a shared secret (AES-like, simplified)
func Encrypt(sharedSecret []byte, plaintext []byte) ([]byte, error) {
	// Simplified encryption - in production use AES-GCM
	// Key = SHA256(sharedSecret || nonce)
	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	keyMaterial := append(sharedSecret, nonce...)
	key := crypto.Hash(keyMaterial)

	// XOR encryption (simplified - use ChaCha20-Poly1305 in production)
	ciphertext := make([]byte, len(plaintext))
	for i, b := range plaintext {
		ciphertext[i] = b ^ key[i%len(key)]
	}

	// Prepend nonce
	result := append(nonce, ciphertext...)
	return result, nil
}

// Decrypt decrypts data using a shared secret
func Decrypt(sharedSecret []byte, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 8 {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:8]
	encrypted := ciphertext[8:]

	// Derive key
	keyMaterial := append(sharedSecret, nonce...)
	key := crypto.Hash(keyMaterial)

	// XOR decryption
	plaintext := make([]byte, len(encrypted))
	for i, b := range encrypted {
		plaintext[i] = b ^ key[i%len(key)]
	}

	return plaintext, nil
}

// =============================================================================
// Peer Credentials (for authentication)
// =============================================================================

// PeerCredentials represents an authenticated peer's identity
type PeerCredentials struct {
	NodeID    NodeID
	PubKey    []byte
	Timestamp time.Time
}

// NewPeerCredentials creates credentials from a version message
func NewPeerCredentials(nodeID NodeID, pubKey []byte) *PeerCredentials {
	return &PeerCredentials{
		NodeID:    nodeID,
		PubKey:    pubKey,
		Timestamp: time.Now(),
	}
}

// IsExpired returns true if credentials are too old
func (p *PeerCredentials) IsExpired(maxAge time.Duration) bool {
	return time.Since(p.Timestamp) > maxAge
}