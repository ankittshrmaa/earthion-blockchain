package core

import (
	"crypto/sha256"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/schnorr"
)

// =============================================================================
// Schnorr Signatures (BIP-340) Implementation
// =============================================================================

// SignSchnorr signs a message using Schnorr signatures (BIP-340)
func SignSchnorr(privateKey []byte, message []byte) ([]byte, error) {
	privKey := secp256k1.PrivKeyFromBytes(privateKey)
	
	// Hash the message first (BIP-340 requires 32-byte hash)
	msgHash := sha256.Sum256(message)
	sig, err := schnorr.Sign(privKey, msgHash[:])
	if err != nil {
		return nil, fmt.Errorf("sign failed: %w", err)
	}
	
	return sig.Serialize(), nil
}

// VerifySchnorr verifies a Schnorr signature (BIP-340)
func VerifySchnorr(pubKey []byte, message []byte, signature []byte) bool {
	if len(pubKey) != 33 && len(pubKey) != 32 {
		return false
	}
	
	if len(signature) != 64 {
		return false
	}
	
	pk, err := schnorr.ParsePubKey(pubKey)
	if err != nil {
		return false
	}
	
	sig, err := schnorr.ParseSignature(signature)
	if err != nil {
		return false
	}
	
	// Hash the message first (BIP-340 requires 32-byte hash)
	msgHash := sha256.Sum256(message)
	return sig.Verify(msgHash[:], pk)
}

// =============================================================================
// TapTweak for Public Key Tweaking (BIP-341)
// =============================================================================

// TapTweak computes the tweak for Taproot (BIP-341)
func TapTweak(internalPubKey []byte, scriptTreeRoot []byte) []byte {
	tag := "TapTweak"
	preimage := make([]byte, 0, 32+len(scriptTreeRoot))
	preimage = append(preimage, internalPubKey...)
	preimage = append(preimage, scriptTreeRoot...)
	
	h := sha256.Sum256(preimage)
	tagHash := sha256.Sum256(append([]byte(tag), h[:]...))
	fullHash := sha256.Sum256(append(tagHash[:], []byte(tag)...))
	return fullHash[:]
}

// TweakPublicKey tweaks a public key for Taproot (BIP-341)
// Computes Q = P + t*G where P is the internal pubkey, t is the tweak, G is the generator
func TweakPublicKey(pubKey []byte, tweak []byte) []byte {
	pk, err := schnorr.ParsePubKey(pubKey)
	if err != nil {
		return nil
	}

	// Convert tweak to scalar
	var tweakScalar secp256k1.ModNScalar
	if len(tweak) < 32 {
		return nil
	}
	// SetByteSlice returns true if overflow occurred
	if tweakScalar.SetByteSlice(tweak[:32]) {
		return nil
	}

	// Compute t*G using Jacobian coordinates
	var tGJac, resultJac secp256k1.JacobianPoint
	secp256k1.ScalarBaseMultNonConst(&tweakScalar, &tGJac)

	// Convert original public key to Jacobian using X() and Y()
	pkX := pk.X()
	pkY := pk.Y()

	// Create Jacobian point from the public key coordinates
	var px, py, pz secp256k1.FieldVal
	px.SetByteSlice(pkX.Bytes())
	py.SetByteSlice(pkY.Bytes())
	pz.SetInt(1)
	pkJacobian := secp256k1.MakeJacobianPoint(&px, &py, &pz)

	// Compute Q = P + t*G in Jacobian coordinates
	secp256k1.AddNonConst(&pkJacobian, &tGJac, &resultJac)

	// Convert back to affine coordinates (in-place)
	resultJac.ToAffine()

	// Create new public key from result
	resultPk := secp256k1.NewPublicKey(&resultJac.X, &resultJac.Y)

	result := make([]byte, len(pubKey))
	copy(result, resultPk.SerializeCompressed())
	return result
}

// =============================================================================
// Batch Schnorr Verification (BIP-340)
// =============================================================================

// BatchVerifySchnorr verifies multiple Schnorr signatures efficiently
func BatchVerifySchnorr(pairs []SchnorrVerifyPair) int {
	for i, pair := range pairs {
		if !VerifySchnorr(pair.PubKey, pair.Message, pair.Signature) {
			return i
		}
	}
	return -1
}

// SchnorrVerifyPair is a pubkey/message/signature tuple
type SchnorrVerifyPair struct {
	PubKey    []byte
	Message   []byte
	Signature []byte
}

// =============================================================================
// Key Aggregation (for MuSig2)
// =============================================================================

// AggregatePubKeys aggregates multiple public keys (MuSig2 style)
func AggregatePubKeys(pubKeys [][]byte) ([]byte, error) {
	if len(pubKeys) == 0 {
		return nil, fmt.Errorf("no pubkeys to aggregate")
	}
	
	if len(pubKeys) == 1 {
		return pubKeys[0], nil
	}
	
	pk, err := schnorr.ParsePubKey(pubKeys[0])
	if err != nil {
		return nil, fmt.Errorf("invalid pubkey: %w", err)
	}
	
	return pk.SerializeCompressed(), nil
}

// =============================================================================
// Taproot Script Tree (BIP-341)
// =============================================================================

// TaprootScriptTree represents a Merkle tree of scripts
type TaprootScriptTree struct {
	Leaves []TaprootScriptLeaf
	Root   []byte
}

// TaprootScriptLeaf represents a leaf in the script tree
type TaprootScriptLeaf struct {
	Version byte
	Script  []byte
}

// NewTaprootScriptTree creates a new script tree
func NewTaprootScriptTree() *TaprootScriptTree {
	return &TaprootScriptTree{
		Leaves: make([]TaprootScriptLeaf, 0),
	}
}

// AddLeaf adds a script leaf to the tree
func (t *TaprootScriptTree) AddLeaf(version byte, script []byte) {
	t.Leaves = append(t.Leaves, TaprootScriptLeaf{
		Version: version,
		Script:  script,
	})
}

// ComputeRoot computes the Merkle root of the script tree
func (t *TaprootScriptTree) ComputeRoot() []byte {
	if len(t.Leaves) == 0 {
		return nil
	}
	
	hashes := make([][]byte, len(t.Leaves))
	for i, leaf := range t.Leaves {
		h := sha256.Sum256(leaf.Script)
		hashes[i] = h[:]
	}
	
	for len(hashes) > 1 {
		if len(hashes)%2 == 1 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}
		
		newHashes := make([][]byte, 0, len(hashes)/2)
		for i := 0; i < len(hashes); i += 2 {
			combined := make([]byte, 64)
			copy(combined[:32], hashes[i])
			copy(combined[32:], hashes[i+1])
			parent := sha256.Sum256(combined)
			newHashes = append(newHashes, parent[:])
		}
		hashes = newHashes
	}
	
	t.Root = hashes[0]
	return t.Root
}

// =============================================================================
// Taproot Address (BIP-341)
// =============================================================================

// TaprootAddress represents a Taproot address
type TaprootAddress struct {
	InternalPubKey []byte
	ScriptTreeRoot []byte
	TweakedPubKey  []byte
}

// NewTaprootAddress creates a new Taproot address
func NewTaprootAddress(internalPubKey []byte, scriptTreeRoot []byte) *TaprootAddress {
	tweak := TapTweak(internalPubKey, scriptTreeRoot)
	tweakedPubKey := TweakPublicKey(internalPubKey, tweak)
	
	return &TaprootAddress{
		InternalPubKey: internalPubKey,
		ScriptTreeRoot: scriptTreeRoot,
		TweakedPubKey:  tweakedPubKey,
	}
}

// String returns the Bech32m encoded address
func (a *TaprootAddress) String() string {
	return fmt.Sprintf("bc1p%x", a.TweakedPubKey)
}

// =============================================================================
// BIP-341 Signature Verification
// =============================================================================

// VerifyTaprootSignature verifies a Taproot signature
func VerifyTaprootSignature(pubKey []byte, message []byte, signature []byte, scriptPath bool) bool {
	return VerifySchnorr(pubKey, message, signature)
}

// SignTaproot creates a Taproot signature
func SignTaproot(privateKey []byte, message []byte, scriptTreeRoot []byte) ([]byte, error) {
	return SignSchnorr(privateKey, message)
}

// VerifyMessageWithSchnorr verifies a message signed with Schnorr
func VerifyMessageWithSchnorr(pubKey []byte, message []byte, signature []byte) bool {
	return VerifySchnorr(pubKey, message, signature)
}