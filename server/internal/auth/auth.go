package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/sovereign-im/sovereign/server/internal/store"
)

// Sentinel errors for authentication operations.
var (
	ErrChallengeExpired  = errors.New("challenge expired")
	ErrChallengeNotFound = errors.New("challenge not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrAccountDisabled   = errors.New("account disabled")
	ErrSessionExpired    = errors.New("session expired")
	ErrCloneDetected     = errors.New("sign count did not increase: possible credential clone")
	ErrInvalidCredential = errors.New("invalid credential")
	ErrRegistrationFailed = errors.New("registration failed")
)

const (
	// DefaultSessionDuration is the default session lifetime (30 days).
	DefaultSessionDuration = 30 * 24 * time.Hour

	// RegistrationChallengeTTL is how long a registration challenge is valid.
	RegistrationChallengeTTL = 60 * time.Second

	// LoginChallengeTTL is how long a login challenge is valid.
	LoginChallengeTTL = 30 * time.Second

	// SessionTokenBytes is the number of random bytes in a session token.
	SessionTokenBytes = 32
)

// Service handles WebAuthn/passkey authentication.
type Service struct {
	store    *store.Store
	webauthn *webauthn.WebAuthn
}

// NewService creates a new auth service with the given store and WebAuthn config.
func NewService(s *store.Store, rpDisplayName, rpID string, rpOrigins []string) (*Service, error) {
	wconfig := &webauthn.Config{
		RPDisplayName: rpDisplayName,
		RPID:          rpID,
		RPOrigins:     rpOrigins,
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("create webauthn: %w", err)
	}

	return &Service{
		store:    s,
		webauthn: w,
	}, nil
}

// RegistrationChallenge is returned by BeginRegistration.
type RegistrationChallenge struct {
	ChallengeID               string
	CredentialCreationOptions []byte // serialized JSON of WebAuthn creation options
}

// LoginChallenge is returned by BeginLogin.
type LoginChallenge struct {
	ChallengeID              string
	CredentialRequestOptions []byte // serialized JSON of WebAuthn request options
}

// AttestationResponse holds the client's registration response fields.
type AttestationResponse struct {
	CredentialID      []byte
	AuthenticatorData []byte
	ClientDataJSON    []byte
	AttestationObject []byte
}

// AssertionResponse holds the client's login assertion fields.
type AssertionResponse struct {
	CredentialID      []byte
	AuthenticatorData []byte
	ClientDataJSON    []byte
	Signature         []byte
}

// SessionResult is returned after successful authentication.
type SessionResult struct {
	Token       string // raw session token (base64url encoded)
	UserID      string
	Username    string
	DisplayName string
}

// SessionInfo is returned by ValidateSession.
type SessionInfo struct {
	SessionID   string
	UserID      string
	Username    string
	DisplayName string
}

// challengePayload is stored in the challenge table's challenge_data column.
type challengePayload struct {
	SessionData webauthn.SessionData `json:"session_data"`
	DisplayName string               `json:"display_name,omitempty"`
}

// --- Registration Flow ---

// BeginRegistration starts a WebAuthn registration ceremony.
// Returns credential creation options and a challenge ID for correlation.
func (svc *Service) BeginRegistration(ctx context.Context, username, displayName string) (*RegistrationChallenge, error) {
	// Check if username is already taken
	_, err := svc.store.GetUserByUsername(ctx, username)
	if err == nil {
		return nil, fmt.Errorf("username %q already taken: %w", username, ErrRegistrationFailed)
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("check username: %w", err)
	}

	// Create temporary user for the ceremony
	userID := uuid.New().String()
	user := &webauthnUser{
		id:          []byte(userID),
		name:        username,
		displayName: displayName,
	}

	// Generate credential creation options
	options, sessionData, err := svc.webauthn.BeginRegistration(user)
	if err != nil {
		return nil, fmt.Errorf("begin registration: %w", err)
	}

	// Serialize session data for storage
	payload := challengePayload{
		SessionData: *sessionData,
		DisplayName: displayName,
	}
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal challenge payload: %w", err)
	}

	// Store challenge
	challengeID := uuid.New().String()
	now := time.Now()
	challenge := &store.Challenge{
		ChallengeID:   challengeID,
		ChallengeData: payloadData,
		Username:      username,
		ChallengeType: "registration",
		CreatedAt:     now.Unix(),
		ExpiresAt:     now.Add(RegistrationChallengeTTL).Unix(),
	}
	if err := svc.store.CreateChallenge(ctx, challenge); err != nil {
		return nil, fmt.Errorf("store challenge: %w", err)
	}

	// Serialize options for the client
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("marshal options: %w", err)
	}

	return &RegistrationChallenge{
		ChallengeID:               challengeID,
		CredentialCreationOptions: optionsJSON,
	}, nil
}

// FinishRegistration completes the WebAuthn registration ceremony.
// Creates the user, credential, and session. Returns a session token.
func (svc *Service) FinishRegistration(ctx context.Context, challengeID string, resp *AttestationResponse) (*SessionResult, error) {
	// Retrieve and validate challenge
	challenge, err := svc.store.GetChallenge(ctx, challengeID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}
		return nil, fmt.Errorf("get challenge: %w", err)
	}

	// Delete challenge (single-use) regardless of outcome
	_ = svc.store.DeleteChallenge(ctx, challengeID)

	// Check expiry
	if time.Now().Unix() > challenge.ExpiresAt {
		return nil, ErrChallengeExpired
	}

	// Deserialize challenge payload
	var payload challengePayload
	if err := json.Unmarshal(challenge.ChallengeData, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal challenge payload: %w", err)
	}

	// Reconstruct user for the library
	user := &webauthnUser{
		id:          payload.SessionData.UserID,
		name:        challenge.Username,
		displayName: payload.DisplayName,
	}

	// Build WebAuthn response JSON and wrap in HTTP request for the library
	responseJSON, err := buildRegistrationResponseJSON(resp)
	if err != nil {
		return nil, fmt.Errorf("build response JSON: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(responseJSON))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Validate with the WebAuthn library
	credential, err := svc.webauthn.FinishRegistration(user, payload.SessionData, httpReq)
	if err != nil {
		return nil, fmt.Errorf("finish registration: %w", err)
	}

	// Persist user, credential, and session
	userID := string(payload.SessionData.UserID)
	now := time.Now().Unix()

	storeUser := &store.User{
		ID:          userID,
		Username:    challenge.Username,
		DisplayName: payload.DisplayName,
		Role:        "member",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := svc.store.CreateUser(ctx, storeUser); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	credID := uuid.New().String()
	storeCred := &store.Credential{
		ID:           credID,
		UserID:       userID,
		CredentialID: credential.ID,
		PublicKey:    credential.PublicKey,
		SignCount:    int64(credential.Authenticator.SignCount),
		CreatedAt:    now,
	}
	if err := svc.store.CreateCredential(ctx, storeCred); err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}

	token, tokenHash, err := generateSession()
	if err != nil {
		return nil, fmt.Errorf("generate session: %w", err)
	}

	sessID := uuid.New().String()
	storeSession := &store.Session{
		ID:           sessID,
		UserID:       userID,
		CredentialID: credID,
		TokenHash:    tokenHash,
		CreatedAt:    now,
		ExpiresAt:    now + int64(DefaultSessionDuration.Seconds()),
		LastSeenAt:   now,
	}
	if err := svc.store.CreateSession(ctx, storeSession); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &SessionResult{
		Token:       token,
		UserID:      userID,
		Username:    challenge.Username,
		DisplayName: payload.DisplayName,
	}, nil
}

// --- Login Flow ---

// BeginLogin starts a WebAuthn login ceremony for the given username.
// Returns credential request options and a challenge ID.
func (svc *Service) BeginLogin(ctx context.Context, username string) (*LoginChallenge, error) {
	// Look up user
	user, err := svc.store.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if !user.Enabled {
		return nil, ErrAccountDisabled
	}

	// Get user's credentials
	creds, err := svc.store.GetCredentialsByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}
	if len(creds) == 0 {
		return nil, fmt.Errorf("user %q has no credentials: %w", username, ErrInvalidCredential)
	}

	waUser := newWebAuthnUser(user, creds)

	// Generate credential request options
	options, sessionData, err := svc.webauthn.BeginLogin(waUser)
	if err != nil {
		return nil, fmt.Errorf("begin login: %w", err)
	}

	// Store session data as challenge
	payloadData, err := json.Marshal(challengePayload{SessionData: *sessionData})
	if err != nil {
		return nil, fmt.Errorf("marshal challenge payload: %w", err)
	}

	challengeID := uuid.New().String()
	now := time.Now()
	challenge := &store.Challenge{
		ChallengeID:   challengeID,
		ChallengeData: payloadData,
		Username:      username,
		ChallengeType: "login",
		CreatedAt:     now.Unix(),
		ExpiresAt:     now.Add(LoginChallengeTTL).Unix(),
	}
	if err := svc.store.CreateChallenge(ctx, challenge); err != nil {
		return nil, fmt.Errorf("store challenge: %w", err)
	}

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("marshal options: %w", err)
	}

	return &LoginChallenge{
		ChallengeID:              challengeID,
		CredentialRequestOptions: optionsJSON,
	}, nil
}

// FinishLogin completes the WebAuthn login ceremony.
// Validates the assertion, updates sign count, and creates a session.
func (svc *Service) FinishLogin(ctx context.Context, challengeID string, resp *AssertionResponse) (*SessionResult, error) {
	// Retrieve and validate challenge
	challenge, err := svc.store.GetChallenge(ctx, challengeID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}
		return nil, fmt.Errorf("get challenge: %w", err)
	}

	// Delete challenge (single-use)
	_ = svc.store.DeleteChallenge(ctx, challengeID)

	if time.Now().Unix() > challenge.ExpiresAt {
		return nil, ErrChallengeExpired
	}

	// Deserialize challenge payload
	var payload challengePayload
	if err := json.Unmarshal(challenge.ChallengeData, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal challenge payload: %w", err)
	}

	// Look up user and credentials
	user, err := svc.store.GetUserByUsername(ctx, challenge.Username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if !user.Enabled {
		return nil, ErrAccountDisabled
	}

	creds, err := svc.store.GetCredentialsByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}

	waUser := newWebAuthnUser(user, creds)

	// Build WebAuthn assertion response JSON
	responseJSON, err := buildAssertionResponseJSON(resp)
	if err != nil {
		return nil, fmt.Errorf("build response JSON: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(responseJSON))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Validate with the WebAuthn library
	credential, err := svc.webauthn.FinishLogin(waUser, payload.SessionData, httpReq)
	if err != nil {
		return nil, fmt.Errorf("finish login: %w", err)
	}

	// Check for credential cloning (sign count didn't increase)
	if credential.Authenticator.CloneWarning {
		return nil, ErrCloneDetected
	}

	// Find the matching store credential and update sign count
	for _, c := range creds {
		if bytes.Equal(c.CredentialID, credential.ID) {
			if err := svc.store.UpdateSignCount(ctx, c.ID, int64(credential.Authenticator.SignCount)); err != nil {
				return nil, fmt.Errorf("update sign count: %w", err)
			}
			break
		}
	}

	// Generate session
	token, tokenHash, err := generateSession()
	if err != nil {
		return nil, fmt.Errorf("generate session: %w", err)
	}

	now := time.Now().Unix()
	sessID := uuid.New().String()
	storeSession := &store.Session{
		ID:         sessID,
		UserID:     user.ID,
		TokenHash:  tokenHash,
		CreatedAt:  now,
		ExpiresAt:  now + int64(DefaultSessionDuration.Seconds()),
		LastSeenAt: now,
	}
	if err := svc.store.CreateSession(ctx, storeSession); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &SessionResult{
		Token:       token,
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	}, nil
}

// --- Session Management ---

// ValidateSession validates a raw session token. Returns user info if valid.
// Updates the session's last_seen_at timestamp.
func (svc *Service) ValidateSession(ctx context.Context, token string) (*SessionInfo, error) {
	tokenHash := hashSessionToken(token)

	sess, err := svc.store.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrInvalidCredential
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	if time.Now().Unix() > sess.ExpiresAt {
		_ = svc.store.DeleteSession(ctx, sess.ID)
		return nil, ErrSessionExpired
	}

	user, err := svc.store.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if !user.Enabled {
		return nil, ErrAccountDisabled
	}

	_ = svc.store.UpdateSessionLastUsed(ctx, sess.ID)

	return &SessionInfo{
		SessionID:   sess.ID,
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	}, nil
}

// RevokeSession deletes a session by its ID.
func (svc *Service) RevokeSession(ctx context.Context, sessionID string) error {
	return svc.store.DeleteSession(ctx, sessionID)
}

// --- Helpers ---

// generateSession creates a new random session token and its SHA-256 hash.
func generateSession() (token string, tokenHash []byte, err error) {
	b := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("generate random bytes: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(token))
	return token, h[:], nil
}

// hashSessionToken computes the SHA-256 hash of a session token.
func hashSessionToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// buildRegistrationResponseJSON constructs the WebAuthn credential creation
// response JSON from individual protobuf fields.
func buildRegistrationResponseJSON(resp *AttestationResponse) ([]byte, error) {
	credIDStr := base64.RawURLEncoding.EncodeToString(resp.CredentialID)
	m := map[string]interface{}{
		"id":    credIDStr,
		"rawId": credIDStr,
		"type":  "public-key",
		"response": map[string]interface{}{
			"attestationObject": base64.RawURLEncoding.EncodeToString(resp.AttestationObject),
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(resp.ClientDataJSON),
		},
	}
	return json.Marshal(m)
}

// buildAssertionResponseJSON constructs the WebAuthn assertion response JSON
// from individual protobuf fields.
func buildAssertionResponseJSON(resp *AssertionResponse) ([]byte, error) {
	credIDStr := base64.RawURLEncoding.EncodeToString(resp.CredentialID)
	m := map[string]interface{}{
		"id":    credIDStr,
		"rawId": credIDStr,
		"type":  "public-key",
		"response": map[string]interface{}{
			"authenticatorData": base64.RawURLEncoding.EncodeToString(resp.AuthenticatorData),
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(resp.ClientDataJSON),
			"signature":         base64.RawURLEncoding.EncodeToString(resp.Signature),
		},
	}
	return json.Marshal(m)
}
