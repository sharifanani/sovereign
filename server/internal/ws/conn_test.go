package ws

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"

	"github.com/sovereign-im/sovereign/server/internal/protocol"
)

// setupTestServer creates a test WebSocket server and returns the URL and cleanup function.
func setupTestServer(t *testing.T, maxMessageSize int) (string, func()) {
	t.Helper()

	hub := NewHub()
	go hub.Run()

	handler := UpgradeHandler(hub, maxMessageSize)
	server := httptest.NewServer(handler)

	url := "ws" + strings.TrimPrefix(server.URL, "http")

	cleanup := func() {
		server.Close()
		hub.Stop()
	}

	return url, cleanup
}

// dialTestServer connects to the test server with the sovereign.v1 subprotocol.
func dialTestServer(t *testing.T, ctx context.Context, url string) *websocket.Conn {
	t.Helper()

	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		Subprotocols: []string{"sovereign.v1"},
	})
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}

	return conn
}

// sendEnvelope marshals and sends a protobuf envelope over the WebSocket.
func sendEnvelope(t *testing.T, ctx context.Context, conn *websocket.Conn, env *protocol.Envelope) {
	t.Helper()

	data, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("Failed to marshal envelope: %v", err)
	}

	if err := conn.Write(ctx, websocket.MessageBinary, data); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
}

// readEnvelope reads and unmarshals a protobuf envelope from the WebSocket.
func readEnvelope(t *testing.T, ctx context.Context, conn *websocket.Conn) *protocol.Envelope {
	t.Helper()

	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if typ != websocket.MessageBinary {
		t.Fatalf("Response message type = %v, want Binary", typ)
	}

	var env protocol.Envelope
	if err := proto.Unmarshal(data, &env); err != nil {
		t.Fatalf("Failed to unmarshal response envelope: %v", err)
	}

	return &env
}

func TestEcho(t *testing.T) {
	tests := []struct {
		name    string
		msgType protocol.MessageType
		reqID   string
		payload []byte
	}{
		{
			name:    "echo MESSAGE_SEND",
			msgType: protocol.MessageType_MESSAGE_SEND,
			reqID:   "req-1",
			payload: []byte("hello world"),
		},
		{
			name:    "echo AUTH_REQUEST",
			msgType: protocol.MessageType_AUTH_REQUEST,
			reqID:   "req-2",
			payload: []byte("auth-data"),
		},
		{
			name:    "echo with empty payload",
			msgType: protocol.MessageType_MESSAGE_SEND,
			reqID:   "req-3",
		},
		{
			name:    "echo with empty request ID",
			msgType: protocol.MessageType_MESSAGE_SEND,
			payload: []byte("no-request-id"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, cleanup := setupTestServer(t, 65536)
			defer cleanup()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn := dialTestServer(t, ctx, url)
			defer conn.Close(websocket.StatusNormalClosure, "")

			env := &protocol.Envelope{
				Type:      tt.msgType,
				RequestId: tt.reqID,
				Payload:   tt.payload,
			}
			sendEnvelope(t, ctx, conn, env)

			resp := readEnvelope(t, ctx, conn)

			if resp.Type != tt.msgType {
				t.Errorf("Type = %v, want %v", resp.Type, tt.msgType)
			}
			if resp.RequestId != tt.reqID {
				t.Errorf("RequestId = %q, want %q", resp.RequestId, tt.reqID)
			}
			if string(resp.Payload) != string(tt.payload) {
				t.Errorf("Payload = %q, want %q", resp.Payload, tt.payload)
			}
		})
	}
}

func TestPingPong(t *testing.T) {
	tests := []struct {
		name      string
		timestamp int64
		reqID     string
	}{
		{
			name:      "current timestamp",
			timestamp: time.Now().UnixMicro(),
			reqID:     "ping-1",
		},
		{
			name:      "zero timestamp",
			timestamp: 0,
			reqID:     "ping-2",
		},
		{
			name:      "large timestamp",
			timestamp: 1700000000000000,
			reqID:     "ping-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, cleanup := setupTestServer(t, 65536)
			defer cleanup()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn := dialTestServer(t, ctx, url)
			defer conn.Close(websocket.StatusNormalClosure, "")

			ping := &protocol.Ping{Timestamp: tt.timestamp}
			pingPayload, err := proto.Marshal(ping)
			if err != nil {
				t.Fatalf("Failed to marshal ping: %v", err)
			}

			env := &protocol.Envelope{
				Type:      protocol.MessageType_PING,
				RequestId: tt.reqID,
				Payload:   pingPayload,
			}
			sendEnvelope(t, ctx, conn, env)

			resp := readEnvelope(t, ctx, conn)

			if resp.Type != protocol.MessageType_PONG {
				t.Errorf("Type = %v, want PONG", resp.Type)
			}
			if resp.RequestId != tt.reqID {
				t.Errorf("RequestId = %q, want %q", resp.RequestId, tt.reqID)
			}

			var pong protocol.Pong
			if err := proto.Unmarshal(resp.Payload, &pong); err != nil {
				t.Fatalf("Failed to unmarshal pong payload: %v", err)
			}

			if pong.Timestamp != tt.timestamp {
				t.Errorf("Pong.Timestamp = %d, want %d", pong.Timestamp, tt.timestamp)
			}
		})
	}
}

func TestInvalidMessage(t *testing.T) {
	url, cleanup := setupTestServer(t, 65536)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send invalid protobuf bytes (truncated varint)
	if err := conn.Write(ctx, websocket.MessageBinary, []byte{0x80}); err != nil {
		t.Fatalf("Failed to write invalid data: %v", err)
	}

	// Should receive an ERROR envelope
	resp := readEnvelope(t, ctx, conn)

	if resp.Type != protocol.MessageType_ERROR {
		t.Fatalf("Type = %v, want ERROR", resp.Type)
	}

	var errMsg protocol.Error
	if err := proto.Unmarshal(resp.Payload, &errMsg); err != nil {
		t.Fatalf("Failed to unmarshal error payload: %v", err)
	}

	if errMsg.Code != 3001 {
		t.Errorf("Error.Code = %d, want 3001", errMsg.Code)
	}
	if errMsg.Message == "" {
		t.Error("Error.Message is empty, want non-empty")
	}

	// Verify connection stays alive after error by sending a valid echo
	echoEnv := &protocol.Envelope{
		Type:      protocol.MessageType_MESSAGE_SEND,
		RequestId: "after-error",
		Payload:   []byte("still alive"),
	}
	sendEnvelope(t, ctx, conn, echoEnv)

	echoResp := readEnvelope(t, ctx, conn)
	if echoResp.RequestId != "after-error" {
		t.Errorf("Echo RequestId = %q, want %q", echoResp.RequestId, "after-error")
	}
}

func TestNonBinaryFrameRejected(t *testing.T) {
	url, cleanup := setupTestServer(t, 65536)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send a text frame instead of binary
	if err := conn.Write(ctx, websocket.MessageText, []byte("hello")); err != nil {
		t.Fatalf("Failed to write text frame: %v", err)
	}

	// Server should close with StatusUnsupportedData
	_, _, err := conn.Read(ctx)
	if err == nil {
		t.Fatal("Expected error from read after text frame, got nil")
	}

	if status := websocket.CloseStatus(err); status != websocket.StatusUnsupportedData {
		t.Errorf("Close status = %d, want %d (StatusUnsupportedData)", status, websocket.StatusUnsupportedData)
	}
}

func TestMessageSizeLimit(t *testing.T) {
	const maxSize = 1024

	url, cleanup := setupTestServer(t, maxSize)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Verify a message within the limit works first
	smallEnv := &protocol.Envelope{
		Type:      protocol.MessageType_MESSAGE_SEND,
		RequestId: "small",
		Payload:   []byte("ok"),
	}
	sendEnvelope(t, ctx, conn, smallEnv)

	resp := readEnvelope(t, ctx, conn)
	if resp.RequestId != "small" {
		t.Fatalf("Small message echo failed: RequestId = %q, want %q", resp.RequestId, "small")
	}

	// Send a message exceeding the limit
	oversized := make([]byte, maxSize+512)
	if err := conn.Write(ctx, websocket.MessageBinary, oversized); err != nil {
		t.Fatalf("Failed to write oversized message: %v", err)
	}

	// Server should close the connection
	_, _, err := conn.Read(ctx)
	if err == nil {
		t.Fatal("Expected error after oversized message, got nil")
	}
}

func TestSendBufferFull(t *testing.T) {
	// Create a conn with a minimal send buffer to test overflow behavior.
	c := &Conn{
		id:   "test-buffer",
		send: make(chan []byte, 1),
	}

	// Fill the buffer with one message
	first := &protocol.Envelope{
		Type:      protocol.MessageType_MESSAGE_SEND,
		RequestId: "first",
	}
	c.sendEnvelope(first)

	// Send another message — should be dropped without blocking
	done := make(chan struct{})
	go func() {
		overflow := &protocol.Envelope{
			Type:      protocol.MessageType_MESSAGE_SEND,
			RequestId: "overflow",
		}
		c.sendEnvelope(overflow)
		close(done)
	}()

	select {
	case <-done:
		// sendEnvelope returned without blocking
	case <-time.After(time.Second):
		t.Fatal("sendEnvelope blocked on full send buffer")
	}

	// Drain the buffer and verify only the first message is present
	<-c.send

	select {
	case <-c.send:
		t.Error("Expected empty buffer after draining one message, but got another")
	default:
		// Buffer is empty — overflow was dropped
	}
}
