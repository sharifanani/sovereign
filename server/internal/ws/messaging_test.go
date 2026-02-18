package ws

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"

	"github.com/sovereign-im/sovereign/server/internal/protocol"
	"github.com/sovereign-im/sovereign/server/internal/store"
)

// seedTwoUsers creates alice and bob with sessions, and a conversation between them.
func seedTwoUsers(t *testing.T, s *store.Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().Unix()

	users := []struct {
		id, username, token string
	}{
		{"alice-id", "alice", "alice-session-token"},
		{"bob-id", "bob", "bob-session-token"},
	}

	for _, u := range users {
		if err := s.CreateUser(ctx, &store.User{
			ID: u.id, Username: u.username, DisplayName: u.username,
			Role: "member", Enabled: true, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("CreateUser(%s): %v", u.username, err)
		}
		h := sha256.Sum256([]byte(u.token))
		if err := s.CreateSession(ctx, &store.Session{
			ID: "sess-" + u.id, UserID: u.id, TokenHash: h[:],
			CreatedAt: now, ExpiresAt: now + 86400, LastSeenAt: now,
		}); err != nil {
			t.Fatalf("CreateSession(%s): %v", u.username, err)
		}
	}
}

func authenticateAs(t *testing.T, ctx context.Context, conn *websocket.Conn, sessionToken string) {
	t.Helper()
	authReq := &protocol.AuthRequest{Username: sessionToken}
	payload, _ := proto.Marshal(authReq)
	sendEnvelope(t, ctx, conn, &protocol.Envelope{
		Type: protocol.MessageType_AUTH_REQUEST, RequestId: "auth", Payload: payload,
	})
	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_AUTH_SUCCESS {
		t.Fatalf("Expected AUTH_SUCCESS, got %v", resp.Type)
	}
}

func TestGroupCreateFlow(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTwoUsers(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, conn, "alice-session-token")

	// Create a group with bob.
	createMsg := &protocol.GroupCreate{Title: "Test Chat", MemberIds: []string{"bob-id"}}
	payload, _ := proto.Marshal(createMsg)
	sendEnvelope(t, ctx, conn, &protocol.Envelope{
		Type: protocol.MessageType_GROUP_CREATE, RequestId: "gc-1", Payload: payload,
	})

	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_GROUP_CREATED {
		t.Fatalf("Type = %v, want GROUP_CREATED", resp.Type)
	}
	if resp.RequestId != "gc-1" {
		t.Errorf("RequestId = %q, want gc-1", resp.RequestId)
	}

	var created protocol.GroupCreated
	if err := proto.Unmarshal(resp.Payload, &created); err != nil {
		t.Fatalf("Unmarshal GroupCreated: %v", err)
	}
	if created.ConversationId == "" {
		t.Error("ConversationId is empty")
	}
	if created.Title != "Test Chat" {
		t.Errorf("Title = %q, want Test Chat", created.Title)
	}
	if len(created.Members) != 2 {
		t.Errorf("Members count = %d, want 2", len(created.Members))
	}
}

func TestMessageSendToOnlineRecipient(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTwoUsers(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect alice and bob.
	aliceConn := dialTestServer(t, ctx, url)
	defer aliceConn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, aliceConn, "alice-session-token")

	bobConn := dialTestServer(t, ctx, url)
	defer bobConn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, bobConn, "bob-session-token")

	// Alice creates a group with bob.
	createMsg := &protocol.GroupCreate{Title: "DM", MemberIds: []string{"bob-id"}}
	payload, _ := proto.Marshal(createMsg)
	sendEnvelope(t, ctx, aliceConn, &protocol.Envelope{
		Type: protocol.MessageType_GROUP_CREATE, RequestId: "gc", Payload: payload,
	})

	createdResp := readEnvelope(t, ctx, aliceConn)
	var created protocol.GroupCreated
	if err := proto.Unmarshal(createdResp.Payload, &created); err != nil {
		t.Fatalf("Unmarshal GroupCreated: %v", err)
	}

	// Bob receives GROUP_MEMBER_ADDED notification (about bob being added).
	bobNotification := readEnvelope(t, ctx, bobConn)
	if bobNotification.Type != protocol.MessageType_GROUP_MEMBER_ADDED {
		t.Fatalf("Bob notification: type = %v, want GROUP_MEMBER_ADDED", bobNotification.Type)
	}

	// Alice sends a message.
	msgSend := &protocol.MessageSend{
		ConversationId:   created.ConversationId,
		EncryptedPayload: []byte("hello bob"),
		MessageType:      "text",
	}
	sendPayload, _ := proto.Marshal(msgSend)
	sendEnvelope(t, ctx, aliceConn, &protocol.Envelope{
		Type: protocol.MessageType_MESSAGE_SEND, RequestId: "ms-1", Payload: sendPayload,
	})

	// Alice receives echo (MESSAGE_RECEIVE with her requestId).
	aliceEcho := readEnvelope(t, ctx, aliceConn)
	if aliceEcho.Type != protocol.MessageType_MESSAGE_RECEIVE {
		t.Fatalf("Alice echo type = %v, want MESSAGE_RECEIVE", aliceEcho.Type)
	}
	if aliceEcho.RequestId != "ms-1" {
		t.Errorf("Alice echo RequestId = %q, want ms-1", aliceEcho.RequestId)
	}

	var aliceReceive protocol.MessageReceive
	if err := proto.Unmarshal(aliceEcho.Payload, &aliceReceive); err != nil {
		t.Fatalf("Unmarshal alice receive: %v", err)
	}
	if aliceReceive.SenderId != "alice-id" {
		t.Errorf("SenderId = %q, want alice-id", aliceReceive.SenderId)
	}
	if aliceReceive.MessageId == "" {
		t.Error("MessageId is empty")
	}

	// Bob receives the message.
	bobReceive := readEnvelope(t, ctx, bobConn)
	if bobReceive.Type != protocol.MessageType_MESSAGE_RECEIVE {
		t.Fatalf("Bob receive type = %v, want MESSAGE_RECEIVE", bobReceive.Type)
	}

	var bobMsg protocol.MessageReceive
	if err := proto.Unmarshal(bobReceive.Payload, &bobMsg); err != nil {
		t.Fatalf("Unmarshal bob receive: %v", err)
	}
	if string(bobMsg.EncryptedPayload) != "hello bob" {
		t.Errorf("payload = %q, want hello bob", bobMsg.EncryptedPayload)
	}
	if bobMsg.SenderId != "alice-id" {
		t.Errorf("SenderId = %q, want alice-id", bobMsg.SenderId)
	}
}

func TestMessageAckUpdatesDeliveryAndNotifiesSender(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTwoUsers(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	aliceConn := dialTestServer(t, ctx, url)
	defer aliceConn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, aliceConn, "alice-session-token")

	bobConn := dialTestServer(t, ctx, url)
	defer bobConn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, bobConn, "bob-session-token")

	// Create group.
	createPayload, _ := proto.Marshal(&protocol.GroupCreate{Title: "DM", MemberIds: []string{"bob-id"}})
	sendEnvelope(t, ctx, aliceConn, &protocol.Envelope{
		Type: protocol.MessageType_GROUP_CREATE, RequestId: "gc", Payload: createPayload,
	})
	createdResp := readEnvelope(t, ctx, aliceConn)
	var created protocol.GroupCreated
	proto.Unmarshal(createdResp.Payload, &created)

	// Drain bob's member-added notification.
	readEnvelope(t, ctx, bobConn)

	// Alice sends a message.
	msgPayload, _ := proto.Marshal(&protocol.MessageSend{
		ConversationId: created.ConversationId, EncryptedPayload: []byte("hi"), MessageType: "text",
	})
	sendEnvelope(t, ctx, aliceConn, &protocol.Envelope{
		Type: protocol.MessageType_MESSAGE_SEND, RequestId: "ms", Payload: msgPayload,
	})

	// Alice gets echo.
	aliceEcho := readEnvelope(t, ctx, aliceConn)
	var echoMsg protocol.MessageReceive
	proto.Unmarshal(aliceEcho.Payload, &echoMsg)

	// Bob receives message.
	readEnvelope(t, ctx, bobConn)

	// Bob sends ACK.
	ackPayload, _ := proto.Marshal(&protocol.MessageAck{MessageId: echoMsg.MessageId})
	sendEnvelope(t, ctx, bobConn, &protocol.Envelope{
		Type: protocol.MessageType_MESSAGE_ACK, RequestId: "ack", Payload: ackPayload,
	})

	// Alice receives MESSAGE_DELIVERED notification.
	deliveredResp := readEnvelope(t, ctx, aliceConn)
	if deliveredResp.Type != protocol.MessageType_MESSAGE_DELIVERED {
		t.Fatalf("Type = %v, want MESSAGE_DELIVERED", deliveredResp.Type)
	}

	var delivered protocol.MessageDelivered
	if err := proto.Unmarshal(deliveredResp.Payload, &delivered); err != nil {
		t.Fatalf("Unmarshal MessageDelivered: %v", err)
	}
	if delivered.MessageId != echoMsg.MessageId {
		t.Errorf("MessageId = %s, want %s", delivered.MessageId, echoMsg.MessageId)
	}
	if delivered.DeliveredTo != "bob-id" {
		t.Errorf("DeliveredTo = %s, want bob-id", delivered.DeliveredTo)
	}
}

func TestMessageSendUnauthorized(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTwoUsers(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, conn, "alice-session-token")

	// Try sending to a conversation that doesn't exist (alice isn't a member).
	msgPayload, _ := proto.Marshal(&protocol.MessageSend{
		ConversationId: "nonexistent-conv", EncryptedPayload: []byte("hi"), MessageType: "text",
	})
	sendEnvelope(t, ctx, conn, &protocol.Envelope{
		Type: protocol.MessageType_MESSAGE_SEND, RequestId: "unauth", Payload: msgPayload,
	})

	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_ERROR {
		t.Fatalf("Type = %v, want ERROR", resp.Type)
	}
	var errMsg protocol.Error
	proto.Unmarshal(resp.Payload, &errMsg)
	if errMsg.Code != 4001 {
		t.Errorf("Code = %d, want 4001 (not a member)", errMsg.Code)
	}
}

func TestMLSKeyPackageUploadAndFetch(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTwoUsers(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Alice uploads, bob fetches.
	aliceConn := dialTestServer(t, ctx, url)
	defer aliceConn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, aliceConn, "alice-session-token")

	// Upload a key package.
	uploadPayload, _ := proto.Marshal(&protocol.MLSKeyPackageUpload{KeyPackageData: []byte("alice-kp")})
	sendEnvelope(t, ctx, aliceConn, &protocol.Envelope{
		Type: protocol.MessageType_MLS_KEY_PACKAGE_UPLOAD, RequestId: "up-1", Payload: uploadPayload,
	})

	// No response expected for upload (silent success).
	// Connect bob and fetch.
	bobConn := dialTestServer(t, ctx, url)
	defer bobConn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, bobConn, "bob-session-token")

	fetchPayload, _ := proto.Marshal(&protocol.MLSKeyPackageFetch{UserId: "alice-id"})
	sendEnvelope(t, ctx, bobConn, &protocol.Envelope{
		Type: protocol.MessageType_MLS_KEY_PACKAGE_FETCH, RequestId: "fetch-1", Payload: fetchPayload,
	})

	resp := readEnvelope(t, ctx, bobConn)
	if resp.Type != protocol.MessageType_MLS_KEY_PACKAGE_RESPONSE {
		t.Fatalf("Type = %v, want MLS_KEY_PACKAGE_RESPONSE", resp.Type)
	}

	var kpResp protocol.MLSKeyPackageResponse
	if err := proto.Unmarshal(resp.Payload, &kpResp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if string(kpResp.KeyPackageData) != "alice-kp" {
		t.Errorf("KeyPackageData = %q, want alice-kp", kpResp.KeyPackageData)
	}
	if kpResp.UserId != "alice-id" {
		t.Errorf("UserId = %q, want alice-id", kpResp.UserId)
	}

	// Second fetch should fail (single-use).
	sendEnvelope(t, ctx, bobConn, &protocol.Envelope{
		Type: protocol.MessageType_MLS_KEY_PACKAGE_FETCH, RequestId: "fetch-2", Payload: fetchPayload,
	})
	resp2 := readEnvelope(t, ctx, bobConn)
	if resp2.Type != protocol.MessageType_ERROR {
		t.Fatalf("Second fetch type = %v, want ERROR", resp2.Type)
	}
}

func TestMLSKeyPackageUploadInvalidData(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTwoUsers(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, conn, "alice-session-token")

	// Upload empty key package.
	uploadPayload, _ := proto.Marshal(&protocol.MLSKeyPackageUpload{KeyPackageData: []byte{}})
	sendEnvelope(t, ctx, conn, &protocol.Envelope{
		Type: protocol.MessageType_MLS_KEY_PACKAGE_UPLOAD, RequestId: "up-bad", Payload: uploadPayload,
	})

	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_ERROR {
		t.Fatalf("Type = %v, want ERROR", resp.Type)
	}
	var errMsg protocol.Error
	proto.Unmarshal(resp.Payload, &errMsg)
	if errMsg.Code != 5001 {
		t.Errorf("Code = %d, want 5001 (invalid key package)", errMsg.Code)
	}
}

func TestMLSWelcomeForwarding(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTwoUsers(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	aliceConn := dialTestServer(t, ctx, url)
	defer aliceConn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, aliceConn, "alice-session-token")

	bobConn := dialTestServer(t, ctx, url)
	defer bobConn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, bobConn, "bob-session-token")

	// Alice sends Welcome to bob.
	welcomePayload, _ := proto.Marshal(&protocol.MLSWelcome{
		ConversationId: "conv-1",
		RecipientId:    "bob-id",
		WelcomeData:    []byte("welcome-data"),
	})
	sendEnvelope(t, ctx, aliceConn, &protocol.Envelope{
		Type: protocol.MessageType_MLS_WELCOME, RequestId: "w-1", Payload: welcomePayload,
	})

	// Bob should receive MLS_WELCOME_RECEIVE.
	resp := readEnvelope(t, ctx, bobConn)
	if resp.Type != protocol.MessageType_MLS_WELCOME_RECEIVE {
		t.Fatalf("Type = %v, want MLS_WELCOME_RECEIVE", resp.Type)
	}

	var received protocol.MLSWelcomeReceive
	if err := proto.Unmarshal(resp.Payload, &received); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if received.ConversationId != "conv-1" {
		t.Errorf("ConversationId = %q, want conv-1", received.ConversationId)
	}
	if received.SenderId != "alice-id" {
		t.Errorf("SenderId = %q, want alice-id", received.SenderId)
	}
	if string(received.WelcomeData) != "welcome-data" {
		t.Errorf("WelcomeData = %q, want welcome-data", received.WelcomeData)
	}
}

func TestMLSCommitBroadcast(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTwoUsers(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	aliceConn := dialTestServer(t, ctx, url)
	defer aliceConn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, aliceConn, "alice-session-token")

	bobConn := dialTestServer(t, ctx, url)
	defer bobConn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, bobConn, "bob-session-token")

	// Create a group first.
	createPayload, _ := proto.Marshal(&protocol.GroupCreate{Title: "Group", MemberIds: []string{"bob-id"}})
	sendEnvelope(t, ctx, aliceConn, &protocol.Envelope{
		Type: protocol.MessageType_GROUP_CREATE, RequestId: "gc", Payload: createPayload,
	})
	createdResp := readEnvelope(t, ctx, aliceConn)
	var created protocol.GroupCreated
	proto.Unmarshal(createdResp.Payload, &created)

	// Drain bob's member-added notification.
	readEnvelope(t, ctx, bobConn)

	// Alice sends MLS Commit.
	commitPayload, _ := proto.Marshal(&protocol.MLSCommit{
		ConversationId: created.ConversationId,
		CommitData:     []byte("commit-data"),
	})
	sendEnvelope(t, ctx, aliceConn, &protocol.Envelope{
		Type: protocol.MessageType_MLS_COMMIT, RequestId: "commit-1", Payload: commitPayload,
	})

	// Bob should receive MLS_COMMIT_BROADCAST.
	resp := readEnvelope(t, ctx, bobConn)
	if resp.Type != protocol.MessageType_MLS_COMMIT_BROADCAST {
		t.Fatalf("Type = %v, want MLS_COMMIT_BROADCAST", resp.Type)
	}

	var broadcast protocol.MLSCommitBroadcast
	if err := proto.Unmarshal(resp.Payload, &broadcast); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if broadcast.ConversationId != created.ConversationId {
		t.Errorf("ConversationId = %q, want %q", broadcast.ConversationId, created.ConversationId)
	}
	if broadcast.SenderId != "alice-id" {
		t.Errorf("SenderId = %q, want alice-id", broadcast.SenderId)
	}
	if string(broadcast.CommitData) != "commit-data" {
		t.Errorf("CommitData = %q, want commit-data", broadcast.CommitData)
	}
}

func TestMLSCommitNonMemberRejected(t *testing.T) {
	url, cleanup, s := setupTestServerWithAuth(t, 65536)
	defer cleanup()
	seedTwoUsers(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialTestServer(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")
	authenticateAs(t, ctx, conn, "alice-session-token")

	// Send Commit for a conversation alice isn't in.
	commitPayload, _ := proto.Marshal(&protocol.MLSCommit{
		ConversationId: "nonexistent-conv",
		CommitData:     []byte("data"),
	})
	sendEnvelope(t, ctx, conn, &protocol.Envelope{
		Type: protocol.MessageType_MLS_COMMIT, RequestId: "commit-bad", Payload: commitPayload,
	})

	resp := readEnvelope(t, ctx, conn)
	if resp.Type != protocol.MessageType_ERROR {
		t.Fatalf("Type = %v, want ERROR", resp.Type)
	}
	var errMsg protocol.Error
	proto.Unmarshal(resp.Payload, &errMsg)
	if errMsg.Code != 4001 {
		t.Errorf("Code = %d, want 4001", errMsg.Code)
	}
}
