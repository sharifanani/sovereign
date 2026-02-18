package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/sovereign-im/sovereign/server/internal/store"
)

// newTestService creates an auth Service backed by an in-memory store.
func newTestService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New(:memory:) error: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	svc, err := NewService(s, "Test Server", "localhost", []string{"http://localhost:8080"})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}
	return svc, s
}

// seedUser creates a user and credential in the store for login tests.
func seedUser(t *testing.T, s *store.Store, userID, username, displayName string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().Unix()

	u := &store.User{
		ID:          userID,
		Username:    username,
		DisplayName: displayName,
		Role:        "member",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.CreateUser(ctx, u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	c := &store.Credential{
		ID:           "cred-" + userID,
		UserID:       userID,
		CredentialID: []byte("webauthn-cred-id-" + userID),
		PublicKey:    []byte("fake-public-key-" + userID),
		SignCount:    0,
		CreatedAt:    now,
	}
	if err := s.CreateCredential(ctx, c); err != nil {
		t.Fatalf("CreateCredential: %v", err)
	}
}

// seedSession creates a valid session for the given user.
func seedSession(t *testing.T, s *store.Store, sessID, userID, token string, expiresAt int64) {
	t.Helper()
	ctx := context.Background()
	h := sha256.Sum256([]byte(token))
	now := time.Now().Unix()

	sess := &store.Session{
		ID:         sessID,
		UserID:     userID,
		TokenHash:  h[:],
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		LastSeenAt: now,
	}
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name          string
		rpDisplayName string
		rpID          string
		rpOrigins     []string
		wantErr       bool
	}{
		{
			name:          "valid config",
			rpDisplayName: "Test Server",
			rpID:          "localhost",
			rpOrigins:     []string{"http://localhost:8080"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := store.New(":memory:")
			if err != nil {
				t.Fatalf("store.New: %v", err)
			}
			defer s.Close()

			svc, err := NewService(s, tt.rpDisplayName, tt.rpID, tt.rpOrigins)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if svc == nil {
				t.Fatal("service is nil")
			}
		})
	}
}

func TestBeginRegistration(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		displayName string
		seedUser    bool // whether to create the user first (to test duplicate)
		wantErr     bool
	}{
		{
			name:        "success",
			username:    "alice",
			displayName: "Alice Wonderland",
		},
		{
			name:        "duplicate username fails",
			username:    "alice",
			displayName: "Alice Wonderland",
			seedUser:    true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, s := newTestService(t)
			ctx := context.Background()

			if tt.seedUser {
				seedUser(t, s, "existing-user", tt.username, tt.displayName)
			}

			result, err := svc.BeginRegistration(ctx, tt.username, tt.displayName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ChallengeID == "" {
				t.Error("ChallengeID is empty")
			}
			if len(result.CredentialCreationOptions) == 0 {
				t.Error("CredentialCreationOptions is empty")
			}

			// Verify challenge was stored
			challenge, err := s.GetChallenge(ctx, result.ChallengeID)
			if err != nil {
				t.Fatalf("GetChallenge: %v", err)
			}
			if challenge.ChallengeType != "registration" {
				t.Errorf("ChallengeType = %q, want %q", challenge.ChallengeType, "registration")
			}
			if challenge.Username != tt.username {
				t.Errorf("Username = %q, want %q", challenge.Username, tt.username)
			}
		})
	}
}

func TestFinishRegistrationErrors(t *testing.T) {
	tests := []struct {
		name        string
		challengeID string
		wantErr     error
	}{
		{
			name:        "invalid challenge ID",
			challengeID: "nonexistent",
			wantErr:     ErrChallengeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t)
			ctx := context.Background()

			resp := &AttestationResponse{
				CredentialID:      []byte("cred-id"),
				AuthenticatorData: []byte("auth-data"),
				ClientDataJSON:    []byte("{}"),
				AttestationObject: []byte("attest"),
			}

			_, err := svc.FinishRegistration(ctx, tt.challengeID, resp)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestFinishRegistrationExpiredChallenge(t *testing.T) {
	svc, s := newTestService(t)
	ctx := context.Background()

	// Create an expired challenge
	now := time.Now()
	challenge := &store.Challenge{
		ChallengeID:   "expired-ch",
		ChallengeData: []byte(`{"session_data":{"challenge":"dGVzdA","user_id":"dXNlcg","allowed_credentials":null,"user_verification":"preferred","extensions":null}}`),
		Username:      "alice",
		ChallengeType: "registration",
		CreatedAt:     now.Add(-120 * time.Second).Unix(),
		ExpiresAt:     now.Add(-60 * time.Second).Unix(), // expired 60 seconds ago
	}
	if err := s.CreateChallenge(ctx, challenge); err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}

	resp := &AttestationResponse{
		CredentialID:      []byte("cred-id"),
		AuthenticatorData: []byte("auth-data"),
		ClientDataJSON:    []byte("{}"),
		AttestationObject: []byte("attest"),
	}

	_, err := svc.FinishRegistration(ctx, "expired-ch", resp)
	if !errors.Is(err, ErrChallengeExpired) {
		t.Errorf("error = %v, want ErrChallengeExpired", err)
	}
}

func TestBeginLogin(t *testing.T) {
	tests := []struct {
		name     string
		username string
		seedUser bool
		wantErr  error
	}{
		{
			name:     "existing user",
			username: "alice",
			seedUser: true,
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			wantErr:  ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, s := newTestService(t)
			ctx := context.Background()

			if tt.seedUser {
				seedUser(t, s, "u1", tt.username, "Display "+tt.username)
			}

			result, err := svc.BeginLogin(ctx, tt.username)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ChallengeID == "" {
				t.Error("ChallengeID is empty")
			}
			if len(result.CredentialRequestOptions) == 0 {
				t.Error("CredentialRequestOptions is empty")
			}
		})
	}
}

func TestBeginLoginDisabledUser(t *testing.T) {
	svc, s := newTestService(t)
	ctx := context.Background()

	// Create a disabled user
	seedUser(t, s, "u1", "alice", "Alice")
	u, _ := s.GetUserByUsername(ctx, "alice")
	u.Enabled = false
	u.UpdatedAt = time.Now().Unix()
	if err := s.UpdateUser(ctx, u); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	_, err := svc.BeginLogin(ctx, "alice")
	if !errors.Is(err, ErrAccountDisabled) {
		t.Errorf("error = %v, want ErrAccountDisabled", err)
	}
}

func TestFinishLoginErrors(t *testing.T) {
	tests := []struct {
		name        string
		challengeID string
		wantErr     error
	}{
		{
			name:        "invalid challenge ID",
			challengeID: "nonexistent",
			wantErr:     ErrChallengeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t)
			ctx := context.Background()

			resp := &AssertionResponse{
				CredentialID:      []byte("cred-id"),
				AuthenticatorData: []byte("auth-data"),
				ClientDataJSON:    []byte("{}"),
				Signature:         []byte("sig"),
			}

			_, err := svc.FinishLogin(ctx, tt.challengeID, resp)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSession(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		setup   func(t *testing.T, s *store.Store)
		wantErr error
	}{
		{
			name:  "valid token",
			token: "valid-token-123",
			setup: func(t *testing.T, s *store.Store) {
				seedUser(t, s, "u1", "alice", "Alice")
				seedSession(t, s, "s1", "u1", "valid-token-123", time.Now().Add(24*time.Hour).Unix())
			},
		},
		{
			name:  "expired token",
			token: "expired-token",
			setup: func(t *testing.T, s *store.Store) {
				seedUser(t, s, "u1", "alice", "Alice")
				seedSession(t, s, "s1", "u1", "expired-token", time.Now().Add(-1*time.Hour).Unix())
			},
			wantErr: ErrSessionExpired,
		},
		{
			name:    "invalid token (not in DB)",
			token:   "unknown-token",
			wantErr: ErrInvalidCredential,
		},
		{
			name:  "disabled user",
			token: "disabled-user-token",
			setup: func(t *testing.T, s *store.Store) {
				seedUser(t, s, "u1", "alice", "Alice")
				seedSession(t, s, "s1", "u1", "disabled-user-token", time.Now().Add(24*time.Hour).Unix())
				ctx := context.Background()
				u, _ := s.GetUserByID(ctx, "u1")
				u.Enabled = false
				u.UpdatedAt = time.Now().Unix()
				s.UpdateUser(ctx, u)
			},
			wantErr: ErrAccountDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, s := newTestService(t)
			ctx := context.Background()

			if tt.setup != nil {
				tt.setup(t, s)
			}

			info, err := svc.ValidateSession(ctx, tt.token)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if info.UserID != "u1" {
				t.Errorf("UserID = %q, want %q", info.UserID, "u1")
			}
			if info.Username != "alice" {
				t.Errorf("Username = %q, want %q", info.Username, "alice")
			}
			if info.DisplayName != "Alice" {
				t.Errorf("DisplayName = %q, want %q", info.DisplayName, "Alice")
			}
		})
	}
}

func TestRevokeSession(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		setup     bool
		wantErr   error
	}{
		{
			name:      "success",
			sessionID: "s1",
			setup:     true,
		},
		{
			name:      "not found",
			sessionID: "nonexistent",
			wantErr:   store.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, s := newTestService(t)
			ctx := context.Background()

			if tt.setup {
				seedUser(t, s, "u1", "alice", "Alice")
				seedSession(t, s, "s1", "u1", "token-1", time.Now().Add(24*time.Hour).Unix())
			}

			err := svc.RevokeSession(ctx, tt.sessionID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify session was revoked
			_, err = svc.ValidateSession(ctx, "token-1")
			if !errors.Is(err, ErrInvalidCredential) {
				t.Errorf("after revoke: error = %v, want ErrInvalidCredential", err)
			}
		})
	}
}

func TestGenerateSession(t *testing.T) {
	token, tokenHash, err := generateSession()
	if err != nil {
		t.Fatalf("generateSession: %v", err)
	}

	if len(token) == 0 {
		t.Error("token is empty")
	}
	if len(tokenHash) != sha256.Size {
		t.Errorf("tokenHash length = %d, want %d", len(tokenHash), sha256.Size)
	}

	// Verify hash matches
	expected := sha256.Sum256([]byte(token))
	for i, b := range tokenHash {
		if b != expected[i] {
			t.Fatalf("tokenHash mismatch at byte %d", i)
		}
	}
}

func TestGenerateSessionUniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, _, err := generateSession()
		if err != nil {
			t.Fatalf("generateSession: %v", err)
		}
		if tokens[token] {
			t.Fatalf("duplicate token generated: %q", token)
		}
		tokens[token] = true
	}
}

func TestHashSessionToken(t *testing.T) {
	token := "test-token-123"
	hash1 := hashSessionToken(token)
	hash2 := hashSessionToken(token)

	if len(hash1) != sha256.Size {
		t.Errorf("hash length = %d, want %d", len(hash1), sha256.Size)
	}

	// Same input should produce same hash
	for i := range hash1 {
		if hash1[i] != hash2[i] {
			t.Fatalf("hash mismatch at byte %d", i)
		}
	}

	// Different input should produce different hash
	hash3 := hashSessionToken("different-token")
	same := true
	for i := range hash1 {
		if hash1[i] != hash3[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different tokens produced the same hash")
	}
}
