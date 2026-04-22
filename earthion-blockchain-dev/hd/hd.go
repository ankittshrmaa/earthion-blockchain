package hd

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	"earthion/crypto"
)

// Hardened derivation offset (0x80000000)
const hardenedOffset uint32 = 0x80000000

var (
	ErrInvalidSeed        = errors.New("invalid seed")
	ErrInvalidPath        = errors.New("invalid derivation path")
	ErrInvalidKey         = errors.New("invalid key")
	ErrHardenedDerivation = errors.New("cannot derive hardened child from public key")
)

// ExtendedKey represents an HD wallet extended key (xprv/xpub)
type ExtendedKey struct {
	Version   []byte
	Depth     byte
	ParentF   []byte
	ChildNum  uint32
	ChainCode []byte
	Key       []byte
	IsPrivate bool
}

// NewMasterKey derives the master key from seed (BIP-39)
func NewMasterKey(seed []byte) (*ExtendedKey, error) {
	if len(seed) < 16 || len(seed) > 64 {
		return nil, ErrInvalidSeed
	}

	h := hmac.New(sha512.New, []byte("Bitcoin seed"))
	h.Write(seed)
	il := h.Sum(nil)

	privateKeyBytes := il[:32]
	chainCode := il[32:]

	return &ExtendedKey{
		Version:   []byte{0x04, 0x88, 0xAD, 0xE4},
		Depth:     0x00,
		ParentF:   []byte{0x00, 0x00, 0x00, 0x00},
		ChildNum:  0,
		ChainCode: chainCode,
		Key:       append([]byte{0x00}, privateKeyBytes...),
		IsPrivate: true,
	}, nil
}

// NeuteredKey converts private extended key to public extended key
func (ek *ExtendedKey) NeuteredKey() (*ExtendedKey, error) {
	if !ek.IsPrivate {
		return nil, ErrInvalidKey
	}

	return &ExtendedKey{
		Version:   []byte{0x04, 0x88, 0xB2, 0x1E},
		Depth:     ek.Depth,
		ParentF:   ek.ParentF,
		ChildNum:  ek.ChildNum,
		ChainCode: ek.ChainCode,
		Key:       ek.Key[1:],
		IsPrivate: false,
	}, nil
}

// PublicKey returns the public extended key
func (ek *ExtendedKey) PublicKey() *ExtendedKey {
	if !ek.IsPrivate {
		return ek
	}

	priv := secp256k1.PrivKeyFromBytes(ek.Key[1:])
	pub := priv.PubKey()

	return &ExtendedKey{
		Version:   []byte{0x04, 0x88, 0xB2, 0x1E},
		Depth:     ek.Depth,
		ParentF:   ek.ParentF,
		ChildNum:  ek.ChildNum,
		ChainCode: ek.ChainCode,
		Key:       pub.SerializeCompressed(),
		IsPrivate: false,
	}
}

// ChildKey derives a child key at the given index
func (ek *ExtendedKey) ChildKey(index uint32) (*ExtendedKey, error) {
	isHardened := index >= hardenedOffset

	if !ek.IsPrivate && isHardened {
		return nil, ErrHardenedDerivation
	}

	var data []byte
	if isHardened {
		data = append(ek.Key, uint32ToBytes(index)...)
	} else {
		data = append(ek.PublicKey().Key, uint32ToBytes(index)...)
	}

	h := hmac.New(sha512.New, ek.ChainCode)
	h.Write(data)
	il := h.Sum(nil)

	childChainCode := il[32:]
	childKeyBytes := il[:32]

	var newKeyBytes []byte
	if ek.IsPrivate {
		parentPriv := secp256k1.PrivKeyFromBytes(ek.Key[1:])
		childKeyInt := new(big.Int).SetBytes(childKeyBytes)

		// BIP-32: child_key = parent_key + IL (scalar addition mod N)
		parentD := parentPriv.ToECDSA().D
		parentD.Add(parentD, childKeyInt)
		parentD.Mod(parentD, secp256k1.S256().N)

		newPriv := secp256k1.PrivKeyFromBytes(parentD.Bytes())
		newKeyBytes = append([]byte{0x00}, newPriv.Serialize()...)
	} else {
		childPriv := secp256k1.PrivKeyFromBytes(childKeyBytes)
		pub := childPriv.PubKey()
		newKeyBytes = pub.SerializeCompressed()
	}

	parentFP := ek.ParentF
	if len(parentFP) == 0 {
		parentFP = crypto.Hash(ek.Key)[0:4]
	}

	return &ExtendedKey{
		Version:   ek.Version,
		Depth:     ek.Depth + 1,
		ParentF:   parentFP,
		ChildNum:  index,
		ChainCode: childChainCode,
		Key:       newKeyBytes,
		IsPrivate: ek.IsPrivate,
	}, nil
}

// DerivePath derives a child key from a path like "m/44'/0'/0'/0/0"
func (ek *ExtendedKey) DerivePath(path string) (*ExtendedKey, error) {
	child := ek
	segments, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	for _, idx := range segments {
		child, err = child.ChildKey(idx)
		if err != nil {
			return nil, err
		}
	}
	return child, nil
}

// Address returns the Base58Check encoded address
func (ek *ExtendedKey) Address() string {
	if ek.IsPrivate {
		return ek.PublicKey().Address()
	}
	pubHash := crypto.Hash(ek.Key)
	addr := append([]byte{0x00}, pubHash[:20]...)
	checksum := crypto.Checksum(addr)
	return string(crypto.Base58Encode(append(addr, checksum...)))
}

// AddressHex returns hex address
func (ek *ExtendedKey) AddressHex() string {
	if ek.IsPrivate {
		return ek.PublicKey().AddressHex()
	}
	return fmt.Sprintf("%x", crypto.Hash(ek.Key)[:20])
}

// PubKeyHash returns the public key hash (20 bytes)
func (ek *ExtendedKey) PubKeyHash() []byte {
	if ek.IsPrivate {
		return crypto.Hash(ek.PublicKey().Key)[:20]
	}
	return crypto.Hash(ek.Key)[:20]
}

// String returns the extended key as base58 string
func (ek *ExtendedKey) String() string {
	data := []byte{
		ek.Version[0], ek.Version[1], ek.Version[2], ek.Version[3],
		ek.Depth,
		ek.ParentF[0], ek.ParentF[1], ek.ParentF[2], ek.ParentF[3],
		uint8(ek.ChildNum >> 24), uint8(ek.ChildNum >> 16), uint8(ek.ChildNum >> 8), uint8(ek.ChildNum),
	}
	data = append(data, ek.ChainCode...)
	data = append(data, ek.Key...)
	checksum := crypto.Checksum(data)
	data = append(data, checksum...)
	return string(crypto.Base58Encode(data))
}

// PrivateKey returns the secp256k1 PrivateKey
func (ek *ExtendedKey) PrivateKey() (*secp256k1.PrivateKey, error) {
	if !ek.IsPrivate {
		return nil, ErrInvalidKey
	}
	return secp256k1.PrivKeyFromBytes(ek.Key[1:]), nil
}

// PublicKeyBytes returns the compressed public key bytes
func (ek *ExtendedKey) PublicKeyBytes() ([]byte, error) {
	if ek.IsPrivate {
		return ek.PublicKey().Key, nil
	}
	return ek.Key, nil
}

// Sign signs data with the private key
func (ek *ExtendedKey) Sign(data []byte) ([]byte, error) {
	if !ek.IsPrivate {
		return nil, ErrInvalidKey
	}
	priv := secp256k1.PrivKeyFromBytes(ek.Key[1:])
	if priv == nil {
		return nil, fmt.Errorf("failed to parse private key")
	}
	sig := ecdsa.Sign(priv, data)
	if sig == nil {
		return nil, fmt.Errorf("failed to create signature")
	}
	return sig.Serialize(), nil
}

// Helper functions
func uint32ToBytes(n uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, n)
	return b
}

func parsePath(path string) ([]uint32, error) {
	if len(path) == 0 {
		return nil, ErrInvalidPath
	}
	if len(path) >= 2 && path[:2] == "m/" {
		path = path[2:]
	}

	var indices []uint32
	parts := splitPath(path)
	for _, part := range parts {
		isHardened := false
		if len(part) > 0 && part[len(part)-1] == '\'' {
			isHardened = true
			part = part[:len(part)-1]
		}
		var n uint64
		_, err := fmt.Sscanf(part, "%d", &n)
		if err != nil {
			return nil, ErrInvalidPath
		}
		index := uint32(n)
		if isHardened {
			index += hardenedOffset
		}
		indices = append(indices, index)
	}
	return indices, nil
}

func splitPath(path string) []string {
	var parts []string
	var current string
	for _, c := range path {
		if c == '/' {
			if len(current) > 0 {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if len(current) > 0 {
		parts = append(parts, current)
	}
	return parts
}

// ParseExtendedKey parses a base58 encoded extended key
func ParseExtendedKey(key string) (*ExtendedKey, error) {
	data := crypto.Base58Decode(key)
	if len(data) < 78 {
		return nil, ErrInvalidKey
	}
	checksum := crypto.Checksum(data[:len(data)-4])
	if string(checksum) != string(data[len(data)-4:]) {
		return nil, ErrInvalidKey
	}
	ek := &ExtendedKey{
		Version:   data[0:4],
		Depth:     data[4],
		ParentF:   data[5:9],
		ChildNum:  binary.BigEndian.Uint32(data[9:13]),
		ChainCode: data[13:45],
		Key:       data[45:78],
	}
	if data[45] == 0x00 {
		ek.IsPrivate = true
	} else {
		ek.IsPrivate = false
	}
	return ek, nil
}
