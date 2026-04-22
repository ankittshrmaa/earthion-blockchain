package core

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"earthion/crypto"
)

// =============================================================================
// SegWit (BIP-141) Implementation
// =============================================================================

const (
	WITNESS_SCALE_FACTOR = 4
	MAX_BLOCK_WEIGHT     = 4000000
	MAX_BLOCK_SIZE      = 1000000
	WITNESS_V0          = 0x00
)

// WitnessProgram represents a SegWit witness program
type WitnessProgram struct {
	Version byte
	Program []byte
}

// Witness represents witness data for a transaction
type Witness struct {
	Items [][]byte
}

// NewWitness creates a new witness
func NewWitness() *Witness {
	return &Witness{Items: make([][]byte, 0)}
}

// Add adds witness data
func (w *Witness) Add(data []byte) {
	w.Items = append(w.Items, data)
}

// Serialize serializes the witness
func (w *Witness) Serialize() []byte {
	var buf bytes.Buffer
	var countVarint [10]byte
	n := encodeVarint36(countVarint[:], uint64(len(w.Items)))
	buf.Write(countVarint[:n])
	for _, item := range w.Items {
		var itemVarint [10]byte
		n := encodeVarint36(itemVarint[:], uint64(len(item)))
		buf.Write(itemVarint[:n])
		buf.Write(item)
	}
	return buf.Bytes()
}

// DeserializeWitness deserializes witness data
func DeserializeWitness(data []byte) (*Witness, error) {
	w := &Witness{}
	offset := 0
	count, n := decodeVarint36(data[offset:])
	offset += n
	w.Items = make([][]byte, 0, count)
	for i := uint64(0); i < count; i++ {
		itemLen, n := decodeVarint36(data[offset:])
		offset += n
		if offset+int(itemLen) > len(data) {
			return nil, fmt.Errorf("witness data truncated")
		}
		w.Items = append(w.Items, data[offset:offset+int(itemLen)])
		offset += int(itemLen)
	}
	return w, nil
}

// P2WPKH (Pay to Witness Public Key Hash) - SegWit v0
type P2WPKH struct {
	AddressHash []byte
}

func NewP2WPKH(addressHash []byte) *P2WPKH {
	if len(addressHash) != 20 {
		return nil
	}
	return &P2WPKH{AddressHash: addressHash}
}

func (p *P2WPKH) Script() []byte {
	script := []byte{0x00, 0x14}
	script = append(script, p.AddressHash...)
	return script
}

// P2WSH (Pay to Witness Script Hash) - SegWit v0
type P2WSH struct {
	ScriptHash []byte
}

func NewP2WSH(scriptHash []byte) *P2WSH {
	if len(scriptHash) != 32 {
		return nil
	}
	return &P2WSH{ScriptHash: scriptHash}
}

func (p *P2WSH) Script() []byte {
	script := []byte{0x00, 0x20}
	script = append(script, p.ScriptHash...)
	return script
}

// SegWitTransaction represents a SegWit-encoded transaction
type SegWitTransaction struct {
	Version  int32
	Inputs   []SegWitTxInput
	Outputs  []SegWitTxOutput
	Witness  []Witness
	LockTime uint32
	Flag     byte
}

// SegWitTxInput represents a SegWit transaction input
type SegWitTxInput struct {
	Txid      []byte
	OutIndex  int
	Sequence  uint32
}

// SegWitTxOutput represents a SegWit transaction output
type SegWitTxOutput struct {
	Value  int64
	Script []byte
}

// NewSegWitTransaction creates a new SegWit transaction
func NewSegWitTransaction() *SegWitTransaction {
	return &SegWitTransaction{
		Version: 2,
		Flag:    0x01,
		Inputs:  make([]SegWitTxInput, 0),
		Outputs: make([]SegWitTxOutput, 0),
		Witness: make([]Witness, 0),
	}
}

// Serialize serializes the SegWit transaction (BIP-144 format)
func (tx *SegWitTransaction) Serialize() []byte {
	var buf bytes.Buffer
	var version [4]byte
	binary.LittleEndian.PutUint32(version[:], uint32(tx.Version))
	buf.Write(version[:])
	buf.WriteByte(tx.Flag)
	var countBuf [10]byte
	n := encodeVarint36(countBuf[:], uint64(len(tx.Inputs)))
	buf.Write(countBuf[:n])
	for _, in := range tx.Inputs {
		buf.Write(in.Txid)
		var outIdx [4]byte
		binary.LittleEndian.PutUint32(outIdx[:], uint32(in.OutIndex))
		buf.Write(outIdx[:])
		buf.WriteByte(0)
		var seq [4]byte
		binary.LittleEndian.PutUint32(seq[:], in.Sequence)
		buf.Write(seq[:])
	}
	n = encodeVarint36(countBuf[:], uint64(len(tx.Outputs)))
	buf.Write(countBuf[:n])
	for _, out := range tx.Outputs {
		var value [8]byte
		binary.LittleEndian.PutUint64(value[:], uint64(out.Value))
		buf.Write(value[:])
		n = encodeVarint36(countBuf[:], uint64(len(out.Script)))
		buf.Write(countBuf[:n])
		buf.Write(out.Script)
	}
	for _, w := range tx.Witness {
		buf.Write(w.Serialize())
	}
	var lock [4]byte
	binary.LittleEndian.PutUint32(lock[:], tx.LockTime)
	buf.Write(lock[:])
	return buf.Bytes()
}

// CalculateWeight calculates transaction weight
// Weight = 3 * baseSize + witnessSize
// baseSize = serialized transaction without witness (flag=0)
// witnessSize = serialized witness data
func (tx *SegWitTransaction) CalculateWeight() int64 {
	// Calculate base size (without witness)
	baseSize := 4 + 1 + varintSize(uint64(len(tx.Inputs)))
	for range tx.Inputs {
		baseSize += 32 + 4 + 1 + 4 // Txid + outIdx + scriptLen + sequence
	}
	baseSize += varintSize(uint64(len(tx.Outputs)))
	for _, out := range tx.Outputs {
		baseSize += 8 + varintSize(uint64(len(out.Script))) + len(out.Script)
	}
	baseSize += 4 // lockTime

	// Calculate witness size separately
	witnessSize := 0
	for _, w := range tx.Witness {
		witnessSize += len(w.Serialize())
	}

	return int64(baseSize*3 + witnessSize)
}

// varintSize returns the size of a varint encoding
func varintSize(val uint64) int {
	if val < 253 {
		return 1
	}
	if val < 0x10000 {
		return 3
	}
	if val < 0x100000000 {
		return 5
	}
	return 9
}

// CalculateVSize calculates virtual size
func (tx *SegWitTransaction) CalculateVSize() int {
	weight := tx.CalculateWeight()
	return int((weight + 3) / 4)
}

// encodeVarint36 encodes a varint
func encodeVarint36(buf []byte, val uint64) int {
	if val < 253 {
		buf[0] = byte(val)
		return 1
	}
	if val < 0x10000 {
		buf[0] = 0xfd
		binary.LittleEndian.PutUint16(buf[1:], uint16(val))
		return 3
	}
	if val < 0x100000000 {
		buf[0] = 0xfe
		binary.LittleEndian.PutUint32(buf[1:], uint32(val))
		return 5
	}
	buf[0] = 0xff
	binary.LittleEndian.PutUint64(buf[1:], val)
	return 9
}

// decodeVarint36 decodes a varint
func decodeVarint36(data []byte) (uint64, int) {
	if len(data) < 1 {
		return 0, 0
	}
	first := data[0]
	if first < 253 {
		return uint64(first), 1
	}
	if first == 0xfd {
		if len(data) < 3 {
			return 0, 0
		}
		return uint64(binary.LittleEndian.Uint16(data[1:])), 3
	}
	if first == 0xfe {
		if len(data) < 5 {
			return 0, 0
		}
		return uint64(binary.LittleEndian.Uint32(data[1:])), 5
	}
	if len(data) < 9 {
		return 0, 0
	}
	return binary.LittleEndian.Uint64(data[1:]), 9
}

// GenerateP2WPKH creates a P2WPKH script
func GenerateP2WPKH(pubKeyHash []byte) []byte {
	return append([]byte{0x00, 0x14}, pubKeyHash...)
}

// GenerateP2WSH creates a P2WSH script from a script
func GenerateP2WSH(script []byte) []byte {
	hash := crypto.Hash(script)
	return append([]byte{0x00, 0x20}, hash...)
}

// IsP2WPKH checks if output is P2WPKH
func IsP2WPKH(script []byte) bool {
	return len(script) == 22 && script[0] == 0x00 && script[1] == 0x14
}

// IsP2WSH checks if output is P2WSH
func IsP2WSH(script []byte) bool {
	return len(script) == 34 && script[0] == 0x00 && script[1] == 0x20
}

// ExtractWitnessProgram extracts the witness program from output
func ExtractWitnessProgram(script []byte) *WitnessProgram {
	if len(script) == 22 && script[0] == 0x00 && script[1] == 0x14 {
		return &WitnessProgram{Version: 0, Program: script[2:22]}
	}
	if len(script) == 34 && script[0] == 0x00 && script[1] == 0x20 {
		return &WitnessProgram{Version: 0, Program: script[2:34]}
	}
	return nil
}