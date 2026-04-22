package wallet

import (
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"

	"earthion/crypto"
)

type Wallet struct {
	PrivateKeyBytes []byte
	PublicKey       []byte
}

func NewWallet() (*Wallet, error) {
	privateKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Get public key from private key
	_, pubKey := btcec.PrivKeyFromBytes(privateKey.Serialize())

	return &Wallet{
		PrivateKeyBytes: privateKey.Serialize(),
		PublicKey:       pubKey.SerializeCompressed(),
	}, nil
}

func decodePublicKey(data []byte) (*btcec.PublicKey, error) {
	return btcec.ParsePubKey(data)
}

// Sign signs data with the private key and returns signature
// Returns nil if signing fails
func (w *Wallet) Sign(data []byte) []byte {
	privateKey, err := btcec.PrivKeyFromBytes(w.PrivateKeyBytes)
	if err != nil {
		return nil
	}
	signature := ecdsa.Sign(privateKey, data)
	if signature == nil {
		return nil
	}
	return signature.Serialize()
}

func VerifySignature(pubKey []byte, data []byte, signature []byte) bool {
	if len(signature) == 0 || len(pubKey) == 0 {
		return false
	}

	decode, err := btcec.ParsePubKey(pubKey)
	if err != nil {
		return false
	}

	sig, err := ecdsa.ParseDERSignature(signature)
	if err != nil {
		return false
	}

	return sig.Verify(data, decode)
}

// Address from public key with Base58 and checksum
func (w *Wallet) Address() []byte {
	pubHash := crypto.Hash(w.PublicKey)
	addressBytes := pubHash[:20]
	fullAddress := crypto.AddChecksum(addressBytes)
	return crypto.Base58Encode(fullAddress)
}

func (w *Wallet) AddressHex() string {
	return hex.EncodeToString(w.Address())
}

// GetRawAddress returns the address bytes (without Base58 encoding)
func (w *Wallet) GetRawAddress() []byte {
	pubHash := crypto.Hash(w.PublicKey)
	return pubHash[:20]
}

// Wallet persistence
func (w *Wallet) Save(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(w)
}

func LoadWallet(filename string) (*Wallet, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	wallet := &Wallet{}
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(wallet)
	if err != nil {
		return nil, err
	}

	// Validate the private key bytes
	if len(wallet.PrivateKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid private key length")
	}

	return wallet, nil
}
