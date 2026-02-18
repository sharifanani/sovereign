package store

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func setupUserForCredentialTests(t *testing.T, s *Store) {
	t.Helper()
	ctx := context.Background()
	u := makeUser("u1", "alice")
	if err := s.CreateUser(ctx, u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
}

func makeCredential(id, userID string, credentialID []byte) *Credential {
	return &Credential{
		ID:           id,
		UserID:       userID,
		CredentialID: credentialID,
		PublicKey:    []byte("pk-" + id),
		SignCount:    0,
		CreatedAt:    time.Now().Unix(),
	}
}

func TestCreateCredential(t *testing.T) {
	tests := []struct {
		name    string
		creds   []*Credential
		wantErr error
	}{
		{
			name:  "success",
			creds: []*Credential{makeCredential("c1", "u1", []byte("cred-id-1"))},
		},
		{
			name: "duplicate credential_id returns ErrConflict",
			creds: []*Credential{
				makeCredential("c1", "u1", []byte("cred-id-1")),
				makeCredential("c2", "u1", []byte("cred-id-1")),
			},
			wantErr: ErrConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			setupUserForCredentialTests(t, s)
			ctx := context.Background()

			var err error
			for _, c := range tt.creds {
				err = s.CreateCredential(ctx, c)
			}

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetCredentialByID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   bool
		wantErr error
	}{
		{
			name:  "found",
			id:    "c1",
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
			setupUserForCredentialTests(t, s)
			ctx := context.Background()

			if tt.setup {
				c := makeCredential("c1", "u1", []byte("cred-id-1"))
				if err := s.CreateCredential(ctx, c); err != nil {
					t.Fatalf("CreateCredential: %v", err)
				}
			}

			got, err := s.GetCredentialByID(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.id {
				t.Errorf("ID = %q, want %q", got.ID, tt.id)
			}
		})
	}
}

func TestGetCredentialsByUserID(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		numCreds  int
		wantCount int
	}{
		{
			name:      "no credentials",
			userID:    "u1",
			numCreds:  0,
			wantCount: 0,
		},
		{
			name:      "single credential",
			userID:    "u1",
			numCreds:  1,
			wantCount: 1,
		},
		{
			name:      "multiple credentials",
			userID:    "u1",
			numCreds:  3,
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			setupUserForCredentialTests(t, s)
			ctx := context.Background()

			for i := 0; i < tt.numCreds; i++ {
				c := makeCredential(
					fmt.Sprintf("c%d", i),
					tt.userID,
					[]byte(fmt.Sprintf("cred-id-%d", i)),
				)
				if err := s.CreateCredential(ctx, c); err != nil {
					t.Fatalf("CreateCredential(%d): %v", i, err)
				}
			}

			got, err := s.GetCredentialsByUserID(ctx, tt.userID)
			if err != nil {
				t.Fatalf("GetCredentialsByUserID: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("len = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestUpdateSignCount(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		signCount int64
		setup     bool
		wantErr   error
	}{
		{
			name:      "success",
			id:        "c1",
			signCount: 5,
			setup:     true,
		},
		{
			name:      "not found",
			id:        "nonexistent",
			signCount: 1,
			wantErr:   ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			setupUserForCredentialTests(t, s)
			ctx := context.Background()

			if tt.setup {
				c := makeCredential("c1", "u1", []byte("cred-id-1"))
				if err := s.CreateCredential(ctx, c); err != nil {
					t.Fatalf("CreateCredential: %v", err)
				}
			}

			err := s.UpdateSignCount(ctx, tt.id, tt.signCount)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := s.GetCredentialByID(ctx, tt.id)
			if err != nil {
				t.Fatalf("GetCredentialByID: %v", err)
			}
			if got.SignCount != tt.signCount {
				t.Errorf("SignCount = %d, want %d", got.SignCount, tt.signCount)
			}
			if got.LastUsedAt == nil {
				t.Error("LastUsedAt is nil after UpdateSignCount, want non-nil")
			}
		})
	}
}

func TestDeleteCredential(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   bool
		wantErr error
	}{
		{
			name:  "success",
			id:    "c1",
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
			setupUserForCredentialTests(t, s)
			ctx := context.Background()

			if tt.setup {
				c := makeCredential("c1", "u1", []byte("cred-id-1"))
				if err := s.CreateCredential(ctx, c); err != nil {
					t.Fatalf("CreateCredential: %v", err)
				}
			}

			err := s.DeleteCredential(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify it was deleted
			_, err = s.GetCredentialByID(ctx, tt.id)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("after delete: error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestCredentialRoundTrip(t *testing.T) {
	s := newTestStore(t)
	setupUserForCredentialTests(t, s)
	ctx := context.Background()

	want := &Credential{
		ID:           "c1",
		UserID:       "u1",
		CredentialID: []byte("external-cred-id"),
		PublicKey:    []byte("public-key-bytes"),
		SignCount:    42,
		CreatedAt:    time.Now().Unix(),
	}

	if err := s.CreateCredential(ctx, want); err != nil {
		t.Fatalf("CreateCredential: %v", err)
	}

	got, err := s.GetCredentialByID(ctx, "c1")
	if err != nil {
		t.Fatalf("GetCredentialByID: %v", err)
	}

	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.UserID != want.UserID {
		t.Errorf("UserID = %q, want %q", got.UserID, want.UserID)
	}
	if string(got.CredentialID) != string(want.CredentialID) {
		t.Errorf("CredentialID = %q, want %q", got.CredentialID, want.CredentialID)
	}
	if string(got.PublicKey) != string(want.PublicKey) {
		t.Errorf("PublicKey = %q, want %q", got.PublicKey, want.PublicKey)
	}
	if got.SignCount != want.SignCount {
		t.Errorf("SignCount = %d, want %d", got.SignCount, want.SignCount)
	}
	if got.LastUsedAt != nil {
		t.Errorf("LastUsedAt = %v, want nil", *got.LastUsedAt)
	}
}
