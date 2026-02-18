package ws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"nhooyr.io/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/sovereign-im/sovereign/server/internal/auth"
	"github.com/sovereign-im/sovereign/server/internal/mls"
	"github.com/sovereign-im/sovereign/server/internal/protocol"
	"github.com/sovereign-im/sovereign/server/internal/store"
)

// Connection states.
const (
	stateAuthenticating int32 = 0
	stateReady          int32 = 1
	stateDisconnected   int32 = 2
)

// Auth timeout before connection is closed.
const authTimeout = 10 * time.Second

// Conn wraps a WebSocket connection with read/write pumps and auth state.
type Conn struct {
	id     string
	ws     *websocket.Conn
	hub    *Hub
	send   chan []byte
	once   sync.Once
	cancel context.CancelFunc

	maxMessageSize int64

	// Auth state (atomic for goroutine safety with auth timer).
	state       atomic.Int32
	authService *auth.Service
	userID      string
	username    string
	challengeID string
	authTimer   *time.Timer

	// Messaging dependencies.
	store      *store.Store
	mlsService *mls.Service
}

// NewConn creates a new Conn.
func NewConn(id string, ws *websocket.Conn, hub *Hub, maxMessageSize int, authService *auth.Service, st *store.Store, mlsSvc *mls.Service) *Conn {
	c := &Conn{
		id:             id,
		ws:             ws,
		hub:            hub,
		send:           make(chan []byte, 256),
		maxMessageSize: int64(maxMessageSize),
		authService:    authService,
		store:          st,
		mlsService:     mlsSvc,
	}
	c.state.Store(stateAuthenticating)
	return c
}

// Run starts the read and write pumps. It blocks until the connection is closed.
func (c *Conn) Run(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)

	c.hub.Register(c)
	defer c.hub.Unregister(c)

	c.ws.SetReadLimit(c.maxMessageSize)

	// Start auth timeout.
	c.startAuthTimeout()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		c.writePump(ctx)
	}()

	go func() {
		defer wg.Done()
		c.readPump(ctx)
	}()

	wg.Wait()
	c.ws.Close(websocket.StatusNormalClosure, "")
}

// startAuthTimeout closes the connection if auth isn't completed within the timeout.
func (c *Conn) startAuthTimeout() {
	c.authTimer = time.AfterFunc(authTimeout, func() {
		if !c.state.CompareAndSwap(stateAuthenticating, stateDisconnected) {
			return // auth already completed
		}
		log.Printf("[%s] Auth timeout", c.id)
		c.ws.Close(websocket.StatusCode(4001), "Authentication Timeout")
		c.close()
	})
}

// readPump reads messages from the WebSocket and processes them.
func (c *Conn) readPump(ctx context.Context) {
	defer c.close()

	for {
		typ, data, err := c.ws.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				log.Printf("[%s] Connection closed normally", c.id)
			} else {
				log.Printf("[%s] Read error: %v", c.id, err)
			}
			return
		}

		if typ != websocket.MessageBinary {
			log.Printf("[%s] Received non-binary message, closing", c.id)
			c.ws.Close(websocket.StatusUnsupportedData, "binary frames only")
			return
		}

		var env protocol.Envelope
		if err := proto.Unmarshal(data, &env); err != nil {
			log.Printf("[%s] Failed to unmarshal envelope: %v", c.id, err)
			c.sendError(&env, 3001, "Invalid message format", false)
			continue
		}

		c.handleEnvelope(ctx, &env)
	}
}

// writePump writes messages from the send channel to the WebSocket.
func (c *Conn) writePump(ctx context.Context) {
	defer c.close()

	for {
		select {
		case data, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.ws.Write(ctx, websocket.MessageBinary, data); err != nil {
				log.Printf("[%s] Write error: %v", c.id, err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// handleEnvelope routes messages based on connection state.
func (c *Conn) handleEnvelope(ctx context.Context, env *protocol.Envelope) {
	state := c.state.Load()

	switch state {
	case stateAuthenticating:
		c.handleAuthMessage(ctx, env)
	case stateReady:
		c.handleReadyMessage(ctx, env)
	default:
		log.Printf("[%s] Received message in disconnected state", c.id)
	}
}

// handleAuthMessage processes messages during the authentication phase.
func (c *Conn) handleAuthMessage(ctx context.Context, env *protocol.Envelope) {
	switch env.Type {
	case protocol.MessageType_AUTH_REQUEST:
		c.handleAuthRequest(ctx, env)
	case protocol.MessageType_AUTH_RESPONSE:
		c.handleAuthResponse(ctx, env)
	case protocol.MessageType_AUTH_REGISTER_REQUEST:
		c.handleAuthRegisterRequest(ctx, env)
	case protocol.MessageType_AUTH_REGISTER_RESPONSE:
		c.handleAuthRegisterResponse(ctx, env)
	case protocol.MessageType_PING:
		c.handlePing(env)
	default:
		c.sendError(env, 3002, "Authentication required", false)
	}
}

// handleReadyMessage processes messages after authentication is complete.
func (c *Conn) handleReadyMessage(ctx context.Context, env *protocol.Envelope) {
	switch env.Type {
	case protocol.MessageType_PING:
		c.handlePing(env)
	case protocol.MessageType_ERROR:
		log.Printf("[%s] Received error message, discarding", c.id)

	// Messaging
	case protocol.MessageType_MESSAGE_SEND:
		c.handleMessageSend(ctx, env)
	case protocol.MessageType_MESSAGE_ACK:
		c.handleMessageAck(ctx, env)

	// Groups
	case protocol.MessageType_GROUP_CREATE:
		c.handleGroupCreate(ctx, env)
	case protocol.MessageType_GROUP_INVITE:
		c.handleGroupInvite(ctx, env)
	case protocol.MessageType_GROUP_LEAVE:
		c.handleGroupLeave(ctx, env)

	// MLS
	case protocol.MessageType_MLS_KEY_PACKAGE_UPLOAD:
		c.handleMLSKeyPackageUpload(ctx, env)
	case protocol.MessageType_MLS_KEY_PACKAGE_FETCH:
		c.handleMLSKeyPackageFetch(ctx, env)
	case protocol.MessageType_MLS_WELCOME:
		c.handleMLSWelcome(ctx, env)
	case protocol.MessageType_MLS_COMMIT:
		c.handleMLSCommit(ctx, env)

	default:
		c.sendError(env, 3001, "Unknown message type", false)
	}
}

// ============================================================================
// Messaging Handlers
// ============================================================================

func (c *Conn) handleMessageSend(ctx context.Context, env *protocol.Envelope) {
	var msg protocol.MessageSend
	if err := proto.Unmarshal(env.Payload, &msg); err != nil {
		c.sendError(env, 3001, "Invalid message.send payload", false)
		return
	}

	// Validate membership.
	isMember, err := c.store.IsUserMember(ctx, msg.ConversationId, c.userID)
	if err != nil {
		log.Printf("[%s] membership check error: %v", c.id, err)
		c.sendError(env, 9001, "Internal error", false)
		return
	}
	if !isMember {
		c.sendError(env, 4001, "Not a member of this conversation", false)
		return
	}

	// Map message_type string to int for storage.
	msgTypeInt := store.MsgTypeApplication

	// Store message.
	messageID, serverTS, err := c.store.InsertMessage(ctx, msg.ConversationId, c.userID, msg.EncryptedPayload, msgTypeInt, 0)
	if err != nil {
		log.Printf("[%s] insert message error: %v", c.id, err)
		c.sendError(env, 9001, "Failed to store message", false)
		return
	}

	// Build MESSAGE_RECEIVE envelope.
	receiveMsg := &protocol.MessageReceive{
		MessageId:        messageID,
		ConversationId:   msg.ConversationId,
		SenderId:         c.userID,
		EncryptedPayload: msg.EncryptedPayload,
		ServerTimestamp:  serverTS,
		MessageType:      msg.MessageType,
	}
	receivePayload, err := proto.Marshal(receiveMsg)
	if err != nil {
		log.Printf("[%s] marshal message receive error: %v", c.id, err)
		return
	}
	receiveEnv := &protocol.Envelope{
		Type:    protocol.MessageType_MESSAGE_RECEIVE,
		Payload: receivePayload,
	}

	// Echo back to sender as delivery confirmation (with the request_id).
	senderEnv := &protocol.Envelope{
		Type:      protocol.MessageType_MESSAGE_RECEIVE,
		RequestId: env.RequestId,
		Payload:   receivePayload,
	}
	c.sendEnvelope(senderEnv)

	// Forward to online group members.
	members, err := c.store.GetMembers(ctx, msg.ConversationId)
	if err != nil {
		log.Printf("[%s] get members error: %v", c.id, err)
		return
	}
	for _, m := range members {
		if m.UserID == c.userID {
			continue
		}
		if c.hub.SendToUser(m.UserID, receiveEnv) {
			// Mark delivered for online recipients.
			if err := c.store.UpdateDeliveryStatus(ctx, messageID, m.UserID, store.DeliveryDelivered); err != nil {
				log.Printf("[%s] update delivery status error: %v", c.id, err)
			}
		}
	}
}

func (c *Conn) handleMessageAck(ctx context.Context, env *protocol.Envelope) {
	var msg protocol.MessageAck
	if err := proto.Unmarshal(env.Payload, &msg); err != nil {
		c.sendError(env, 3001, "Invalid message.ack payload", false)
		return
	}

	// Update delivery status to DELIVERED.
	if err := c.store.UpdateDeliveryStatus(ctx, msg.MessageId, c.userID, store.DeliveryDelivered); err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			log.Printf("[%s] update delivery status error: %v", c.id, err)
		}
		return
	}

	// Notify sender that message was delivered.
	senderID, err := c.store.GetMessageSenderID(ctx, msg.MessageId)
	if err != nil {
		log.Printf("[%s] get message sender error: %v", c.id, err)
		return
	}

	deliveredMsg := &protocol.MessageDelivered{
		MessageId:   msg.MessageId,
		DeliveredTo: c.userID,
	}
	deliveredPayload, err := proto.Marshal(deliveredMsg)
	if err != nil {
		log.Printf("[%s] marshal delivered error: %v", c.id, err)
		return
	}
	deliveredEnv := &protocol.Envelope{
		Type:    protocol.MessageType_MESSAGE_DELIVERED,
		Payload: deliveredPayload,
	}
	c.hub.SendToUser(senderID, deliveredEnv)
}

// ============================================================================
// Group Handlers
// ============================================================================

func (c *Conn) handleGroupCreate(ctx context.Context, env *protocol.Envelope) {
	var msg protocol.GroupCreate
	if err := proto.Unmarshal(env.Payload, &msg); err != nil {
		c.sendError(env, 3001, "Invalid group.create payload", false)
		return
	}

	conv, err := c.store.CreateConversation(ctx, msg.Title, c.userID, msg.MemberIds)
	if err != nil {
		log.Printf("[%s] create conversation error: %v", c.id, err)
		c.sendError(env, 9001, "Failed to create group", false)
		return
	}

	// Build member list for response.
	members, err := c.store.GetMembers(ctx, conv.ID)
	if err != nil {
		log.Printf("[%s] get members error: %v", c.id, err)
		c.sendError(env, 9001, "Failed to get group members", false)
		return
	}

	var pbMembers []*protocol.GroupMember
	for _, m := range members {
		user, err := c.store.GetUserByID(ctx, m.UserID)
		if err != nil {
			log.Printf("[%s] get user %s error: %v", c.id, m.UserID, err)
			continue
		}
		pbMembers = append(pbMembers, &protocol.GroupMember{
			UserId:      user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        m.Role,
		})
	}

	// Send GROUP_CREATED to creator.
	created := &protocol.GroupCreated{
		ConversationId: conv.ID,
		Title:          msg.Title,
		Members:        pbMembers,
	}
	c.sendTypedResponse(env, protocol.MessageType_GROUP_CREATED, created)

	// Notify all members with GROUP_MEMBER_ADDED.
	for _, m := range members {
		added := &protocol.GroupMemberAdded{
			ConversationId: conv.ID,
			UserId:         m.UserID,
			AddedBy:        c.userID,
		}
		addedPayload, err := proto.Marshal(added)
		if err != nil {
			continue
		}
		addedEnv := &protocol.Envelope{
			Type:    protocol.MessageType_GROUP_MEMBER_ADDED,
			Payload: addedPayload,
		}
		if m.UserID != c.userID {
			c.hub.SendToUser(m.UserID, addedEnv)
		}
	}
}

func (c *Conn) handleGroupInvite(ctx context.Context, env *protocol.Envelope) {
	var msg protocol.GroupInvite
	if err := proto.Unmarshal(env.Payload, &msg); err != nil {
		c.sendError(env, 3001, "Invalid group.invite payload", false)
		return
	}

	// Validate admin role.
	role, err := c.store.GetMemberRole(ctx, msg.ConversationId, c.userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.sendError(env, 4001, "Not a member of this conversation", false)
		} else {
			c.sendError(env, 9001, "Internal error", false)
		}
		return
	}
	if role != "admin" {
		c.sendError(env, 4003, "Only admins can invite members", false)
		return
	}

	// Add the member.
	if err := c.store.AddMember(ctx, msg.ConversationId, msg.UserId, "member"); err != nil {
		if errors.Is(err, store.ErrConflict) {
			c.sendError(env, 4002, "User is already a member", false)
		} else {
			log.Printf("[%s] add member error: %v", c.id, err)
			c.sendError(env, 9001, "Failed to add member", false)
		}
		return
	}

	// Notify all group members (including new member).
	added := &protocol.GroupMemberAdded{
		ConversationId: msg.ConversationId,
		UserId:         msg.UserId,
		AddedBy:        c.userID,
	}
	addedPayload, err := proto.Marshal(added)
	if err != nil {
		return
	}
	addedEnv := &protocol.Envelope{
		Type:    protocol.MessageType_GROUP_MEMBER_ADDED,
		Payload: addedPayload,
	}

	members, err := c.store.GetMembers(ctx, msg.ConversationId)
	if err != nil {
		log.Printf("[%s] get members error: %v", c.id, err)
		return
	}
	memberIDs := make([]string, len(members))
	for i, m := range members {
		memberIDs[i] = m.UserID
	}
	c.hub.BroadcastToGroup(memberIDs, addedEnv, "")
}

func (c *Conn) handleGroupLeave(ctx context.Context, env *protocol.Envelope) {
	var msg protocol.GroupLeave
	if err := proto.Unmarshal(env.Payload, &msg); err != nil {
		c.sendError(env, 3001, "Invalid group.leave payload", false)
		return
	}

	// Check if user is admin; if so, transfer admin to next oldest member.
	role, err := c.store.GetMemberRole(ctx, msg.ConversationId, c.userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.sendError(env, 4001, "Not a member of this conversation", false)
		} else {
			c.sendError(env, 9001, "Internal error", false)
		}
		return
	}

	if role == "admin" {
		if err := c.store.TransferAdmin(ctx, msg.ConversationId, c.userID); err != nil {
			log.Printf("[%s] transfer admin error: %v", c.id, err)
		}
	}

	// Remove the member.
	if err := c.store.RemoveMember(ctx, msg.ConversationId, c.userID); err != nil {
		log.Printf("[%s] remove member error: %v", c.id, err)
		c.sendError(env, 9001, "Failed to leave group", false)
		return
	}

	// Notify remaining members.
	removed := &protocol.GroupMemberRemoved{
		ConversationId: msg.ConversationId,
		UserId:         c.userID,
		RemovedBy:      c.userID,
	}
	removedPayload, err := proto.Marshal(removed)
	if err != nil {
		return
	}
	removedEnv := &protocol.Envelope{
		Type:    protocol.MessageType_GROUP_MEMBER_REMOVED,
		Payload: removedPayload,
	}

	members, err := c.store.GetMembers(ctx, msg.ConversationId)
	if err != nil {
		log.Printf("[%s] get members error: %v", c.id, err)
		return
	}
	memberIDs := make([]string, len(members))
	for i, m := range members {
		memberIDs[i] = m.UserID
	}
	c.hub.BroadcastToGroup(memberIDs, removedEnv, "")
}

// ============================================================================
// MLS Handlers
// ============================================================================

func (c *Conn) handleMLSKeyPackageUpload(ctx context.Context, env *protocol.Envelope) {
	var msg protocol.MLSKeyPackageUpload
	if err := proto.Unmarshal(env.Payload, &msg); err != nil {
		c.sendError(env, 3001, "Invalid mls.key_package.upload payload", false)
		return
	}

	if err := c.mlsService.UploadKeyPackage(ctx, c.userID, msg.KeyPackageData); err != nil {
		if errors.Is(err, mls.ErrInvalidPayload) {
			c.sendError(env, 5001, "Invalid key package data", false)
		} else {
			log.Printf("[%s] upload key package error: %v", c.id, err)
			c.sendError(env, 9001, "Failed to store key package", false)
		}
		return
	}
	// No explicit response per spec; success is silent.
}

func (c *Conn) handleMLSKeyPackageFetch(ctx context.Context, env *protocol.Envelope) {
	var msg protocol.MLSKeyPackageFetch
	if err := proto.Unmarshal(env.Payload, &msg); err != nil {
		c.sendError(env, 3001, "Invalid mls.key_package.fetch payload", false)
		return
	}

	data, err := c.mlsService.FetchKeyPackage(ctx, msg.UserId)
	if err != nil {
		if errors.Is(err, mls.ErrNoKeyPackage) {
			c.sendError(env, 5005, "No key package available for user", false)
		} else {
			log.Printf("[%s] fetch key package error: %v", c.id, err)
			c.sendError(env, 9001, "Failed to fetch key package", false)
		}
		return
	}

	resp := &protocol.MLSKeyPackageResponse{
		UserId:         msg.UserId,
		KeyPackageData: data,
	}
	c.sendTypedResponse(env, protocol.MessageType_MLS_KEY_PACKAGE_RESPONSE, resp)
}

func (c *Conn) handleMLSWelcome(ctx context.Context, env *protocol.Envelope) {
	var msg protocol.MLSWelcome
	if err := proto.Unmarshal(env.Payload, &msg); err != nil {
		c.sendError(env, 3001, "Invalid mls.welcome payload", false)
		return
	}

	// Forward the Welcome to the recipient.
	welcomeReceive := &protocol.MLSWelcomeReceive{
		ConversationId: msg.ConversationId,
		SenderId:       c.userID,
		WelcomeData:    msg.WelcomeData,
	}
	receivePayload, err := proto.Marshal(welcomeReceive)
	if err != nil {
		log.Printf("[%s] marshal welcome receive error: %v", c.id, err)
		return
	}
	welcomeEnv := &protocol.Envelope{
		Type:    protocol.MessageType_MLS_WELCOME_RECEIVE,
		Payload: receivePayload,
	}
	c.hub.SendToUser(msg.RecipientId, welcomeEnv)
}

func (c *Conn) handleMLSCommit(ctx context.Context, env *protocol.Envelope) {
	var msg protocol.MLSCommit
	if err := proto.Unmarshal(env.Payload, &msg); err != nil {
		c.sendError(env, 3001, "Invalid mls.commit payload", false)
		return
	}

	// Validate membership.
	isMember, err := c.store.IsUserMember(ctx, msg.ConversationId, c.userID)
	if err != nil {
		log.Printf("[%s] membership check error: %v", c.id, err)
		c.sendError(env, 9001, "Internal error", false)
		return
	}
	if !isMember {
		c.sendError(env, 4001, "Not a member of this conversation", false)
		return
	}

	// Broadcast to all group members except sender.
	commitBroadcast := &protocol.MLSCommitBroadcast{
		ConversationId: msg.ConversationId,
		SenderId:       c.userID,
		CommitData:     msg.CommitData,
	}
	broadcastPayload, err := proto.Marshal(commitBroadcast)
	if err != nil {
		log.Printf("[%s] marshal commit broadcast error: %v", c.id, err)
		return
	}
	broadcastEnv := &protocol.Envelope{
		Type:    protocol.MessageType_MLS_COMMIT_BROADCAST,
		Payload: broadcastPayload,
	}

	members, err := c.store.GetMembers(ctx, msg.ConversationId)
	if err != nil {
		log.Printf("[%s] get members error: %v", c.id, err)
		return
	}
	memberIDs := make([]string, len(members))
	for i, m := range members {
		memberIDs[i] = m.UserID
	}
	c.hub.BroadcastToGroup(memberIDs, broadcastEnv, c.userID)
}

// ============================================================================
// Offline Delivery
// ============================================================================

// deliverPendingMessages sends all pending messages to the user on connect.
func (c *Conn) deliverPendingMessages(ctx context.Context) {
	msgs, err := c.store.GetPendingMessages(ctx, c.userID)
	if err != nil {
		log.Printf("[%s] get pending messages error: %v", c.id, err)
		return
	}

	for _, m := range msgs {
		receiveMsg := &protocol.MessageReceive{
			MessageId:        m.ID,
			ConversationId:   m.GroupID,
			SenderId:         m.SenderID,
			EncryptedPayload: m.Payload,
			ServerTimestamp:  m.ServerTimestamp,
		}
		c.sendTypedResponse(nil, protocol.MessageType_MESSAGE_RECEIVE, receiveMsg)

		// Mark as delivered.
		if err := c.store.UpdateDeliveryStatus(ctx, m.ID, c.userID, store.DeliveryDelivered); err != nil {
			log.Printf("[%s] update delivery status for %s error: %v", c.id, m.ID, err)
		}
	}

	if len(msgs) > 0 {
		log.Printf("[%s] Delivered %d pending messages to user %s", c.id, len(msgs), c.userID)
	}
}

// ============================================================================
// Auth Message Handlers
// ============================================================================

func (c *Conn) handleAuthRequest(ctx context.Context, env *protocol.Envelope) {
	var req protocol.AuthRequest
	if err := proto.Unmarshal(env.Payload, &req); err != nil {
		log.Printf("[%s] Failed to unmarshal auth request: %v", c.id, err)
		c.sendAuthError(env, 3001, "Invalid auth request payload")
		return
	}

	// Try session token reconnection: the client may send a session token
	// in the username field for reconnection without a WebAuthn ceremony.
	info, err := c.authService.ValidateSession(ctx, req.Username)
	if err == nil {
		// Valid session token — skip WebAuthn ceremony.
		if !c.transitionToReady(ctx, info.UserID, info.Username) {
			return // auth timer already fired
		}
		c.sendAuthSuccess(env, "", info.UserID, info.Username, info.DisplayName)
		log.Printf("[%s] Session token reconnection for user %s", c.id, info.Username)
		return
	}

	// Not a valid session token — proceed with normal WebAuthn login.
	challenge, err := c.authService.BeginLogin(ctx, req.Username)
	if err != nil {
		c.handleAuthError(env, err)
		return
	}

	c.challengeID = challenge.ChallengeID

	// Send AUTH_CHALLENGE.
	challengeMsg := &protocol.AuthChallenge{
		Challenge:                challenge.CredentialRequestOptions, // raw options JSON
		CredentialRequestOptions: challenge.CredentialRequestOptions,
	}
	c.sendTypedResponse(env, protocol.MessageType_AUTH_CHALLENGE, challengeMsg)
}

func (c *Conn) handleAuthResponse(ctx context.Context, env *protocol.Envelope) {
	if c.challengeID == "" {
		c.sendAuthError(env, 3002, "No active login challenge")
		return
	}

	var resp protocol.AuthResponse
	if err := proto.Unmarshal(env.Payload, &resp); err != nil {
		log.Printf("[%s] Failed to unmarshal auth response: %v", c.id, err)
		c.sendAuthError(env, 3001, "Invalid auth response payload")
		return
	}

	assertion := &auth.AssertionResponse{
		CredentialID:      resp.CredentialId,
		AuthenticatorData: resp.AuthenticatorData,
		ClientDataJSON:    resp.ClientDataJson,
		Signature:         resp.Signature,
	}

	result, err := c.authService.FinishLogin(ctx, c.challengeID, assertion)
	c.challengeID = ""
	if err != nil {
		c.handleAuthError(env, err)
		return
	}

	if !c.transitionToReady(ctx, result.UserID, result.Username) {
		return
	}
	c.sendAuthSuccess(env, result.Token, result.UserID, result.Username, result.DisplayName)
	log.Printf("[%s] Login successful for user %s", c.id, result.Username)
}

func (c *Conn) handleAuthRegisterRequest(ctx context.Context, env *protocol.Envelope) {
	var req protocol.AuthRegisterRequest
	if err := proto.Unmarshal(env.Payload, &req); err != nil {
		log.Printf("[%s] Failed to unmarshal register request: %v", c.id, err)
		c.sendAuthError(env, 3001, "Invalid register request payload")
		return
	}

	challenge, err := c.authService.BeginRegistration(ctx, req.Username, req.DisplayName)
	if err != nil {
		c.handleAuthError(env, err)
		return
	}

	c.challengeID = challenge.ChallengeID

	// Send AUTH_REGISTER_CHALLENGE.
	challengeMsg := &protocol.AuthRegisterChallenge{
		Challenge:                 challenge.CredentialCreationOptions,
		CredentialCreationOptions: challenge.CredentialCreationOptions,
	}
	c.sendTypedResponse(env, protocol.MessageType_AUTH_REGISTER_CHALLENGE, challengeMsg)
}

func (c *Conn) handleAuthRegisterResponse(ctx context.Context, env *protocol.Envelope) {
	if c.challengeID == "" {
		c.sendAuthError(env, 3002, "No active registration challenge")
		return
	}

	var resp protocol.AuthRegisterResponse
	if err := proto.Unmarshal(env.Payload, &resp); err != nil {
		log.Printf("[%s] Failed to unmarshal register response: %v", c.id, err)
		c.sendAuthError(env, 3001, "Invalid register response payload")
		return
	}

	attestation := &auth.AttestationResponse{
		CredentialID:      resp.CredentialId,
		AuthenticatorData: resp.AuthenticatorData,
		ClientDataJSON:    resp.ClientDataJson,
		AttestationObject: resp.AttestationObject,
	}

	result, err := c.authService.FinishRegistration(ctx, c.challengeID, attestation)
	c.challengeID = ""
	if err != nil {
		c.handleAuthError(env, err)
		return
	}

	if !c.transitionToReady(ctx, result.UserID, result.Username) {
		return
	}

	// Send AUTH_REGISTER_SUCCESS.
	success := &protocol.AuthRegisterSuccess{
		UserId:       result.UserID,
		SessionToken: result.Token,
	}
	c.sendTypedResponse(env, protocol.MessageType_AUTH_REGISTER_SUCCESS, success)
	log.Printf("[%s] Registration successful for user %s", c.id, result.Username)
}

// ============================================================================
// Auth Helpers
// ============================================================================

// transitionToReady atomically transitions from authenticating to ready.
// Returns false if the transition failed (e.g., auth timer already fired).
func (c *Conn) transitionToReady(ctx context.Context, userID, username string) bool {
	if !c.state.CompareAndSwap(stateAuthenticating, stateReady) {
		return false
	}
	c.authTimer.Stop()
	c.userID = userID
	c.username = username
	c.hub.SetAuthenticated(c, userID)

	// Deliver pending messages after successful authentication.
	go c.deliverPendingMessages(ctx)

	return true
}

// handleAuthError sends an appropriate AUTH_ERROR based on the error type.
// Fatal errors also close the WebSocket connection.
func (c *Conn) handleAuthError(env *protocol.Envelope, err error) {
	switch {
	case errors.Is(err, auth.ErrAccountDisabled):
		c.sendAuthError(env, 2004, "Account disabled")
		c.ws.Close(websocket.StatusCode(4005), "Account Disabled")
		c.close()
	case errors.Is(err, auth.ErrSessionExpired):
		c.sendAuthError(env, 1002, "Session expired")
		c.ws.Close(websocket.StatusCode(4004), "Session Expired")
		c.close()
	case errors.Is(err, auth.ErrUserNotFound):
		c.sendAuthError(env, 1001, "Invalid credential")
	case errors.Is(err, auth.ErrInvalidCredential):
		c.sendAuthError(env, 1001, "Invalid credential")
	case errors.Is(err, auth.ErrChallengeExpired):
		c.sendAuthError(env, 1004, "Challenge expired")
	case errors.Is(err, auth.ErrChallengeNotFound):
		c.sendAuthError(env, 1004, "Challenge not found")
	case errors.Is(err, auth.ErrCloneDetected):
		c.sendAuthError(env, 1001, "Credential clone detected")
		c.ws.Close(websocket.StatusCode(4002), "Authentication Failed")
		c.close()
	case errors.Is(err, auth.ErrRegistrationFailed):
		c.sendAuthError(env, 1003, "Registration failed")
	default:
		log.Printf("[%s] Auth error: %v", c.id, err)
		c.sendAuthError(env, 9001, "Internal error")
	}
}

func (c *Conn) sendAuthError(origEnv *protocol.Envelope, code int32, message string) {
	errMsg := &protocol.AuthError{
		ErrorCode: code,
		Message:   message,
	}
	c.sendTypedResponse(origEnv, protocol.MessageType_AUTH_ERROR, errMsg)
}

func (c *Conn) sendAuthSuccess(origEnv *protocol.Envelope, sessionToken, userID, username, displayName string) {
	success := &protocol.AuthSuccess{
		SessionToken: sessionToken,
		UserId:       userID,
		Username:     username,
		DisplayName:  displayName,
	}
	c.sendTypedResponse(origEnv, protocol.MessageType_AUTH_SUCCESS, success)
}

// sendTypedResponse marshals a protobuf message and sends it in an envelope.
func (c *Conn) sendTypedResponse(origEnv *protocol.Envelope, msgType protocol.MessageType, msg proto.Message) {
	payload, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("[%s] Failed to marshal %s: %v", c.id, msgType, err)
		return
	}
	requestID := ""
	if origEnv != nil {
		requestID = origEnv.RequestId
	}
	env := &protocol.Envelope{
		Type:      msgType,
		RequestId: requestID,
		Payload:   payload,
	}
	c.sendEnvelope(env)
}

// ============================================================================
// System Message Handlers
// ============================================================================

// handlePing responds to a PING with a PONG echoing the timestamp.
func (c *Conn) handlePing(env *protocol.Envelope) {
	var ping protocol.Ping
	if err := proto.Unmarshal(env.Payload, &ping); err != nil {
		log.Printf("[%s] Failed to unmarshal ping: %v", c.id, err)
		c.sendError(env, 3001, "Invalid ping payload", false)
		return
	}

	pong := &protocol.Pong{
		Timestamp: ping.Timestamp,
	}

	c.sendTypedResponse(env, protocol.MessageType_PONG, pong)
}

// sendEnvelope serializes and queues an envelope for sending.
func (c *Conn) sendEnvelope(env *protocol.Envelope) {
	data, err := proto.Marshal(env)
	if err != nil {
		log.Printf("[%s] Failed to marshal envelope: %v", c.id, err)
		return
	}

	select {
	case c.send <- data:
	default:
		log.Printf("[%s] Send buffer full, dropping message", c.id)
	}
}

// sendError sends an error envelope to the client.
func (c *Conn) sendError(origEnv *protocol.Envelope, code int32, message string, fatal bool) {
	errMsg := &protocol.Error{
		Code:    code,
		Message: message,
		Fatal:   fatal,
	}
	c.sendTypedResponse(origEnv, protocol.MessageType_ERROR, errMsg)
}

// close cancels the connection context, closing both pumps.
func (c *Conn) close() {
	c.once.Do(func() {
		c.state.Store(stateDisconnected)
		c.cancel()
	})
}

// connID generates a unique connection ID.
func connID() string {
	return fmt.Sprintf("conn-%d", time.Now().UnixNano())
}
