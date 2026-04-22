package p2p

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// =============================================================================
// Peer Discovery
// =============================================================================

// BootstrapNodes are hardcoded seed nodes for initial discovery
var BootstrapNodes = []string{
	"127.0.0.1:8333",
	"127.0.0.1:8334",
	// Add your seed nodes here
}

// DiscoveryService handles peer discovery
type DiscoveryService struct {
	localNode *LocalNode
	peers     map[NodeID]*PeerInfo // Known peers
	peersMu   sync.RWMutex

	// Bootstrap
	bootstrapAddrs []string

	// Callbacks
	onNewPeers func([]*PeerInfo) // Called when new peers found
}

// PeerInfo holds information about a known peer
type PeerInfo struct {
	ID           NodeID
	Addr         string
	Services     uint64
	BestHeight   int // Best known block height
	LastSeen     time.Time
	LastAttempt time.Time
	IsActive     bool
	FailCount    int // Consecutive connection failures
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(local *LocalNode, bootstrap []string) *DiscoveryService {
	// Use default bootstrap if none provided
	if len(bootstrap) == 0 {
		bootstrap = BootstrapNodes
	}

	return &DiscoveryService{
		localNode:      local,
		peers:          make(map[NodeID]*PeerInfo),
		bootstrapAddrs: bootstrap,
	}
}

// AddKnownPeer adds a peer to the known peers list
func (ds *DiscoveryService) AddKnownPeer(info *PeerInfo) {
	ds.peersMu.Lock()
	defer ds.peersMu.Unlock()

	info.LastSeen = time.Now()
	ds.peers[info.ID] = info
}

// RemovePeer removes a peer from the known list
func (ds *DiscoveryService) RemovePeer(nodeID NodeID) {
	ds.peersMu.Lock()
	defer ds.peersMu.Unlock()
	delete(ds.peers, nodeID)
}

// GetPeer returns peer info by ID
func (ds *DiscoveryService) GetPeer(nodeID NodeID) (*PeerInfo, bool) {
	ds.peersMu.RLock()
	defer ds.peersMu.RUnlock()

	info, ok := ds.peers[nodeID]
	return info, ok
}

// SelectPeers selects peers for connection
// Prioritizes: active peers, higher block height, diverse addresses
func (ds *DiscoveryService) SelectPeers(count int, connected map[NodeID]bool) []*PeerInfo {
	ds.peersMu.RLock()
	defer ds.peersMu.RUnlock()

	var candidates []*PeerInfo
	now := time.Now()

	// Filter and score peers
	for _, info := range ds.peers {
		// Skip already connected
		if connected[info.ID] {
			continue
		}

		// Skip recently attempted
		if now.Sub(info.LastAttempt) < 5*time.Minute {
			continue
		}

		// Skip too many failures
		if info.FailCount > 3 {
			continue
		}

		candidates = append(candidates, info)
	}

	// Shuffle for randomness
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	// Sort by score (height - failures)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if ds.scorePeer(candidates[j]) > ds.scorePeer(candidates[i]) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	if len(candidates) > count {
		candidates = candidates[:count]
	}

	return candidates
}

// scorePeer calculates a peer score for selection priority
func (ds *DiscoveryService) scorePeer(info *PeerInfo) int {
	score := info.BestHeight*10 - info.FailCount*100
	if info.IsActive {
		score += 50
	}
	return score
}

// GetBootstrapAddresses returns bootstrap addresses
func (ds *DiscoveryService) GetBootstrapAddresses() []string {
	return ds.bootstrapAddrs
}

// =============================================================================
// Ping/Pong for Liveness Detection
// =============================================================================

const (
	PingInterval = 2 * time.Minute
	PingTimeout  = 30 * time.Second
)

// PingPong handles ping/pong message exchange
type PingPong struct {
	localNode *LocalNode
	peers     map[NodeID]*PingState
	peersMu   sync.RWMutex

	// Config
	interval time.Duration
	timeout  time.Duration
}

// PingState tracks ping state for a peer
type PingState struct {
	LastPing    time.Time
	LastPong    time.Time
	LastLatency time.Duration

	// Failures
	consecutiveFails int
	lastFailTime     time.Time
}

// NewPingPong creates a new ping/pong handler
func NewPingPong(local *LocalNode) *PingPong {
	return &PingPong{
		localNode: local,
		peers:     make(map[NodeID]*PingState),
		interval:  PingInterval,
		timeout:  PingTimeout,
	}
}

// HandlePing processes an incoming ping
func (pp *PingPong) HandlePing(p *Peer, msg *Message) (*Message, error) {
	if msg.Payload == nil || len(msg.Payload) < 4 {
		return nil, fmt.Errorf("invalid ping payload")
	}

	// Create pong response with same nonce
	nonce := msg.Payload[:8]

	// Update peer state
	peerState := pp.getOrCreatePeerState(p.NodeID)
	peerState.LastPing = time.Now()

	// Respond with pong
	return &Message{
		Type:    MsgPong,
		Payload: nonce,
	}, nil
}

// HandlePong processes an incoming pong
func (pp *PingPong) HandlePong(p *Peer, msg *Message) error {
	peerState, ok := pp.peers[p.NodeID]
	if !ok {
		return fmt.Errorf("unknown peer")
	}

	if peerState.LastPing.IsZero() {
		return fmt.Errorf("unsolicited pong")
	}

	// Calculate latency
	peerState.LastPong = time.Now()
	peerState.LastLatency = peerState.LastPong.Sub(peerState.LastPing)

	// Reset fail count on successful pong
	peerState.consecutiveFails = 0

	return nil
}

// RecordFailure records a ping failure
func (pp *PingPong) RecordFailure(nodeID NodeID) {
	peerState, ok := pp.peers[nodeID]
	if !ok {
		return
	}
	peerState.consecutiveFails++
	peerState.lastFailTime = time.Now()
}

// Latency returns the last measured latency for a peer
func (pp *PingPong) Latency(nodeID NodeID) time.Duration {
	peerState, ok := pp.peers[nodeID]
	if !ok {
		return 0
	}
	return peerState.LastLatency
}

// IsDead checks if a peer is considered dead (too many failures)
func (pp *PingPong) IsDead(nodeID NodeID) bool {
	peerState, ok := pp.peers[nodeID]
	if !ok {
		return false
	}
	return peerState.consecutiveFails >= 3
}

func (pp *PingPong) getOrCreatePeerState(nodeID NodeID) *PingState {
	pp.peersMu.Lock()
	defer pp.peersMu.Unlock()

	state, ok := pp.peers[nodeID]
	if !ok {
		state = &PingState{}
		pp.peers[nodeID] = state
	}
	return state
}

// =============================================================================
// Peer Exchange (Addr Message)
// =============================================================================

const (
	MaxAddrPerMessage = 100
	MaxKnownPeers    = 1000
	AddrInterval     = 24 * time.Hour
)

// AddressManager manages known peer addresses
type AddressManager struct {
	localNode *LocalNode

	mu          sync.RWMutex
	addresses   map[string]*AddrRecord // key = "ip:port"
	bucketDays  int                 // Number of address "buckets" by day
	buckets     []map[string]*AddrRecord

	onNewAddress func(*PeerInfo)
}

// AddrRecord represents a known address
type AddrRecord struct {
	Addr        string
	Services    uint64
	BestHeight  int
	Source     NodeID // Which peer told us about this
	Timestamp  time.Time
	ExpiresAt  time.Time
	IsVerified bool
}

// NewAddressManager creates a new address manager
func NewAddressManager(local *LocalNode) *AddressManager {
	am := &AddressManager{
		localNode:   local,
		addresses:   make(map[string]*AddrRecord),
		bucketDays:  7,
	}
	am.buckets = make([]map[string]*AddrRecord, am.bucketDays)
	for i := range am.buckets {
		am.buckets[i] = make(map[string]*AddrRecord)
	}
	return am
}

// AddAddress adds a new address to the manager
func (am *AddressManager) AddAddress(info *PeerInfo, source NodeID) {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	record := &AddrRecord{
		Addr:       info.Addr,
		Services:   info.Services,
		BestHeight: info.BestHeight,
		Source:    source,
		Timestamp:  now,
		ExpiresAt:  now.Add(AddrInterval),
		IsVerified: false,
	}

	am.addresses[info.Addr] = record

	// Add to today's bucket
	bucketIdx := 0
	am.buckets[bucketIdx][info.Addr] = record

	// Callback
	if am.onNewAddress != nil {
		am.onNewAddress(info)
	}
}

// GetAddresses returns a random selection of addresses
func (am *AddressManager) GetAddresses(count int) []*PeerInfo {
	am.mu.RLock()
	defer am.mu.RUnlock()

	now := time.Now()
	var valid []*PeerInfo

	for _, record := range am.addresses {
		if record.ExpiresAt.After(now) {
			valid = append(valid, &PeerInfo{
				ID:         record.Source, // Use source as ID for now
				Addr:       record.Addr,
				Services:   record.Services,
				BestHeight: record.BestHeight,
				LastSeen:   record.Timestamp,
				IsActive:  true,
			})
		}
	}

	// Shuffle
	rand.Shuffle(len(valid), func(i, j int) {
		valid[i], valid[j] = valid[j], valid[i]
	})

	if len(valid) > count {
		valid = valid[:count]
	}

	return valid
}

// Cleanup removes expired addresses
func (am *AddressManager) Cleanup() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	removed := 0

	for addr, record := range am.addresses {
		if record.ExpiresAt.Before(now) {
			delete(am.addresses, addr)
			removed++
		}
	}

	// Clear old buckets
	for i := am.bucketDays - 1; i > 0; i-- {
		am.buckets[i] = am.buckets[i-1]
	}
	am.buckets[0] = make(map[string]*AddrRecord)

	// Compact buckets
	for _, bucket := range am.buckets {
		for addr, record := range bucket {
			if record.ExpiresAt.Before(now) {
				delete(bucket, addr)
			}
		}
	}

	fmt.Printf("[discovery] Cleaned up %d expired addresses\n", removed)
}

// AddLocalAddress adds our own address (for advertised addresses)
func (am *AddressManager) AddLocalAddress(addr string, services uint64, height int) {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	record := &AddrRecord{
		Addr:        addr,
		Services:    services,
		BestHeight:  height,
		Source:     am.localNode.ID,
		Timestamp:  now,
		ExpiresAt: now.Add(AddrInterval * 30), // Stay longer in our own list
		IsVerified: true,
	}
	am.addresses[addr] = record
}

// GetAllAddresses returns all known addresses
func (am *AddressManager) GetAllAddresses() []string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	addrs := make([]string, 0, len(am.addresses))
	for addr := range am.addresses {
		addrs = append(addrs, addr)
	}
	return addrs
}

// =============================================================================
// Utility Functions
// =============================================================================

// HostToAddr converts a host string to a peer address
// Handles "host:port" or "host" (with default port)
func HostToAddr(host string, defaultPort int) string {
	// Already has port
	for _, c := range host {
		if c == ':' {
			return host
		}
	}
	return fmt.Sprintf("%s:%d", host, defaultPort)
}

// ParsePeerAddress parses "nodeID@host:port" format
func ParsePeerAddress(s string) (NodeID, string, error) {
	// Simple format: host:port (default)
	// Extended format: nodeid@host:port

	var host string
	var nodeID NodeID

	n, err := hex.Decode(nodeID[:], []byte(s[:8]))
	if err == nil && n == 8 {
		// Has nodeID prefix
		parts := splitAt(s, '@')
		if len(parts) == 2 {
			copy(nodeID[:], parts[0])
			host = parts[1]
		}
	}

	if host == "" {
		host = s
	}

	return nodeID, host, nil
}

func splitAt(s string, delim byte) []string {
	var result []string
	start := 0
	for i, c := range s {
		if byte(c) == delim {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// EnsureASCII ensures address contains only ASCII characters
func EnsureASCII(addr string) bool {
	for _, c := range addr {
		if c > 127 {
			return false
		}
	}
	return true
}