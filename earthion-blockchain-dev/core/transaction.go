package core

import (
	"bytes"
	"encoding/hex"
	"encoding/gob"
	"fmt"
	"strings"

	"earthion/crypto"
	"earthion/wallet"
)

// ChainID is used to prevent cross-chain replay attacks
// Change this value when creating a new chain to invalidate old transactions
var ChainID = []byte{0x01} // Version 1

type Transaction struct {
	ID      []byte
	Inputs  []TXInput
	Outputs []TXOutput
}

type TXInput struct {
	Txid []byte
	OutIndex int
	Signature []byte
	PubKey []byte
}

type TXOutput struct {
	Value int
	PubKey []byte
}

func (tx *Transaction) Serialize() []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	_ = encoder.Encode(tx)
	return buf.Bytes()
}

// SerializeWithoutID returns canonical serialization without the ID
// This is used for computing a deterministic TX ID
func (tx *Transaction) SerializeWithoutID() []byte {
	// Create a copy without ID to get canonical serialization
	txCopy := &Transaction{
		ID:      nil, // No ID for canonical form
		Inputs:  make([]TXInput, len(tx.Inputs)),
		Outputs: make([]TXOutput, len(tx.Outputs)),
	}
	for i, in := range tx.Inputs {
		txCopy.Inputs[i] = TXInput{
			Txid:      in.Txid,
			OutIndex:  in.OutIndex,
			Signature: nil, // Exclude signature for canonical form
			PubKey:    in.PubKey,
		}
	}
	for i, out := range tx.Outputs {
		txCopy.Outputs[i] = TXOutput{
			Value:  out.Value,
			PubKey: out.PubKey,
		}
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	_ = encoder.Encode(txCopy)
	return buf.Bytes()
}

// SetID generates a unique, deterministic ID for the transaction
// Uses chain ID + canonical serialized data to prevent malleability
func (tx *Transaction) SetID() {
	// Use canonical serialization (without ID or signatures)
	// This prevents TXID malleability attacks
	canonicalData := tx.SerializeWithoutID()

	// Include chain ID to prevent cross-chain replay
	data := append(ChainID, canonicalData...)

	// Double hash to get the final TX ID
	tx.ID = crypto.DoubleHash(data)
}

// Sign signs the transaction data with the private key
// The signature covers the TX ID + outputs to ensure transaction integrity
func (tx *Transaction) Sign(privateKey *wallet.Wallet) error {
	tx.SetID()

	data := tx.GetSignData()

	for i := range tx.Inputs {
		sig := privateKey.Sign(data)
		if sig == nil {
			return fmt.Errorf("failed to sign transaction input %d", i)
		}
		tx.Inputs[i].Signature = sig
	}

	return nil
}

// GetSignData returns the data to be signed
// This must match between Sign() and Verify() - uses TX ID + output values + chain ID
func (tx *Transaction) GetSignData() []byte {
	var data []byte

	// Include chain ID to bind this transaction to this chain
	data = append(data, ChainID...)

	// Include transaction ID (already includes chain ID in its computation)
	data = append(data, tx.ID...)

	// Include output values and pubkeys (deterministic, no signatures)
	for _, out := range tx.Outputs {
		data = append(data, IntToHex(int64(out.Value))...)
		data = append(data, out.PubKey...)
	}

	return crypto.Hash(data)
}

// Verify checks if the transaction signature is valid
func (tx *Transaction) Verify() bool {
	if len(tx.Inputs) == 0 {
		return false
	}

	// Coinbase transactions don't need verification
	if tx.IsCoinbase() {
		return true
	}

	data := tx.GetSignData()

	for _, input := range tx.Inputs {
		// CRITICAL: Both signature and pubkey must be present
		if len(input.Signature) == 0 {
			return false
		}
		if len(input.PubKey) == 0 {
			return false
		}
		if !wallet.VerifySignature(input.PubKey, data, input.Signature) {
			return false
		}
	}

	return true
}

// IsCoinbase checks if this is a coinbase transaction
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && tx.Inputs[0].OutIndex == -1
}

// CoinbaseTx creates a reward transaction for mining
// pubKey should be the 20-byte pubkey hash (not full public key)
// blockIndex and prevBlockHash are used to create a unique TX ID
func CoinbaseTx(pubKeyHash []byte, amount int, blockIndex int, prevBlockHash []byte) *Transaction {
	tx := &Transaction{
		Inputs: []TXInput{
			{
				Txid: []byte{},
				OutIndex: -1,
				Signature: nil,
				PubKey: pubKeyHash, // Store pubkey hash
			},
		},
		Outputs: []TXOutput{
			{
				Value: amount,
				PubKey: pubKeyHash, // Store pubkey hash (20 bytes)
			},
		},
	}

	// Generate unique TX ID using block index + prev block hash + timestamp
	tx.SetUniqueID(blockIndex, prevBlockHash)
	return tx
}

// SetUniqueID generates a unique coinbase TX ID
// Uses block index + previous block hash for cryptographic uniqueness
func (tx *Transaction) SetUniqueID(blockIndex int, prevBlockHash []byte) {
	data := tx.Serialize()
	data = append(data, IntToHex(int64(blockIndex))...)
	data = append(data, prevBlockHash...)
	tx.ID = crypto.DoubleHash(data)
}

// SetIDWithIndex generates a unique ID using block index
func (tx *Transaction) SetIDWithIndex(blockIndex int) {
	// Include block index to make TX ID unique
	data := tx.Serialize()
	data = append(data, IntToHex(int64(blockIndex))...)
	tx.ID = crypto.DoubleHash(data)
}

// NewTransaction creates a new transaction that spends previous outputs
// to should be 20-byte pubkey hash
func NewTransaction(from *wallet.Wallet, to []byte, amount int, bc *Blockchain) (*Transaction, error) {
	utxos := bc.UTXOIndex()
	// Use raw address (20-byte pubkey hash) for balance lookup
	balance := bc.GetBalance(from.GetRawAddress())

	if amount > balance {
		return nil, fmt.Errorf("insufficient balance: need %d, have %d", amount, balance)
	}

	var inputs []TXInput
	var outputs []TXOutput

	// Gather UTXOs
	var total int
	for key, out := range utxos {
		// Parse key to get txID and outIdx - key format is "txIDHex:outIdx"
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			continue
		}
		txIDHex := parts[0]
		outIdx := 0
		if _, err := fmt.Sscanf(parts[1], "%d", &outIdx); err != nil {
			continue
		}

		txID, _ := hex.DecodeString(txIDHex)

		// Check if output belongs to sender - TXOutput.PubKey IS the pubkey hash (20 bytes)
		if !bytes.Equal(out.PubKey, from.GetRawAddress()) {
			continue
		}

		// Validate that the referenced output exists in the UTXO set
		// This ensures we're spending a valid, existing output
		if _, exists := utxos[key]; !exists {
			continue // Skip already-spent outputs
		}

		inputs = append(inputs, TXInput{
			Txid:      txID,
			OutIndex:  outIdx,
			Signature: nil,
			PubKey:    from.PublicKey, // Input stores full pubkey for verification
		})

		total += out.Value

		if total >= amount {
			break
		}
	}

	// Validate we have enough inputs
	if len(inputs) == 0 && amount > 0 {
		return nil, fmt.Errorf("no valid UTXOs found for spending")
	}

	// Output to recipient - store pubkey hash (20 bytes)
	outputs = append(outputs, TXOutput{
		Value: amount,
		PubKey: to, // Should be 20-byte pubkey hash
	})

	// Output change back to sender - store pubkey hash (20 bytes)
	if total > amount {
		outputs = append(outputs, TXOutput{
			Value: total - amount,
			PubKey: from.GetRawAddress(), // Use pubkey hash
		})
	}

	// CRITICAL: Validate that total input value equals total output value
	// This prevents creating or destroying coins
	var totalOutputValue int
	for _, out := range outputs {
		totalOutputValue += out.Value
	}

	if total != totalOutputValue {
		return nil, fmt.Errorf("input/output value mismatch: inputs=%d, outputs=%d", total, totalOutputValue)
	}

	tx := &Transaction{
		Inputs: inputs,
		Outputs: outputs,
	}

	if err := tx.Sign(from); err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	
	return tx, nil
}

// =============================================================================
// JSON Serialization Helpers
// =============================================================================

// TransactionJSON is a JSON-friendly version of Transaction
type TransactionJSON struct {
	ID       string          `json:"id"`
	Inputs  []TXInputJSON   `json:"inputs"`
	Outputs []TXOutputJSON `json:"outputs"`
}

// TXInputJSON is a JSON-friendly version of TXInput
type TXInputJSON struct {
	Txid     string `json:"txid"`
	OutIndex int    `json:"outIndex"`
	Signature string `json:"signature"`
	PubKey   string `json:"pubKey"`
}

// TXOutputJSON is a JSON-friendly version of TXOutput
type TXOutputJSON struct {
	Value  int    `json:"value"`
	PubKey string `json:"pubKey"`
}

// ToJSON converts Transaction to JSON format
func (tx *Transaction) ToJSON() TransactionJSON {
	tj := TransactionJSON{
		ID: hex.EncodeToString(tx.ID),
	}

	tj.Inputs = make([]TXInputJSON, len(tx.Inputs))
	for i, in := range tx.Inputs {
		tj.Inputs[i] = TXInputJSON{
			Txid:     hex.EncodeToString(in.Txid),
			OutIndex: in.OutIndex,
			Signature: hex.EncodeToString(in.Signature),
			PubKey:   hex.EncodeToString(in.PubKey),
		}
	}

	tj.Outputs = make([]TXOutputJSON, len(tx.Outputs))
	for i, out := range tx.Outputs {
		tj.Outputs[i] = TXOutputJSON{
			Value:  out.Value,
			PubKey: hex.EncodeToString(out.PubKey),
		}
	}

	return tj
}

// FromJSON converts JSON format to Transaction
func TransactionFromJSON(tj TransactionJSON) *Transaction {
	tx := &Transaction{
		ID: decodeHex(tj.ID),
	}

	tx.Inputs = make([]TXInput, len(tj.Inputs))
	for i, in := range tj.Inputs {
		tx.Inputs[i] = TXInput{
			Txid:     decodeHex(in.Txid),
			OutIndex: in.OutIndex,
			Signature: decodeHex(in.Signature),
			PubKey:   decodeHex(in.PubKey),
		}
	}

	tx.Outputs = make([]TXOutput, len(tj.Outputs))
	for i, out := range tj.Outputs {
		tx.Outputs[i] = TXOutput{
			Value:  out.Value,
			PubKey: decodeHex(out.PubKey),
		}
	}

	return tx
}

func decodeHex(s string) []byte {
	if s == "" {
		return []byte{}
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return []byte{} // Return empty on error rather than silently ignoring
	}
	return b
}

// =============================================================================
// MessagePack Serialization (more efficient than JSON)
// =============================================================================

// TXInputMsgPack is a msgpack-friendly version of TXInput
type TXInputMsgPack struct {
	Txid     string `msgpack:"txid"`
	OutIndex int    `msgpack:"outIndex"`
	Signature string `msgpack:"signature"`
	PubKey   string `msgpack:"pubKey"`
}

// TXOutputMsgPack is a msgpack-friendly version of TXOutput
type TXOutputMsgPack struct {
	Value  int    `msgpack:"value"`
	PubKey string `msgpack:"pubKey"`
}

// TransactionMsgPack is a msgpack-friendly version of Transaction
type TransactionMsgPack struct {
	ID       string              `msgpack:"id"`
	Inputs  []TXInputMsgPack   `msgpack:"inputs"`
	Outputs []TXOutputMsgPack `msgpack:"outputs"`
}

// ToMsgPack converts Transaction to MessagePack format
func (tx *Transaction) ToMsgPack() TransactionMsgPack {
	tm := TransactionMsgPack{
		ID: hex.EncodeToString(tx.ID),
	}

	tm.Inputs = make([]TXInputMsgPack, len(tx.Inputs))
	for i, in := range tx.Inputs {
		tm.Inputs[i] = TXInputMsgPack{
			Txid:     hex.EncodeToString(in.Txid),
			OutIndex: in.OutIndex,
			Signature: hex.EncodeToString(in.Signature),
			PubKey:   hex.EncodeToString(in.PubKey),
		}
	}

	tm.Outputs = make([]TXOutputMsgPack, len(tx.Outputs))
	for i, out := range tx.Outputs {
		tm.Outputs[i] = TXOutputMsgPack{
			Value:  out.Value,
			PubKey: hex.EncodeToString(out.PubKey),
		}
	}

	return tm
}

// TransactionFromMsgPack converts MessagePack format to Transaction
func TransactionFromMsgPack(tm TransactionMsgPack) *Transaction {
	tx := &Transaction{
		ID: decodeHex(tm.ID),
	}

	tx.Inputs = make([]TXInput, len(tm.Inputs))
	for i, in := range tm.Inputs {
		tx.Inputs[i] = TXInput{
			Txid:     decodeHex(in.Txid),
			OutIndex: in.OutIndex,
			Signature: decodeHex(in.Signature),
			PubKey:   decodeHex(in.PubKey),
		}
	}

	tx.Outputs = make([]TXOutput, len(tm.Outputs))
	for i, out := range tm.Outputs {
		tx.Outputs[i] = TXOutput{
			Value:  out.Value,
			PubKey: decodeHex(out.PubKey),
		}
	}

	return tx
}
