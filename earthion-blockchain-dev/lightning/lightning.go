package lightning

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"earthion/crypto"
)

// =============================================================================
// Lightning Network Implementation
// =============================================================================

const (
	// Channel constants
	MaxPendingHTLCs        = 483
	MinChannelCapacity     = 10000      // Minimum satoshis
	MaxChannelCapacity     = 16777215   // ~167 BTC
	CSVDelay              = 144        // 6 blocks for CSV
	MinDustLimit          = 546        // Dust limit
	
	// HTLC constants
	HTLCTimeoutBlocks     = 40         // 40 blocks (~40 min with 1 min blocks)
	ExpiryTimeout         = 2 * time.Hour // For in-memory HTLCs
)

// =============================================================================
// HTLC (Hash Time Locked Contract)
// =============================================================================

// HTLC represents a Hash Time Locked Contract
type HTLC struct {
	ID                string
	Amount            int64         // Amount in satoshis
	PaymentHash       []byte        // SHA256 of preimage
	Preimage          []byte        // Revealed when claimed (empty if not yet revealed)
	ExpiryBlock       uint32        // Block height when HTLC expires
	Direction         HTLCDirection // Incoming or outgoing
	State             HTLCState
	CreatedAt         time.Time
	ClaimedAt         time.Time
}

// HTLCDirection indicates direction of HTLC
type HTLCDirection int

const (
	HTLCDirectionIncoming HTLCDirection = iota
	HTLCDirectionOutgoing
)

// HTLCState represents the state of an HTLC
type HTLCState int

const (
	HTLCStateOffered HTLCState = iota
	HTLCStateFulfilled
	HTLCStateFailed
	HTLCStateExpired
)

// NewHTLC creates a new HTLC
func NewHTLC(amount int64, paymentHash []byte, expiryBlock uint32, direction HTLCDirection) *HTLC {
	return &HTLC{
		ID:              generateHTLCID(paymentHash),
		Amount:          amount,
		PaymentHash:    paymentHash,
		ExpiryBlock:    expiryBlock,
		Direction:      direction,
		State:          HTLCStateOffered,
		CreatedAt:      time.Now(),
	}
}

// IsExpired checks if HTLC has expired
func (h *HTLC) IsExpired(currentBlock uint32) bool {
	return currentBlock >= h.ExpiryBlock && h.State == HTLCStateOffered
}

// CanClaim checks if HTLC can be claimed (has preimage)
func (h *HTLC) CanClaim() bool {
	return h.State == HTLCStateOffered && len(h.Preimage) > 0
}

// Claim claims the HTLC with the preimage
func (h *HTLC) Claim(preimage []byte) bool {
	// Verify preimage matches hash
	if !verifyPreimage(preimage, h.PaymentHash) {
		return false
	}
	
	h.Preimage = preimage
	h.State = HTLCStateFulfilled
	h.ClaimedAt = time.Now()
	return true
}

// =============================================================================
// Payment Channel
// =============================================================================

// PaymentChannel represents a Lightning payment channel
type PaymentChannel struct {
	ChannelID        []byte
	FundingTXID      []byte
	FundingOutputIdx uint32
	
	// Participants
	LocalNode        []byte  // Local public key
	RemoteNode       []byte  // Remote public key
	
	// Channel state
	Capacity         int64   // Total capacity in satoshis
	LocalBalance     int64   // Local's current balance
	RemoteBalance    int64   // Remote's current balance
	LocalCommitFee   int64   // Local's commitment fee
	RemoteCommitFee  int64   // Remote's commitment fee
	
	// HTLCs
	PendingHTLCs     []*HTLC
	CompletedHTLCs  []*HTLC
	
	// Revocation
	LocalRevocation  []byte  // Current local revocation seed
	RemoteRevocation []byte  // Current remote revocation seed
	
	// State
	IsOpen           bool
	IsFunding        bool
	FundingBlock     uint32
	ClosingBlock     uint32
	
	// Timing
	CreatedAt        time.Time
	UpdatedAt        time.Time
	
	// Mutual close
	MutualCloseSig   []byte
	
	mu               sync.RWMutex
}

// NewPaymentChannel creates a new payment channel
func NewPaymentChannel(localNode, remoteNode []byte, capacity int64) *PaymentChannel {
	return &PaymentChannel{
		ChannelID:       generateChannelID(localNode, remoteNode),
		LocalNode:       localNode,
		RemoteNode:      remoteNode,
		Capacity:        capacity,
		LocalBalance:    capacity / 2,
		RemoteBalance:   capacity / 2,
		PendingHTLCs:    make([]*HTLC, 0),
		CompletedHTLCs:  make([]*HTLC, 0),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		IsOpen:          false,
		IsFunding:      true,
	}
}

// AddHTLC adds an HTLC to the channel
func (c *PaymentChannel) AddHTLC(h *HTLC) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if len(c.PendingHTLCs) >= MaxPendingHTLCs {
		return fmt.Errorf("max pending HTLCs reached")
	}
	
	if h.Amount > c.LocalBalance {
		return fmt.Errorf("insufficient local balance")
	}
	
	c.PendingHTLCs = append(c.PendingHTLCs, h)
	c.LocalBalance -= h.Amount
	c.UpdatedAt = time.Now()
	
	return nil
}

// FulFillHTLC fulfills an incoming HTLC
func (c *PaymentChannel) FulFillHTLC(paymentHash []byte, preimage []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for _, h := range c.PendingHTLCs {
		if bytesEqual(h.PaymentHash, paymentHash) {
			if !h.Claim(preimage) {
				return fmt.Errorf("invalid preimage")
			}
			
			// Move to completed
			c.RemoteBalance += h.Amount
			c.removeHTLC(h.ID)
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	
	return fmt.Errorf("HTLC not found")
}

// removeHTLC removes an HTLC by ID
func (c *PaymentChannel) removeHTLC(id string) {
	newList := make([]*HTLC, 0)
	for _, h := range c.PendingHTLCs {
		if h.ID != id {
			newList = append(newList, h)
		}
	}
	c.PendingHTLCs = newList
}

// GetLocalBalance returns local balance
func (c *PaymentChannel) GetLocalBalance() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LocalBalance
}

// GetRemoteBalance returns remote balance
func (c *PaymentChannel) GetRemoteBalance() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.RemoteBalance
}

// =============================================================================
// Channel State Commitment
// =============================================================================

// CommitmentTransaction represents a commitment transaction
type CommitmentTransaction struct {
	ChannelID    []byte
	Sequence     uint64
	Outputs      []CommitmentOutput
	HTLCs        []*HTLC
	RevocationHash []byte
	IsLocal      bool
}

// CommitmentOutput represents an output in commitment tx
type CommitmentOutput struct {
	Amount    int64
	PubKey    []byte
	CSVDelay  uint32  // Relative timeout (only for local)
}

// NewCommitmentTransaction creates a new commitment transaction
func NewCommitmentTransaction(channelID []byte, sequence uint64, isLocal bool) *CommitmentTransaction {
	return &CommitmentTransaction{
		ChannelID:    channelID,
		Sequence:     sequence,
		Outputs:      make([]CommitmentOutput, 0),
		HTLCs:        make([]*HTLC, 0),
		IsLocal:      isLocal,
	}
}

// Build builds the commitment transaction outputs
func (ct *CommitmentTransaction) Build(localBalance, remoteBalance int64, local, remote []byte, csvDelay uint32) {
	// To local output
	if localBalance > MinDustLimit {
		ct.Outputs = append(ct.Outputs, CommitmentOutput{
			Amount:   localBalance,
			PubKey:   local,
			CSVDelay: csvDelay,
		})
	}
	
	// To remote output (no delay)
	if remoteBalance > MinDustLimit {
		ct.Outputs = append(ct.Outputs, CommitmentOutput{
			Amount:   remoteBalance,
			PubKey:   remote,
			CSVDelay: 0,
		})
	}
}

// =============================================================================
// Lightning Node
// =============================================================================

// Node represents a Lightning Network node
type Node struct {
	NodeID      []byte
	PrivateKey  []byte
	PublicKey   []byte
	
	// Channels
	Channels    map[string]*PaymentChannel
	
	// Network
	Peers       map[string]*PeerConnection
	
	// Router
	Router      *Router
	
	// Graph
	ChannelGraph *ChannelGraph
	
	mu          sync.RWMutex
}

// NewNode creates a new Lightning node
func NewNode(privateKey []byte) (*Node, error) {
	pubKey := getPublicKeyFromPriv(privateKey)
	
	return &Node{
		NodeID:       pubKey,
		PrivateKey:  privateKey,
		PublicKey:   pubKey,
		Channels:    make(map[string]*PaymentChannel),
		Peers:       make(map[string]*PeerConnection),
		Router:      NewRouter(),
		ChannelGraph: NewChannelGraph(),
	}, nil
}

// OpenChannel opens a payment channel with a peer
func (n *Node) OpenChannel(peerPubKey []byte, capacity int64) (*PaymentChannel, error) {
	if capacity < MinChannelCapacity || capacity > MaxChannelCapacity {
		return nil, fmt.Errorf("invalid capacity")
	}
	
	channel := NewPaymentChannel(n.PublicKey, peerPubKey, capacity)
	n.mu.Lock()
	n.Channels[string(channel.ChannelID)] = channel
	n.mu.Unlock()
	
	return channel, nil
}

// ReceivePayment receives an incoming HTLC
func (n *Node) ReceivePayment(channelID []byte, amount int64, paymentHash []byte, expiryBlock uint32) error {
	n.mu.RLock()
	channel, ok := n.Channels[string(channelID)]
	n.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("channel not found")
	}
	
	htlc := NewHTLC(amount, paymentHash, expiryBlock, HTLCDirectionIncoming)
	return channel.AddHTLC(htlc)
}

// SendPayment sends a payment through a channel
func (n *Node) SendPayment(channelID []byte, amount int64, paymentHash []byte, expiryBlock uint32) error {
	n.mu.RLock()
	channel, ok := n.Channels[string(channelID)]
	n.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("channel not found")
	}
	
	htlc := NewHTLC(amount, paymentHash, expiryBlock, HTLCDirectionOutgoing)
	return channel.AddHTLC(htlc)
}

// =============================================================================
// Peer Connection
// =============================================================================

// PeerConnection represents a connection to another Lightning node
type PeerConnection struct {
	NodeID      []byte
	Addr        string
	IsConnected bool
	LastPing    time.Time
	Channels    [][]byte
}

// =============================================================================
// Router
// =============================================================================

// Router handles payment routing
type Router struct {
	routes    map[string][]*Route
	mu        sync.RWMutex
}

// Route represents a payment route
type Route struct {
	Channels   []RouteHop
	TotalFee    int64
	TotalAmount int64
	Expiry     time.Time
}

// RouteHop represents a hop in a route
type RouteHop struct {
	ChannelID   []byte
	NodeID      []byte
	Amount      int64
	Fee         int64
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		routes: make(map[string][]*Route),
	}
}

// FindRoute finds a route to destination
func (r *Router) FindRoute(source, destination []byte, amount int64) (*Route, error) {
	// Simplified: In production would use Dijkstra's algorithm
	// with channel capacity and fees
	
	// For now, return error (needs graph data)
	return nil, fmt.Errorf("no route found - graph not initialized")
}

// AddRoute adds a known route
func (r *Router) AddRoute(paymentHash string, route *Route) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[paymentHash] = append(r.routes[paymentHash], route)
}

// =============================================================================
// Channel Graph
// =============================================================================

// ChannelGraph represents the Lightning Network topology
type ChannelGraph struct {
	Nodes     map[string]*GraphNode
	Channels  map[string]*GraphChannel
	mu        sync.RWMutex
}

// GraphNode represents a node in the graph
type GraphNode struct {
	NodeID    []byte
	Alias     string
	Features  uint64
	LastUpdate time.Time
}

// GraphChannel represents a channel in the graph
type GraphChannel struct {
	ChannelID    []byte
	Node1        []byte
	Node2        []byte
	Capacity     int64
	FeeBase      int64
	FeeRate      int64
	LastUpdate   time.Time
}

// NewChannelGraph creates a new channel graph
func NewChannelGraph() *ChannelGraph {
	return &ChannelGraph{
		Nodes:    make(map[string]*GraphNode),
		Channels: make(map[string]*GraphChannel),
	}
}

// AddNode adds a node to the graph
func (g *ChannelGraph) AddNode(nodeID []byte, alias string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Nodes[string(nodeID)] = &GraphNode{
		NodeID:    nodeID,
		Alias:     alias,
		LastUpdate: time.Now(),
	}
}

// AddChannel adds a channel to the graph
func (g *ChannelGraph) AddChannel(channelID []byte, node1, node2 []byte, capacity int64, feeBase, feeRate int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Channels[string(channelID)] = &GraphChannel{
		ChannelID:  channelID,
		Node1:      node1,
		Node2:      node2,
		Capacity:  capacity,
		FeeBase:   feeBase,
		FeeRate:   feeRate,
		LastUpdate: time.Now(),
	}
}

// =============================================================================
// Onions (Sphinx Routing)
// =============================================================================

// OnionPacket represents a Sphinx onion packet
type OnionPacket struct {
	Version     byte
	PublicKey   []byte  // EPHEMERAL KEY
	EncrytedData []byte // Encrypted hop data
	HMAC        []byte  // Integrity check
}

// OnionHop represents a single hop in the route
type OnionHop struct {
	NextPubKey   []byte
	Amount       int64
	Expiry       uint32
	ShortChannelID []byte
	FEATURES     uint16
}

// =============================================================================
// Utility Functions
// =============================================================================

func generateHTLCID(paymentHash []byte) string {
	h := sha256.Sum256(paymentHash)
	return fmt.Sprintf("%x", h[:8])
}

func generateChannelID(local, remote []byte) []byte {
	combined := make([]byte, 0, len(local)+len(remote))
	combined = append(combined, local...)
	combined = append(combined, remote...)
	return crypto.Hash(combined)
}

func verifyPreimage(preimage, hash []byte) bool {
	h := sha256.Sum256(preimage)
	return bytesEqual(h[:], hash)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func getPublicKeyFromPriv(priv []byte) []byte {
	// Uses existing wallet logic
	return crypto.Hash(priv)[:33]
}

// =============================================================================
// HTLC Failure Codes (BOLT #4)
// =============================================================================

const (
	HTLCFailureCodePerm            = 0x8001
	HTLCFailureCodeTempNode        = 0x8002
	HTLCFailureCodeRequired        = 0x8003
	HTLCFailureCodeAmtTooLow       = 0x8004
	HTLCFailureCodeFeeInsufficient = 0x8005
	HTLCFailureCodeIncorrect       = 0x8006
	HTLCFailureCodeExpiryTooSoon   = 0x8007
	HTLCFailureCodeUnknownNextPeer = 0x8008
)

// HTLCFailure represents an HTLC failure
type HTLCFailure struct {
	Code       uint16
	OnionRoute []byte  // Encrypted failure details
}

// =============================================================================
// Invoice (BOLT #11)
// =============================================================================

// Invoice represents a Lightning invoice
type Invoice struct {
	PaymentHash   []byte
	Amount        int64
	Description   string
	Expiry        time.Time
	CreatedAt     time.Time
	NodePubKey    []byte
}

// NewInvoice creates a new invoice
func NewInvoice(paymentHash []byte, amount int64, description string, expiryHours int, nodePubKey []byte) *Invoice {
	return &Invoice{
		PaymentHash: paymentHash,
		Amount:      amount,
		Description: description,
		Expiry:      time.Now().Add(time.Duration(expiryHours) * time.Hour),
		CreatedAt:   time.Now(),
		NodePubKey:  nodePubKey,
	}
}

// IsExpired checks if invoice has expired
func (i *Invoice) IsExpired() bool {
	return time.Now().After(i.Expiry)
}

// Encode encodes the invoice as a bech32 string (simplified)
func (i *Invoice) Encode() string {
	data := make([]byte, 0)
	data = append(data, 0x00) // version
	data = append(data, i.PaymentHash...)
	binary.BigEndian.PutUint64(data[len(data):], uint64(i.Amount))
	return fmt.Sprintf("lni1%x", data)
}