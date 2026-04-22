package p2p

import (
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"earthion/core"
)

// ConnectionState represents the state of a peer connection
type ConnectionState int

const (
	StateConnected    ConnectionState = iota // Initial connection established
	StateHandshake                          // Performing version handshake
	StateReady                              // Ready for messaging
	StateDisconnecting                      // Graceful disconnect
)

// Peer represents a connected peer in the network
type Peer struct {
	// Identity
	NodeID    NodeID
	Addr      net.Addr
	Direction Direction // Inbound or Outbound

	// Connection
	conn      net.Conn
	connMutex sync.RWMutex

	// State
	state     atomic.Int32
	lastPing  atomic.Int64
	version   *MessageVersion

	// Capabilities
	Services    uint64
	ProtocolVersion int

	// Chain state
	BestHeight int
	KnownBlocks map[string]bool   // Known block hashes
	KnownTXs    map[string]bool  // Known transaction hashes

	// Metrics
	bytesSent   atomic.Int64
	bytesRecv   atomic.Int64
	connectedAt time.Time

	// Callbacks (set by server)
	onBlock      func(*core.Block)
	onTransaction func(*core.Transaction)
	onDisconnect func(*Peer)

	// For inbound connections, we store the credentials after handshake
	Credentials *PeerCredentials

	// Messaging
	msgChan chan *Message
	quit    chan struct{}
	wg      sync.WaitGroup
}

// Direction indicates if connection is inbound or outbound
type Direction bool

const (
	DirInbound  Direction = true
	DirOutbound Direction = false
)

// NewPeer creates a new peer object
func NewPeer(conn net.Conn, dir Direction, nodeID NodeID) *Peer {
	p := &Peer{
		Addr:       conn.RemoteAddr(),
		Direction:  dir,
		NodeID:     nodeID,
		conn:       conn,
		KnownBlocks: make(map[string]bool),
		KnownTXs:   make(map[string]bool),
		msgChan:    make(chan *Message, 256),
		quit:       make(chan struct{}),
		connectedAt: time.Now(),
	}
	p.state.Store(int32(StateConnected))
	return p
}

// SetConnection sets the underlying connection
func (p *Peer) SetConnection(conn net.Conn) {
	p.connMutex.Lock()
	defer p.connMutex.Unlock()
	p.conn = conn
}

// GetConnection returns the underlying connection (thread-safe)
func (p *Peer) GetConnection() net.Conn {
	p.connMutex.RLock()
	defer p.connMutex.RUnlock()
	return p.conn
}

// SetState sets the connection state
func (p *Peer) SetState(state ConnectionState) {
	p.state.Store(int32(state))
}

// GetState gets the connection state
func (p *Peer) GetState() ConnectionState {
	return ConnectionState(p.state.Load())
}

// IsReady returns true if peer is ready for messaging
func (p *Peer) IsReady() bool {
	return p.GetState() == StateReady
}

// SetVersion sets the version information
func (p *Peer) SetVersion(v *MessageVersion) {
	p.version = v
	p.ProtocolVersion = v.Version
	p.BestHeight = v.BestHeight
	p.Services = v.Services
}

// UpdateLastPing updates the last ping timestamp
func (p *Peer) UpdateLastPing() {
	p.lastPing.Store(time.Now().UnixNano())
}

// LastPingTime returns the last ping time
func (p *Peer) LastPingTime() time.Time {
	ts := p.lastPing.Load()
	if ts == 0 {
		return time.Now()
	}
	return time.Unix(0, ts)
}

// RemoteAddress returns the peer's address as string
func (p *Peer) RemoteAddress() string {
	return p.Addr.String()
}

// NodeIDString returns the node ID as hex string
func (p *Peer) NodeIDString() string {
	return p.NodeID.String()
}

// String returns a string representation of the peer
func (p *Peer) String() string {
	dirStr := "outbound"
	if p.Direction == DirInbound {
		dirStr = "inbound"
	}
	return fmt.Sprintf("Peer{%s %s %s height=%d}", p.NodeIDString()[:8], p.Addr, dirStr, p.BestHeight)
}

// MarkBlockKnown marks a block as known to this peer
func (p *Peer) MarkBlockKnown(hash []byte) {
	p.KnownBlocks[hex.EncodeToString(hash)] = true
}

// MarkTxKnown marks a transaction as known to this peer
func (p *Peer) MarkTxKnown(hash []byte) {
	p.KnownTXs[hex.EncodeToString(hash)] = true
}

// IsBlockKnown checks if peer already knows this block
func (p *Peer) IsBlockKnown(hash []byte) bool {
	return p.KnownBlocks[hex.EncodeToString(hash)]
}

// IsTxKnown checks if peer already knows this transaction
func (p *Peer) IsTxKnown(hash []byte) bool {
	return p.KnownTXs[hex.EncodeToString(hash)]
}

// AddBytesSent adds to the bytes sent counter
func (p *Peer) AddBytesSent(n int64) {
	p.bytesSent.Add(n)
}

// AddBytesRecv adds to the bytes received counter
func (p *Peer) AddBytesRecv(n int64) {
	p.bytesRecv.Add(n)
}

// BytesSent returns total bytes sent to this peer
func (p *Peer) BytesSent() int64 {
	return p.bytesSent.Load()
}

// BytesRecv returns total bytes received from this peer
func (p *Peer) BytesRecv() int64 {
	return p.bytesRecv.Load()
}

// Uptime returns the duration since connection
func (p *Peer) Uptime() time.Duration {
	return time.Since(p.connectedAt)
}

// MessageChan returns the channel for incoming messages
func (p *Peer) MessageChan() chan *Message {
	return p.msgChan
}

// =============================================================================
// Connection Management
// =============================================================================

// SetReadDeadline sets the read deadline
func (p *Peer) SetReadDeadline(t time.Time) error {
	p.connMutex.RLock()
	defer p.connMutex.RUnlock()
	if p.conn != nil {
		return p.conn.SetReadDeadline(t)
	}
	return nil
}

// SetWriteDeadline sets the write deadline
func (p *Peer) SetWriteDeadline(t time.Time) error {
	p.connMutex.RLock()
	defer p.connMutex.RUnlock()
	if p.conn != nil {
		return p.conn.SetWriteDeadline(t)
	}
	return nil
}

// Close closes the connection gracefully
func (p *Peer) Close() {
	p.SetState(StateDisconnecting)
	close(p.quit)

	p.connMutex.Lock()
	if p.conn != nil {
		p.conn.Close()
	}
	p.connMutex.Unlock()

	p.wg.Wait()

	// Notify disconnection
	if p.onDisconnect != nil {
		p.onDisconnect(p)
	}
}

// IsConnected checks if the peer is still connected
func (p *Peer) IsConnected() bool {
	p.connMutex.RLock()
	defer p.connMutex.RUnlock()
	if p.conn == nil {
		return false
	}
	// Check if connection is closed
	type connection interface {
		Close() error
	}
	// Simple check - try to read would fail on closed connection
	return true
}

// =============================================================================
// Callbacks
// =============================================================================

// SetBlockHandler sets the callback for new blocks
func (p *Peer) SetBlockHandler(f func(*core.Block)) {
	p.onBlock = f
}

// SetTransactionHandler sets the callback for new transactions
func (p *Peer) SetTransactionHandler(f func(*core.Transaction)) {
	p.onTransaction = f
}

// SetDisconnectHandler sets the callback for disconnection
func (p *Peer) SetDisconnectHandler(f func(*Peer)) {
	p.onDisconnect = f
}

// =============================================================================
// Peer Pool
// =============================================================================

// PeerPool manages connected peers
type PeerPool struct {
	mu       sync.RWMutex
	peers    map[NodeID]*Peer
	addrPeers map[string]*Peer // By address to prevent duplicates
	onAdd    func(*Peer)
	onRemove func(*Peer)
}

// NewPeerPool creates a new peer pool
func NewPeerPool() *PeerPool {
	return &PeerPool{
		peers:     make(map[NodeID]*Peer),
		addrPeers: make(map[string]*Peer),
	}
}

// Add adds a peer to the pool
func (pp *PeerPool) Add(p *Peer) error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	// Check if already connected
	if _, exists := pp.peers[p.NodeID]; exists {
		return fmt.Errorf("peer already connected")
	}

	// Check by address
	if _, exists := pp.addrPeers[p.Addr.String()]; exists {
		return fmt.Errorf("address already connected")
	}

	pp.peers[p.NodeID] = p
	pp.addrPeers[p.Addr.String()] = p

	if pp.onAdd != nil {
		pp.onAdd(p)
	}

	return nil
}

// Remove removes a peer from the pool
func (pp *PeerPool) Remove(nodeID NodeID) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if p, exists := pp.peers[nodeID]; exists {
		delete(pp.peers, nodeID)
		delete(pp.addrPeers, p.Addr.String())

		if pp.onRemove != nil {
			pp.onRemove(p)
		}
	}
}

// Get returns a peer by NodeID
func (pp *PeerPool) Get(nodeID NodeID) (*Peer, bool) {
	pp.mu.RLock()
	defer pp.mu.RUnlock()
	p, ok := pp.peers[nodeID]
	return p, ok
}

// GetByAddr returns a peer by address
func (pp *PeerPool) GetByAddr(addr string) (*Peer, bool) {
	pp.mu.RLock()
	defer pp.mu.RUnlock()
	p, ok := pp.addrPeers[addr]
	return p, ok
}

// List returns all peers
func (pp *PeerPool) List() []*Peer {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	peers := make([]*Peer, 0, len(pp.peers))
	for _, p := range pp.peers {
		peers = append(peers, p)
	}
	return peers
}

// Count returns the number of connected peers
func (pp *PeerPool) Count() int {
	pp.mu.RLock()
	defer pp.mu.RUnlock()
	return len(pp.peers)
}

// ForEach iterates over all peers
func (pp *PeerPool) ForEach(f func(*Peer)) {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	for _, p := range pp.peers {
		f(p)
	}
}

// Broadcast sends a message to all connected peers except optionally excluded
func (pp *PeerPool) Broadcast(msg *Message, exclude *Peer) {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	for _, p := range pp.peers {
		if exclude != nil && p.NodeID == exclude.NodeID {
			continue
		}
		if p.IsReady() {
			select {
			case p.msgChan <- msg:
			default:
				// Channel full, skip
			}
		}
	}
}

// SetOnAdd sets the callback for when a peer is added
func (pp *PeerPool) SetOnAdd(f func(*Peer)) {
	pp.onAdd = f
}

// SetOnRemove sets the callback for when a peer is removed
func (pp *PeerPool) SetOnRemove(f func(*Peer)) {
	pp.onRemove = f
}

// DisconnectAll disconnects all peers
func (pp *PeerPool) DisconnectAll() {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	for _, p := range pp.peers {
		p.Close()
	}
	pp.peers = make(map[NodeID]*Peer)
	pp.addrPeers = make(map[string]*Peer)
}