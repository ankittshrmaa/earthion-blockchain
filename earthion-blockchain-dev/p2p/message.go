package p2p

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"earthion/crypto"
)

// =============================================================================
// Message Types
// =============================================================================

// MessageType represents the type of P2P message
type MessageType string

// Message type constants
const (
	// Control messages
	MsgVersion   MessageType = "version"
	MsgVerAck   MessageType = "verack"
	MsgPing     MessageType = "ping"
	MsgPong     MessageType = "pong"
	MsgGetAddr  MessageType = "getaddr"
	MsgAddr     MessageType = "addr"
	MsgReject   MessageType = "reject"

	// Block messages
	MsgGetBlocks  MessageType = "getblocks"
	MsgGetHeaders MessageType = "getheaders"
	MsgBlock     MessageType = "block"
	MsgHeaders   MessageType = "headers"

	// Transaction messages
	MsgGetTX   MessageType = "gettx"
	MsgTX      MessageType = "tx"

	// Inventory messages
	MsgInv    MessageType = "inv"
	MsgGetData MessageType = "getdata"
	MsgNotFound MessageType = "notfound"

	// Mempool
	MsgMemPool MessageType = "mempool"
)

// Message is a P2P network message
type Message struct {
	Type    MessageType
	Payload []byte
}

// =============================================================================
// Message Serialization
// =============================================================================

// Maximum message size (10MB)
const MaxMessageSize = 10 * 1024 * 1024

// WriteMessage writes a message to a connection
func WriteMessage(w io.Writer, msg *Message) error {
	// Serialize the message
	data, err := serializeMessage(msg)
	if err != nil {
		return fmt.Errorf("serialize: %w", err)
	}

	// Write length + message
	length := uint32(len(data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return fmt.Errorf("write length: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	return nil
}

// ReadMessage reads a message from a connection
// Protocol format: 4-byte length (big-endian) + payload
// The payload contains: 12-byte command + 4-byte checksum + actual data
func ReadMessage(r io.Reader) (*Message, error) {
	// Read length (4 bytes, big-endian)
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}

	// Validate length before reading
	// Minimum: 12 (command) + 4 (checksum) = 16 bytes header
	// Maximum: prevent memory exhaustion attacks
	if length == 0 {
		return nil, fmt.Errorf("invalid zero-length message")
	}
	if length > MaxMessageSize {
		return nil, fmt.Errorf("message too large: %d", length)
	}

	// Read the full message frame (command + checksum + payload)
	data := make([]byte, 20+length) // 20 = 12 (command) + 4 (length) + 4 (checksum)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	// Deserialize and validate the message
	msg, err := deserializeMessage(data)
	if err != nil {
		return nil, fmt.Errorf("deserialize: %w", err)
	}

	return msg, nil
}

// serializeMessage serializes a message to bytes
func serializeMessage(msg *Message) ([]byte, error) {
	// Format: command (12 bytes null-padded) + length (4 bytes) + checksum (4 bytes) + payload
	buf := make([]byte, 0, 12+4+4+len(msg.Payload))

	// Command (null-padded to 12 bytes)
	command := []byte(msg.Type)
	if len(command) > 12 {
		return nil, fmt.Errorf("command too long: %s", msg.Type)
	}
	buf = append(buf, command...)
	for len(buf) < 12 {
		buf = append(buf, 0)
	}

	// Length
	length := uint32(len(msg.Payload))
	buf = append(buf, byte(length>>24), byte(length>>16), byte(length>>8), byte(length))

	// Checksum (first 4 bytes of double SHA256)
	checksum := crypto.Hash(crypto.Hash(msg.Payload))
	buf = append(buf, checksum[:4]...)

	// Payload
	buf = append(buf, msg.Payload...)

	return buf, nil
}

// deserializeMessage deserializes a message from bytes
// The data includes: 12-byte command (null-padded) + 4-byte length + 4-byte checksum + payload
// This version extracts length from the data itself (no external parameter needed)
func deserializeMessage(data []byte) (*Message, error) {
	// Minimum: 12 (command) + 4 (length) + 4 (checksum) = 20 bytes header before payload
	if len(data) < 20 {
		return nil, fmt.Errorf("data too short: %d bytes (minimum 20 for header)", len(data))
	}

	// Parse command (first 12 bytes, null-padded)
	command := string(data[:12])
	// Trim trailing zeros to get actual command string
	for i := 0; i < len(command); i++ {
		if command[i] == 0 {
			command = command[:i]
			break
		}
	}
	if command == "" {
		return nil, fmt.Errorf("empty command field")
	}

	// Extract length from the header (bytes 12-15)
	length := uint32(data[12])<<24 | uint32(data[13])<<16 | uint32(data[14])<<8 | uint32(data[15])

	// Verify we have enough data for command + checksum + payload
	expectedLen := 20 + int(length)
	if len(data) < expectedLen {
		return nil, fmt.Errorf("data too short: expected %d bytes, got %d", expectedLen, len(data))
	}

	// Verify checksum (first 4 bytes of double SHA256 of payload)
	payload := data[20:20+length]
	checksum := crypto.Hash(crypto.Hash(payload))
	if string(checksum[:4]) != string(data[16:20]) {
		return nil, fmt.Errorf("checksum mismatch: invalid message payload")
	}

	return &Message{
		Type:    MessageType(command),
		Payload: payload,
	}, nil
}

// =============================================================================
// Version Message
// =============================================================================

// MessageVersion contains version information
type MessageVersion struct {
	Version    int    // Protocol version
	Services   uint64 // Services offered
	Timestamp  int64  // Unix timestamp
	BestHeight int    // Best block height
	ID         NodeID // Node ID
	// AddrMe     - Our advertised address (optional)
	// AddrYou   - Their address (optional)
}

// Serialize serializes the version message
func (v *MessageVersion) Serialize() []byte {
	// Version (4) + Services (8) + Timestamp (8) + BestHeight (4) + ID (32)
	buf := make([]byte, 56)

	binary.BigEndian.PutUint32(buf[0:4], uint32(v.Version))
	binary.BigEndian.PutUint64(buf[4:12], v.Services)
	binary.BigEndian.PutUint64(buf[12:20], uint64(v.Timestamp))
	binary.BigEndian.PutUint32(buf[20:24], uint32(v.BestHeight))
	copy(buf[24:56], v.ID[:])

	return buf
}

// DeserializeMessageVersion deserializes a version message
func DeserializeMessageVersion(data []byte) *MessageVersion {
	if len(data) < 56 {
		return nil
	}

	return &MessageVersion{
		Version:    int(binary.BigEndian.Uint32(data[0:4])),
		Services:  binary.BigEndian.Uint64(data[4:12]),
		Timestamp: int64(binary.BigEndian.Uint64(data[12:20])),
		BestHeight: int(binary.BigEndian.Uint32(data[20:24])),
		ID:         decodeNodeID(data[24:56]),
	}
}

func decodeNodeID(data []byte) NodeID {
	var id NodeID
	copy(id[:], data)
	return id
}

// =============================================================================
// Inventory/Vectors
// =============================================================================

const (
	InvTypeError  uint8 = 0
	InvTypeTX    uint8 = 1
	InvTypeBlock uint8 = 2
	InvTypeFilteredBlock uint8 = 3 // BIP37
	InvTypeCompactBlock uint8 = 4 // BIP152
)

// InventoryVector represents an inventory item
type InventoryVector struct {
	Type uint8
	Hash [32]byte
}

// InvMessage represents an inventory message
type InvMessage struct {
	Count int
	Vectors []InventoryVector
}

// NewInvMessage creates a new inventory message
func NewInvMessage() *InvMessage {
	return &InvMessage{
		Vectors: make([]InventoryVector, 0),
	}
}

// AddBlock adds a block to inventory
func (im *InvMessage) AddBlock(hash []byte) {
	if len(hash) != 32 {
		return
	}
	im.Vectors = append(im.Vectors, InventoryVector{
		Type: InvTypeBlock,
	})
	copy(im.Vectors[len(im.Vectors)-1].Hash[:], hash)
	im.Count = len(im.Vectors)
}

// AddTx adds a transaction to inventory
func (im *InvMessage) AddTx(txid []byte) {
	if len(txid) != 32 {
		return
	}
	im.Vectors = append(im.Vectors, InventoryVector{
		Type: InvTypeTX,
	})
	copy(im.Vectors[len(im.Vectors)-1].Hash[:], txid)
	im.Count = len(im.Vectors)
}

// Serialize serializes the inv message
func (im *InvMessage) Serialize() []byte {
	// Count (varint) + Vectors (36 each)
	size := 1 + len(im.Vectors)*36
	buf := make([]byte, size)

	// Write count as varint (simplified - just 1 byte for < 253)
	buf[0] = byte(im.Count)
	offset := 1

	for _, v := range im.Vectors {
		buf[offset] = v.Type
		copy(buf[offset+1:offset+33], v.Hash[:])
		offset += 36
	}

	return buf
}

// DeserializeInvMessage deserializes an inv message
func DeserializeInvMessage(data []byte) *InvMessage {
	if len(data) < 1 {
		return nil
	}

	count := int(data[0])
	if count*36+1 > len(data) {
		return nil
	}

	im := &InvMessage{
		Count: count,
		Vectors: make([]InventoryVector, count),
	}

	for i := 0; i < count; i++ {
		offset := 1 + i*36
		im.Vectors[i] = InventoryVector{
			Type: data[offset],
		}
		copy(im.Vectors[i].Hash[:], data[offset+1:offset+33])
	}

	return im
}

// =============================================================================
// Reject Message
// =============================================================================

// RejectCode represents the reason for rejection
type RejectCode uint8

const (
	RejectMalformed       RejectCode = 0x01
	RejectInvalid        RejectCode = 0x10
	RejectObsolete      RejectCode = 0x11
	RejectDuplicate    RejectCode = 0x12
	RejectNonStandard  RejectCode = 0x40
	RejectDust        RejectCode = 0x41
	RejectInsufficientFee RejectCode = 0x42
	RejectCheckpoint   RejectCode = 0x43
)

// RejectMessage represents a reject message
type RejectMessage struct {
	Message  MessageType // The message that was rejected
	CCode    RejectCode // Reason
	Reason   string     // Human-readable reason
	Hash     []byte     // Specific data (tx/block hash)
}

// Serialize serializes the reject message
func (rm *RejectMessage) Serialize() []byte {
	// Message (1 + command) + Code (1) + Reason (varint + string) + Hash (32)
	reason := []byte(rm.Reason)
	size := 1 + 1 + 1 + len(reason) + 32
	buf := make([]byte, size)

	offset := 0
	buf[offset] = byte(len(rm.Message))
	offset++

	copy(buf[offset:], rm.Message)
	offset += len(rm.Message)

	buf[offset] = byte(rm.CCode)
	offset++

	buf[offset] = byte(len(reason))
	offset++

	copy(buf[offset:], reason)
	offset += len(reason)

	if len(rm.Hash) == 32 {
		copy(buf[offset:], rm.Hash)
	}

	return buf
}

// =============================================================================
// Address Message
// =============================================================================

// AddrMessage represents an address message
type AddrMessage struct {
	Count   int64   // Number of addresses
	AddrList []AddrEntry
}

// AddrEntry represents a peer address entry
type AddrEntry struct {
	Timestamp time.Time
	Services  uint64
	Addr      string // IP:port
}

// Serialize serializes address message
func (am *AddrMessage) Serialize() []byte {
	count := len(am.AddrList)
	if count > 1000 {
		count = 1000
	}

	// Format: count (varint) + entries (timestamp 4 + services 8 + addr len 1 + addr)
	size := 1 + count*(4+8+1+22) // max IPv6 length
	buf := make([]byte, 0, size)

	// Count as varint (single byte for < 253)
	buf = append(buf, byte(count))

	// Entries
	for i := 0; i < count; i++ {
		entry := am.AddrList[i]

		// Timestamp (4 bytes, unix timestamp)
		ts := uint32(entry.Timestamp.Unix())
		buf = append(buf, byte(ts>>24), byte(ts>>16), byte(ts>>8), byte(ts))

		// Services (8 bytes)
		binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], entry.Services)
		buf = buf[:len(buf)+8]

		// Address length + address
		addrBytes := []byte(entry.Addr)
		if len(addrBytes) > 255 {
			addrBytes = addrBytes[:255]
		}
		buf = append(buf, byte(len(addrBytes)))
		buf = append(buf, addrBytes...)
	}

	return buf
}

// DeserializeAddrMessage deserializes address message
func DeserializeAddrMessage(data []byte) *AddrMessage {
	if len(data) < 1 {
		return nil
	}

	count := int(data[0])
	offset := 1

	if count*30+1 > len(data) {
		count = (len(data) - 1) / 30
	}

	addrMsg := &AddrMessage{
		Count:    int64(count),
		AddrList: make([]AddrEntry, 0, count),
	}

	for i := 0; i < count && offset+13 < len(data); i++ {
		// Timestamp (4 bytes)
		ts := int64(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		// Services (8 bytes)
		services := binary.BigEndian.Uint64(data[offset : offset+8])
		offset += 8

		// Address length
		addrLen := int(data[offset])
		offset++

		// Address
		if offset+addrLen <= len(data) {
			addr := string(data[offset : offset+addrLen])
			offset += addrLen

			addrMsg.AddrList = append(addrMsg.AddrList, AddrEntry{
				Timestamp: time.Unix(ts, 0),
				Services:  services,
				Addr:      addr,
			})
		}
	}

	return addrMsg
}

// =============================================================================
// GetBlocks/GetHeaders Request
// =============================================================================

// GetBlocksMessage represents a getblocks request
type GetBlocksMessage struct {
	Version   uint32
	BlockLocators [][]byte // Block hashes from chain
	HashStop   []byte      // Stop at this hash
}

// Serialize serializes getblocks
func (gbm *GetBlocksMessage) Serialize() []byte {
	count := len(gbm.BlockLocators)
	size := 4 + 1 + count*32 + 32
	buf := make([]byte, size)

	binary.BigEndian.PutUint32(buf[0:4], gbm.Version)
	buf[4] = byte(count)
	offset := 5

	for _, hash := range gbm.BlockLocators {
		copy(buf[offset:offset+32], hash)
		offset += 32
	}

	if len(gbm.HashStop) >= 32 {
		copy(buf[offset:], gbm.HashStop)
	}

	return buf
}

// GetHeadersMessage similar but for headers only
type GetHeadersMessage = GetBlocksMessage

// DeserializeGetBlocksMessage deserializes getblocks message
func DeserializeGetBlocksMessage(data []byte) *GetBlocksMessage {
	if len(data) < 5 {
		return nil
	}

	gbm := &GetBlocksMessage{}
	gbm.Version = binary.BigEndian.Uint32(data[0:4])

	count := int(data[4])
	if count*32+5 > len(data) {
		return nil
	}

	gbm.BlockLocators = make([][]byte, count)
	offset := 5
	for i := 0; i < count; i++ {
		gbm.BlockLocators[i] = make([]byte, 32)
		copy(gbm.BlockLocators[i], data[offset:offset+32])
		offset += 32
	}

	if offset+32 <= len(data) {
		gbm.HashStop = make([]byte, 32)
		copy(gbm.HashStop, data[offset:offset+32])
	}

	return gbm
}