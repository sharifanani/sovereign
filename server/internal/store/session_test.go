package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"
)

func setupUserForSessionTests(t *testing.T, s *Store) {
	t.Helper()
	ctx := context.Background()
	u := makeUser("u1", "alice")
	if err := s.CreateUser(ctx, u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
}

func makeSession(id, userID string, tokenHash []byte, expiresAt int64) *Session {
	now := time.Now().Unix()
	return &Session{
		ID:         id,
		UserID:     userID,
		TokenHash:  tokenHash,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		LastSeenAt: now,
	}
}

func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

func TestCreateSession(t *testing.T) {
	s := newTestStore(t)
	setupUserForSessionTests(t, s)
	ctx := context.Background()

	sess := makeSession("s1", "u1", hashToken("token-1"), time.Now().Add(24*time.Hour).Unix())
	err := s.CreateSession(ctx, sess)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
}

func TestGetSessionByTokenHash(t *testing.T) {
	tests := []struct {
		name      string
		tokenHash []byte
		setup     bool
		wantErr   error
	}{
		{
			name:      "found",
			tokenHash: hashToken("token-1"),
			setup:     true,
		},
		{
			name:      "not found",
			tokenHash: hashToken("nonexistent"),
			wantErr:   ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			setupUserForSessionTests(t, s)
			ctx := context.Background()

			if tt.setup {
				sess := makeSession("s1", "u1", hashToken("token-1"), time.Now().Add(24*time.Hour).Unix())
				if err := s.CreateSession(ctx, sess); err != nil {
					t.Fatalf("CreateSession: %v", err)
				}
			}

			got, err := s.GetSessionByTokenHash(ctx, tt.tokenHash)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != "s1" {
				t.Errorf("ID = %q, want %q", got.ID, "s1")
			}
		})
	}
}

func TestUpdateSessionLastUsed(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   bool
		wantErr error
	}{
		{
			name:  "success",
			id:    "s1",
			setup: true,
		},
		{
			name:    "not found",
			id:      "nonexistent",
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			setupUserForSessionTests(t, s)
			ctx := context.Background()

			if tt.setup {
				sess := makeSession("s1", "u1", hashToken("token-1"), time.Now().Add(24*time.Hour).Unix())
				sess.LastSeenAt = time.Now().Add(-1 * time.Hour).Unix()
				if err := s.CreateSession(ctx, sess); err != nil {
					t.Fatalf("CreateSession: %v", err)
				}
			}

			err := s.UpdateSessionLastUsed(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify last_seen_at was updated to approximately now
			got, err := s.GetSessionByTokenHash(ctx, hashToken("token-1"))
			if err != nil {
				t.Fatalf("GetSessionByTokenHash: %v", err)
			}
			now := time.Now().Unix()
			if got.LastSeenAt < now-2 || got.LastSeenAt > now+2 {
				t.Errorf("LastSeenAt = %d, want approximately %d", got.LastSeenAt, now)
			}
		})
	}
}

func TestDeleteSession(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   bool
		wantErr error
	}{
		{
			name:  "success",
			id:    "s1",
			setup: true,
		},
		{
			name:    "not found",
			id:      "nonexistent",
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			setupUserForSessionTests(t, s)
			ctx := context.Background()

			if tt.setup {
				sess := makeSession("s1", "u1", hashToken("token-1"), time.Now().Add(24*time.Hour).Unix())
				if err := s.CreateSession(ctx, sess); err != nil {
					t.Fatalf("CreateSession: %v", err)
				}
			}

			err := s.DeleteSession(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			_, err = s.GetSessionByTokenHash(ctx, hashToken("token-1"))
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("after delete: error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	tests := []struct {
		name        string
		sessions    []struct{ id, token string; expired bool }
		wantDeleted int64
		wantRemain  int
	}{
		{
			name:        "no sessions",
			wantDeleted: 0,
			wantRemain:  0,
		},
		{
			name: "only expired sessions",
			sessions: []struct{ id, token string; expired bool }{
				{"s1", "t1", true},
				{"s2", "t2", true},
			},
			wantDeleted: 2,
			wantRemain:  0,
		},
		{
			name: "only valid sessions",
			sessions: []struct{ id, token string; expired bool }{
				{"s1", "t1", false},
				{"s2", "t2", false},
			},
			wantDeleted: 0,
			wantRemain:  2,
		},
		{
			name: "mix of expired and valid",
			sessions: []struct{ id, token string; expired bool }{
				{"s1", "t1", true},
				{"s2", "t2", false},
				{"s3", "t3", true},
				{"s4", "t4", false},
			},
			wantDeleted: 2,
			wantRemain:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			setupUserForSessionTests(t, s)
			ctx := context.Background()

			now := time.Now().Unix()
			for _, sess := range tt.sessions {
				expiresAt := now + 3600 // 1 hour from now
				if sess.expired {
					expiresAt = now - 1 // already expired
				}
				session := makeSession(sess.id, "u1", hashToken(sess.token), expiresAt)
				if err := s.CreateSession(ctx, session); err != nil {
					t.Fatalf("CreateSession(%q): %v", sess.id, err)
				}
			}

			deleted, err := s.DeleteExpiredSessions(ctx)
			if err != nil {
				t.Fatalf("DeleteExpiredSessions: %v", err)
			}
			if deleted != tt.wantDeleted {
				t.Errorf("deleted = %d, want %d", deleted, tt.wantDeleted)
			}
		})
	}
}

func TestSessionWithCredentialID(t *testing.T) {
	s := newTestStore(t)
	setupUserForSessionTests(t, s)
	ctx := context.Background()

	// Create credential first
	cred := makeCredential("c1", "u1", []byte("cred-id-1"))
	if err := s.CreateCredential(ctx, cred); err != nil {
		t.Fatalf("CreateCredential: %v", err)
	}

	// Create session with credential ID
	sess := makeSession("s1", "u1", hashToken("token-1"), time.Now().Add(24*time.Hour).Unix())
	sess.CredentialID = "c1"
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := s.GetSessionByTokenHash(ctx, hashToken("token-1"))
	if err != nil {
		t.Fatalf("GetSessionByTokenHash: %v", err)
	}
	if got.CredentialID != "c1" {
		t.Errorf("CredentialID = %q, want %q", got.CredentialID, "c1")
	}
}
