package p2p

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"earthion/crypto"
)

// =============================================================================
// DoS Protection
// =============================================================================

// DoSConfig holds DoS protection configuration
type DoSConfig struct {
	// Connection rate limiting
	MaxConnectionsPerIP    int           // Max connections from single IP
	ConnectionRateLimit  time.Duration // Min time between connections
	BanDuration         time.Duration // How long to ban

	// Message rate limiting
	MaxMessagesPerSecond int           // Max messages per second
	MessageBurst         int          // Max burst before rate limiting

	// Size limits
	MaxMessageSize int // Max message size in bytes
	MaxHeaders    int // Max headers in getheaders

	// Timeouts
	HandshakeTimeout time.Duration
	ResponseTimeout time.Duration
}

// DefaultDoSConfig returns sensible defaults
func DefaultDoSConfig() DoSConfig {
	return DoSConfig{
		MaxConnectionsPerIP:   4,
		ConnectionRateLimit:   5 * time.Second,
		BanDuration:          24 * time.Hour,
		MaxMessagesPerSecond:  50,
		MessageBurst:         100,
		MaxMessageSize:        10 * 1024 * 1024,
		MaxHeaders:           2000,
		HandshakeTimeout:     30 * time.Second,
		ResponseTimeout:     60 * time.Second,
	}
}

// DoSDetector detects and mitigates DoS attacks
type DoSDetector struct {
	cfg DoSConfig

	// Per-IP tracking
	mu           sync.RWMutex
	connections   map[string]*IPState
	banned       map[string]*BanRecord
	ipCounter    int

	// Global rate limiting
	msgCounter   atomic.Int64
	lastSecond  atomic.Int64
	overLimit   atomic.Bool

	// Callbacks
	onBan       func(string, string) // IP, reason
	onDisconnect func(string)     // IP
}

// IPState tracks state for an IP address
type IPState struct {
	IP             string
	Connections    int
	FirstSeen      time.Time
	LastSeen       time.Time
	Messages       int
	MessageRate    int // Per second
	LastMessage    time.Time
	ResourceExhausted bool
	FailCount      int

	// Rolling counters
	recentMessages int
	windowStart    time.Time
}

// BanRecord represents a ban
type BanRecord struct {
	IP        string
	Reason    string
	BannedAt  time.Time
	ExpiresAt time.Time
}

// NewDoSDetector creates a new DoS detector
func NewDoSDetector(cfg DoSConfig) *DoSDetector {
	return &DoSDetector{
		cfg:         cfg,
		connections: make(map[string]*IPState),
		banned:      make(map[string]*BanRecord),
	}
}

// CheckConnection checks if a connection should be allowed
// Returns: allowed (bool), reason (string)
func (d *DoSDetector) CheckConnection(ip string) (bool, string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	// Check if banned
	if ban, banned := d.banned[ip]; banned {
		if now.Before(ban.ExpiresAt) {
			return false, fmt.Sprintf("banned until %v", ban.ExpiresAt)
		}
		// Ban expired
		delete(d.banned, ip)
	}

	// Get or create IP state
	state, exists := d.connections[ip]
	if !exists {
		state = &IPState{
			IP:        ip,
			FirstSeen:  now,
			LastSeen:  now,
		}
		d.ipCounter++
		d.connections[ip] = state
	}

	// Check connection limit
	if state.Connections >= d.cfg.MaxConnectionsPerIP {
		return false, "too many connections"
	}

	// Check connection rate
	if now.Sub(state.LastSeen) < d.cfg.ConnectionRateLimit {
		state.FailCount++
		if state.FailCount >= 3 {
			d.ban(ip, "connection flood")
		}
		return false, "connection rate limit"
	}

	state.Connections++
	state.LastSeen = now

	return true, ""
}

// ConnectionClosed should be called when a connection closes
func (d *DoSDetector) ConnectionClosed(ip string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if state, ok := d.connections[ip]; ok {
		state.Connections--
		if state.Connections < 0 {
			state.Connections = 0
		}
	}
}

// CheckMessage checks if a message should be allowed
// Returns: allowed (bool), reason (string)
func (d *DoSDetector) CheckMessage(ip string, msgSize int) (bool, string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	// Check message size
	if msgSize > d.cfg.MaxMessageSize {
		return false, "message too large"
	}

	// Check global rate
	if !d.overLimit.Load() {
		d.checkGlobalRate()
	}

	if d.overLimit.Load() {
		return false, "system overloaded"
	}

	// Get IP state
	state, ok := d.connections[ip]
	if !ok {
		return true, ""
	}

	// Update message count for rate calculation
	if now.Sub(state.windowStart) > time.Second {
		state.recentMessages = 0
		state.windowStart = now
	}

	state.recentMessages++
	state.Messages++

	// Check per-IP rate
	if state.recentMessages > d.cfg.MaxMessagesPerSecond {
		state.ResourceExhausted = true
		d.ban(ip, "message flood")
		return false, "message rate limit"
	}

	return true, ""
}

// checkGlobalRate checks global message rate
func (d *DoSDetector) checkGlobalRate() {
	now := time.Now().Unix()
	last := d.lastSecond.Load()

	if now != last {
		count := d.msgCounter.Swap(0)
		d.lastSecond.Store(now)

		// Check if we're over limit
		d.overLimit.Store(count > int64(d.cfg.MaxMessagesPerSecond))
	}
}

// RecordMessage records an incoming message
func (d *DoSDetector) RecordMessage() {
	d.msgCounter.Add(1)
}

// Ban bans an IP address
func (d *DoSDetector) ban(ip string, reason string) {
	now := time.Now()
	expires := now.Add(d.cfg.BanDuration)

	d.banned[ip] = &BanRecord{
		IP:        ip,
		Reason:    reason,
		BannedAt:  now,
		ExpiresAt: expires,
	}

	if d.onBan != nil {
		d.onBan(ip, reason)
	}
}

// Unban unbans an IP address
func (d *DoSDetector) Unban(ip string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.banned, ip)
}

// IsBanned checks if an IP is banned
func (d *DoSDetector) IsBanned(ip string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if ban, ok := d.banned[ip]; ok {
		return time.Now().Before(ban.ExpiresAt)
	}
	return false
}

// SetOnBan sets the ban callback
func (d *DoSDetector) SetOnBan(f func(string, string)) {
	d.onBan = f
}

// SetOnDisconnect sets the disconnect callback
func (d *DoSDetector) SetOnDisconnect(f func(string)) {
	d.onDisconnect = f
}

// Stats returns DoS statistics
func (d *DoSDetector) Stats() (connections, banned int) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.ipCounter, len(d.banned)
}

// Cleanup removes expired bans
func (d *DoSDetector) Cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for ip, ban := range d.banned {
		if ban.ExpiresAt.Before(now) {
			delete(d.banned, ip)
			cleaned++
		}
	}

	// Also clean old connection states
	for ip, state := range d.connections {
		if now.Sub(state.LastSeen) > time.Hour && state.Connections == 0 {
			delete(d.connections, ip)
			d.ipCounter--
		}
	}

	if cleaned > 0 {
		fmt.Printf("[dos] Cleaned up %d expired bans\n", cleaned)
	}
}

// =============================================================================
// Message Validation
// =============================================================================

// MessageValidator validates incoming messages
type MessageValidator struct {
	cfg DoSConfig

	// Track request types
	getDataCount   map[string]int
	getHeadersMap  map[string]int
	invCount      map[string]int

	mu sync.RWMutex
}

// NewMessageValidator creates a new validator
func NewMessageValidator(cfg DoSConfig) *MessageValidator {
	return &MessageValidator{
		cfg:           cfg,
		getDataCount:  make(map[string]int),
		getHeadersMap: make(map[string]int),
		invCount:      make(map[string]int),
	}
}

// ValidateInventory validates an inventory message
func (mv *MessageValidator) ValidateInventory(ip string, inv *InvMessage) error {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	// Check count
	if inv.Count > 50000 {
		return fmt.Errorf("too many inventory items: %d", inv.Count)
	}

	// Track count
	mv.invCount[ip] += inv.Count

	// Rate limit
	if mv.invCount[ip] > 10000 {
		return fmt.Errorf("inventory limit exceeded")
	}

	return nil
}

// ValidateGetHeaders validates a getheaders request
func (mv *MessageValidator) ValidateGetHeaders(ip string, count int) error {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	if count > mv.cfg.MaxHeaders {
		return fmt.Errorf("too many headers requested: %d", count)
	}

	mv.getHeadersMap[ip] += count
	if mv.getHeadersMap[ip] > mv.cfg.MaxHeaders {
		return fmt.Errorf("headers limit exceeded")
	}

	return nil
}

// Reset resets counters for an IP
func (mv *MessageValidator) Reset(ip string) {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	delete(mv.getDataCount, ip)
	delete(mv.getHeadersMap, ip)
	delete(mv.invCount, ip)
}

// =============================================================================
// Connection Filter
// =============================================================================

// ConnectionFilter filters connections based on various criteria
type ConnectionFilter struct {
	cfg DoSConfig

	mu     sync.RWMutex
	scores map[string]*PeerScore
}

// PeerScore tracks peer behavior score
type PeerScore struct {
	IP           string
	Score        int // Higher = better (100 baseline)
	ValidMsgs    int
	InvalidMsgs  int
	DisconnectCount int
	LastActivity time.Time
	Banned       bool
}

// NewConnectionFilter creates a new filter
func NewConnectionFilter(cfg DoSConfig) *ConnectionFilter {
	return &ConnectionFilter{
		cfg:    cfg,
		scores: make(map[string]*PeerScore),
	}
}

// CheckIP filters an IP based on various criteria
func (cf *ConnectionFilter) CheckIP(ip string) (bool, string) {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	if score, ok := cf.scores[ip]; ok && score.Banned {
		return false, "peer banned"
	}

	// Check basics
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false, "invalid IP"
	}

	// Check for private IPs (if not allowing local)
	// Note: In production, could allow localhost for testing
	// if parsed.IsPrivate() || parsed.IsLoopback() { ... }

	return true, ""
}

// RecordValidMessage records a valid message
func (cf *ConnectionFilter) RecordValidMessage(ip string) {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	score, ok := cf.scores[ip]
	if !ok {
		score = &PeerScore{IP: ip, Score: 100}
		cf.scores[ip] = score
	}

	score.ValidMsgs++
	score.Score = min(100, score.Score+1) // Increase score
	score.LastActivity = time.Now()
}

// RecordInvalidMessage records an invalid message
func (cf *ConnectionFilter) RecordInvalidMessage(ip string) {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	score, ok := cf.scores[ip]
	if !ok {
		score = &PeerScore{IP: ip, Score: 100}
		cf.scores[ip] = score
	}

	score.InvalidMsgs++
	score.Score = max(-100, score.Score-10) // Decrease score

	if score.Score < -50 {
		score.Banned = true
	}
}

// RecordDisconnect records a disconnect
func (cf *ConnectionFilter) RecordDisconnect(ip string) {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	score, ok := cf.scores[ip]
	if !ok {
		return
	}

	score.DisconnectCount++
// Update score on disconnect (slightly negative)
	score.Score = max(-100, score.Score-5)
}

// Score returns the score for an IP
func (cf *ConnectionFilter) Score(ip string) int {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	if score, ok := cf.scores[ip]; ok {
		return score.Score
	}
	return 100 // Default score
}

// GetBanned returns all banned IPs
func (cf *ConnectionFilter) GetBanned() []string {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	var banned []string
	for ip, score := range cf.scores {
		if score.Banned {
			banned = append(banned, ip)
		}
	}
	return banned
}

// =============================================================================
// Encryption Helper
// =============================================================================

// EncryptedConnection wraps a connection with encryption
type EncryptedConnection struct {
	conn       net.Conn
	cipher     *CryptoCipher
	nonce      uint64
	writeNonce uint64
}

// CryptoCipher is a simple cipher for encryption
// Note: In production, use proper AES-GCM or ChaCha20-Poly1305
type CryptoCipher struct {
	key [32]byte
}

// NewCryptoCipher creates a new cipher
func NewCryptoCipher(key []byte) *CryptoCipher {
	var c CryptoCipher
	copy(c.key[:], crypto.Hash(key)[:32])
	return &c
}

// Encrypt encrypts data
func (c *CryptoCipher) Encrypt(plaintext []byte, nonce uint64) []byte {
	// Simplified - use proper AEAD in production
	key := crypto.Hash(append(c.key[:], []byte(fmt.Sprint(nonce))...))
	ciphertext := make([]byte, len(plaintext))
	for i, b := range plaintext {
		ciphertext[i] = b ^ key[i%len(key)]
	}
	return ciphertext
}

// Decrypts data
func (c *CryptoCipher) Decrypt(ciphertext []byte, nonce uint64) []byte {
	// XOR is symmetric
	return c.Decrypt(ciphertext, nonce)
}

// =============================================================================
// Utility Functions
// =============================================================================

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}