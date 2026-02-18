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
	"github.com/sovereign-im/sovereign/server/internal/protocol"
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
}

// NewConn creates a new Conn.
func NewConn(id string, ws *websocket.Conn, hub *Hub, maxMessageSize int, authService *auth.Service) *Conn {
	c := &Conn{
		id:             id,
		ws:             ws,
		hub:            hub,
		send:           make(chan []byte, 256),
		maxMessageSize: int64(maxMessageSize),
		authService:    authService,
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
		c.handleReadyMessage(env)
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
func (c *Conn) handleReadyMessage(env *protocol.Envelope) {
	switch env.Type {
	case protocol.MessageType_PING:
		c.handlePing(env)
	case protocol.MessageType_ERROR:
		log.Printf("[%s] Received error message, discarding", c.id)
	default:
		// Phase B: echo for now, real routing in Phase C.
		c.echoEnvelope(env)
	}
}

// --- Auth Message Handlers ---

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
		if !c.transitionToReady(info.UserID, info.Username) {
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

	if !c.transitionToReady(result.UserID, result.Username) {
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

	if !c.transitionToReady(result.UserID, result.Username) {
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

// --- Auth Helpers ---

// transitionToReady atomically transitions from authenticating to ready.
// Returns false if the transition failed (e.g., auth timer already fired).
func (c *Conn) transitionToReady(userID, username string) bool {
	if !c.state.CompareAndSwap(stateAuthenticating, stateReady) {
		return false
	}
	c.authTimer.Stop()
	c.userID = userID
	c.username = username
	c.hub.SetAuthenticated(c, userID)
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

// --- Existing Message Handlers ---

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

// echoEnvelope sends the envelope back to the sender.
func (c *Conn) echoEnvelope(env *protocol.Envelope) {
	c.sendEnvelope(env)
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
