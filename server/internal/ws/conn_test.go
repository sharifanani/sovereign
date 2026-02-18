package ws

import (
	"context"
	"crypto/sha256"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"

	"github.com/sovereign-im/sovereign/server/internal/auth"
	"github.com/sovereign-im/sovereign/server/internal/protocol"
	"github.com/sovereign-im/sovereign/server/internal/store"
)

const testSessionToken = "test-session-token-abc123"

// setupTestServer creates a test WebSocket server without auth (nil service).
// Only use for tests that don't need auth (ping, binary frame, etc.).
func setupTestServer(t *testing.T, maxMessageSize int) (string, func()) {
	t.Helper()

	hub := NewHub()
	go hub.Run()

	handler := UpgradeHandler(hub, maxMessageSize, nil)
	server := httptest.NewServer(handler)

	url := "ws" + strings.TrimPrefix(server.URL, "http")

	cleanup := func() {
		server.Close()
		hub.Stop()
	}

	return url, cleanup
}

// setupTestServerWithAuth creates a test WebSocket server backed by an in-memory
// SQLite store and auth service. Returns the URL, cleanup, and store so tests
// can seed users/sessions.
func setupTestServerWithAuth(t *testing.T, maxMessageSize int) (string, func(), *store.Store) {
	t.Helper()

	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	authSvc, err := auth.NewService(s, "Test Server", "localhost", []string{"http://localhost:8080"})
	if err != nil {
		s.Close()
		t.Fatalf("auth.NewService: %v", err)
	}

	hub := NewHub()
	go hub.Run()

	handler := UpgradeHandler(hub, maxMessageSize, authSvc)
	server := httptest.NewServer(handler)

	url := "ws" + strings.TrimPrefix(server.URL, "http")

	cleanup := func() {
		server.Close()
		hub.Stop()
		s.Close()
	}

	return url, cleanup, s
}

// seedTestUser creates a user and session in the store for session-token auth.
func seedTestUser(t *testing.T, s *store.Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().Unix()

	u := &store.User{
		ID:          "test-user-id",
		Username:    "testuser",
		DisplayName: "Test User",
		Role:        "member",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.CreateUser(ctx, u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	h := sha256.Sum256([]byte(testSessionToken))
	sess := &store.Session{
		ID:         "test-session-id",
		UserID:     "test-user-id",
		TokenHash:  h[:],
		CreatedAt:  now,
		ExpiresAt:  now + 86400, // 24 hours
		LastSeenAt: now,
	}
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
}

// authenticateConn sends a session token auth request and reads the success response.
func authenticateConn(t *testing.T, ctx context.Context, conn *websocket.Conn) {
	t.Helper()

	// Send AUTH_REQUEST with the session token as "username" for session-token reconnection.
	authReq := &protocol.AuthRequest{
		Username: testSessionToken,
	}
	payload, err := proto.Marshal(authReq)
	if err != nil {
		t.Fatalf("Failed to marshal AuthRequest: %v", err)
	}

	env := &protocol.Envelope{
		Type:      protocol.MessageType_AUTH_REQUEST,
		RequestId: "auth-req",
		Payload:   payload,
	}
	sendEnvelope(t, ctx, conn, env)

	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_AUTH_SUCCESS {
		t.Fatalf("Expected AUTH_SUCCESS, got %v", resp.Type)
	}
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
			url, cleanup, s := setupTestServerWithAuth(t, 65536)
			defer cleanup()
			seedTestUser(t, s)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn := dialTestServer(t, ctx, url)
			defer conn.Close(websocket.StatusNormalClosure, "")

			// Authenticate first
			authenticateConn(t, ctx, conn)

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
			// Ping works during auth phase too, use simple server
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

	// Verify connection stays alive after error — send a valid PING
	// (PING works during auth phase)
	pingPayload, _ := proto.Marshal(&protocol.Ping{Timestamp: 1234})
	pingEnv := &protocol.Envelope{
		Type:      protocol.MessageType_PING,
		RequestId: "after-error",
		Payload:   pingPayload,
	}
	sendEnvelope(t, ctx, conn, pingEnv)

	pongResp := readEnvelope(t, ctx, conn)
	if pongResp.Type != protocol.MessageType_PONG {
		t.Errorf("After-error response type = %v, want PONG", pongResp.Type)
	}
	if pongResp.RequestId != "after-error" {
		t.Errorf("After-error RequestId = %q, want %q", pongResp.RequestId, "after-error")
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

	url, cleanup, s := setupTestServerWithAuth(t, maxSize)
	defer cleanup()
	seedTestUser(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Authenticate first
	authenticateConn(t, ctx, conn)

	// Verify a message within the limit works
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

// --- Auth Lifecycle Tests ---

func TestAuthTimeout(t *testing.T) {
	// Use a custom Conn to test auth timeout without waiting 10 seconds.
	// We test the timeout mechanism by creating a connection and
	// verifying the server closes it when no auth message arrives.
	url, cleanup := setupTestServer(t, 65536)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Don't send any auth message. Wait for the server to close the connection.
	_, _, err := conn.Read(ctx)
	if err == nil {
		t.Fatal("Expected connection to be closed by auth timeout")
	}

	status := websocket.CloseStatus(err)
	if status != 4001 {
		t.Errorf("Close status = %d, want 4001 (Auth Timeout)", status)
	}
}

func TestSessionTokenReconnection(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTestUser(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send AUTH_REQUEST with session token
	authReq := &protocol.AuthRequest{
		Username: testSessionToken,
	}
	payload, err := proto.Marshal(authReq)
	if err != nil {
		t.Fatalf("Failed to marshal AuthRequest: %v", err)
	}

	env := &protocol.Envelope{
		Type:      protocol.MessageType_AUTH_REQUEST,
		RequestId: "session-recon",
		Payload:   payload,
	}
	sendEnvelope(t, ctx, conn, env)

	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_AUTH_SUCCESS {
		t.Fatalf("Type = %v, want AUTH_SUCCESS", resp.Type)
	}

	var success protocol.AuthSuccess
	if err := proto.Unmarshal(resp.Payload, &success); err != nil {
		t.Fatalf("Failed to unmarshal AuthSuccess: %v", err)
	}

	if success.UserId != "test-user-id" {
		t.Errorf("UserId = %q, want %q", success.UserId, "test-user-id")
	}
	if success.Username != "testuser" {
		t.Errorf("Username = %q, want %q", success.Username, "testuser")
	}
	if success.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", success.DisplayName, "Test User")
	}
}

func TestMessageBeforeAuth(t *testing.T) {
	url, cleanup, _ := setupTestServerWithAuth(t, 65536)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send a non-auth message before authenticating
	env := &protocol.Envelope{
		Type:      protocol.MessageType_MESSAGE_SEND,
		RequestId: "pre-auth",
		Payload:   []byte("should fail"),
	}
	sendEnvelope(t, ctx, conn, env)

	// Should receive ERROR with "Authentication required"
	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_ERROR {
		t.Fatalf("Type = %v, want ERROR", resp.Type)
	}

	var errMsg protocol.Error
	if err := proto.Unmarshal(resp.Payload, &errMsg); err != nil {
		t.Fatalf("Failed to unmarshal Error: %v", err)
	}
	if errMsg.Code != 3002 {
		t.Errorf("Code = %d, want 3002", errMsg.Code)
	}
}

func TestAuthLoginBeginForNonExistentUser(t *testing.T) {
	url, cleanup, _ := setupTestServerWithAuth(t, 65536)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send AUTH_REQUEST for a user that doesn't exist
	authReq := &protocol.AuthRequest{
		Username: "nonexistent",
	}
	payload, _ := proto.Marshal(authReq)

	env := &protocol.Envelope{
		Type:      protocol.MessageType_AUTH_REQUEST,
		RequestId: "login-fail",
		Payload:   payload,
	}
	sendEnvelope(t, ctx, conn, env)

	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_AUTH_ERROR {
		t.Fatalf("Type = %v, want AUTH_ERROR", resp.Type)
	}

	var authErr protocol.AuthError
	if err := proto.Unmarshal(resp.Payload, &authErr); err != nil {
		t.Fatalf("Failed to unmarshal AuthError: %v", err)
	}
	if authErr.ErrorCode != 1001 {
		t.Errorf("ErrorCode = %d, want 1001 (Invalid credential)", authErr.ErrorCode)
	}
}

func TestAuthRegisterBegin(t *testing.T) {
	url, cleanup, _ := setupTestServerWithAuth(t, 65536)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send AUTH_REGISTER_REQUEST for a new user
	regReq := &protocol.AuthRegisterRequest{
		Username:    "newuser",
		DisplayName: "New User",
	}
	payload, _ := proto.Marshal(regReq)

	env := &protocol.Envelope{
		Type:      protocol.MessageType_AUTH_REGISTER_REQUEST,
		RequestId: "reg-1",
		Payload:   payload,
	}
	sendEnvelope(t, ctx, conn, env)

	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_AUTH_REGISTER_CHALLENGE {
		t.Fatalf("Type = %v, want AUTH_REGISTER_CHALLENGE", resp.Type)
	}

	var challenge protocol.AuthRegisterChallenge
	if err := proto.Unmarshal(resp.Payload, &challenge); err != nil {
		t.Fatalf("Failed to unmarshal AuthRegisterChallenge: %v", err)
	}

	if len(challenge.CredentialCreationOptions) == 0 {
		t.Error("CredentialCreationOptions is empty")
	}
}

func TestAuthRegisterDuplicateUsername(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTestUser(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Try to register with an existing username
	regReq := &protocol.AuthRegisterRequest{
		Username:    "testuser", // already exists from seedTestUser
		DisplayName: "Duplicate",
	}
	payload, _ := proto.Marshal(regReq)

	env := &protocol.Envelope{
		Type:      protocol.MessageType_AUTH_REGISTER_REQUEST,
		RequestId: "reg-dup",
		Payload:   payload,
	}
	sendEnvelope(t, ctx, conn, env)

	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_AUTH_ERROR {
		t.Fatalf("Type = %v, want AUTH_ERROR", resp.Type)
	}

	var authErr protocol.AuthError
	if err := proto.Unmarshal(resp.Payload, &authErr); err != nil {
		t.Fatalf("Failed to unmarshal AuthError: %v", err)
	}
	if authErr.ErrorCode != 1003 {
		t.Errorf("ErrorCode = %d, want 1003 (Registration failed)", authErr.ErrorCode)
	}
}
