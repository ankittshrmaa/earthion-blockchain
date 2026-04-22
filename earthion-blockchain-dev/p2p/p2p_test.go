package p2p

import (
	"testing"
	"time"
)

// TestPeerPool tests peer pool creation
func TestPeerPool(t *testing.T) {
	pool := NewPeerPool()
	if pool == nil {
		t.Fatal("NewPeerPool returned nil")
	}

	if pool.Count() != 0 {
		t.Errorf("expected 0 peers, got %d", pool.Count())
	}
}

// TestLocalNode tests local node creation
func TestLocalNode(t *testing.T) {
	node, err := NewLocalNode()
	if err != nil {
		t.Fatalf("NewLocalNode error: %v", err)
	}

	if node == nil {
		t.Fatal("NewLocalNode returned nil")
	}

	if node.ID.String() == "" {
		t.Error("LocalNode ID is empty")
	}
}

// TestMessageSerialization tests message serialization
func TestMessageSerialization(t *testing.T) {
	msg := &Message{
		Type:    MsgVersion,
		Payload: []byte("test payload"),
	}

	data, err := serializeMessage(msg)
	if err != nil {
		t.Fatalf("serializeMessage error: %v", err)
	}
	
	if len(data) == 0 {
		t.Error("Serialize returned empty data")
	}

	msg2, err := deserializeMessage(data)
	if err != nil {
		t.Fatalf("deserializeMessage error: %v", err)
	}

	if msg2.Type != msg.Type {
		t.Errorf("expected type %v, got %v", msg.Type, msg2.Type)
	}

	if string(msg2.Payload) != string(msg.Payload) {
		t.Errorf("expected payload %s, got %s", msg.Payload, msg2.Payload)
	}
}

// TestInvMessage tests inv message
func TestInvMessage(t *testing.T) {
	inv := NewInvMessage()
	if inv == nil {
		t.Fatal("NewInvMessage returned nil")
	}

	testHash := []byte("test hash")
	inv.AddBlock(testHash)
	inv.AddTx(testHash)

	data := inv.Serialize()
	if len(data) == 0 {
		t.Error("Serialize returned empty data")
	}
}

// TestMessageType tests message types
func TestMessageType(t *testing.T) {
	tests := []struct {
		msgType MessageType
		want   string
	}{
		{MsgVerAck, "verack"},
		{MsgGetAddr, "getaddr"},
		{MsgAddr, "addr"},
		{MsgInv, "inv"},
		{MsgGetData, "getdata"},
		{MsgBlock, "block"},
		{MsgGetHeaders, "getheaders"},
		{MsgHeaders, "headers"},
	}

	for _, tt := range tests {
		got := string(tt.msgType)
		if got != tt.want {
			t.Errorf("MessageType(%s) = %q, want %q", tt.msgType, got, tt.want)
		}
	}
}

// TestConfigDefault tests default config
func TestConfigDefault(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.ListenAddr == "" {
		t.Error("ListenAddr is empty")
	}
	
	if cfg.MaxConnections == 0 {
		t.Error("MaxConnections is 0")
	}
	
	if cfg.PingInterval == 0 {
		t.Error("PingInterval is 0")
	}
	
	if cfg.ConnectionTimeout == 0 {
		t.Error("ConnectionTimeout is 0")
	}
}

// TestConfigWithOptions tests config with options
func TestConfigWithOptions(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ListenAddr = ":9999"
	cfg.MaxConnections = 50
	cfg.PingInterval = 15 * time.Second
	
	if cfg.ListenAddr != ":9999" {
		t.Errorf("expected :9999, got %s", cfg.ListenAddr)
	}
	
	if cfg.MaxConnections != 50 {
		t.Errorf("expected 50, got %d", cfg.MaxConnections)
	}
}

// TestPingMessage tests ping message
func TestPingMessage(t *testing.T) {
	msg := &Message{
		Type:    MsgPing,
		Payload: []byte{},
	}

	_, err := serializeMessage(msg)
	if err != nil {
		t.Fatalf("serializeMessage error: %v", err)
	}
}