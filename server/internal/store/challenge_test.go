package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func makeChallenge(id, username, challengeType string, expiresAt int64) *Challenge {
	return &Challenge{
		ChallengeID:   id,
		ChallengeData: []byte(`{"test":"data"}`),
		Username:      username,
		ChallengeType: challengeType,
		CreatedAt:     time.Now().Unix(),
		ExpiresAt:     expiresAt,
	}
}

func TestCreateChallenge(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	c := makeChallenge("ch1", "alice", "registration", time.Now().Add(60*time.Second).Unix())
	if err := s.CreateChallenge(ctx, c); err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}
}

func TestGetChallenge(t *testing.T) {
	tests := []struct {
		name        string
		challengeID string
		setup       bool
		wantErr     error
	}{
		{
			name:        "found",
			challengeID: "ch1",
			setup:       true,
		},
		{
			name:        "not found",
			challengeID: "nonexistent",
			wantErr:     ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			ctx := context.Background()

			if tt.setup {
				c := makeChallenge("ch1", "alice", "registration", time.Now().Add(60*time.Second).Unix())
				if err := s.CreateChallenge(ctx, c); err != nil {
					t.Fatalf("CreateChallenge: %v", err)
				}
			}

			got, err := s.GetChallenge(ctx, tt.challengeID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ChallengeID != tt.challengeID {
				t.Errorf("ChallengeID = %q, want %q", got.ChallengeID, tt.challengeID)
			}
		})
	}
}

func TestGetChallengeWithEmptyUsername(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	c := makeChallenge("ch1", "", "login", time.Now().Add(30*time.Second).Unix())
	if err := s.CreateChallenge(ctx, c); err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}

	got, err := s.GetChallenge(ctx, "ch1")
	if err != nil {
		t.Fatalf("GetChallenge: %v", err)
	}
	if got.Username != "" {
		t.Errorf("Username = %q, want empty", got.Username)
	}
}

func TestDeleteChallenge(t *testing.T) {
	tests := []struct {
		name        string
		challengeID string
		setup       bool
		wantErr     error
	}{
		{
			name:        "success",
			challengeID: "ch1",
			setup:       true,
		},
		{
			name:        "not found",
			challengeID: "nonexistent",
			wantErr:     ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			ctx := context.Background()

			if tt.setup {
				c := makeChallenge("ch1", "alice", "registration", time.Now().Add(60*time.Second).Unix())
				if err := s.CreateChallenge(ctx, c); err != nil {
					t.Fatalf("CreateChallenge: %v", err)
				}
			}

			err := s.DeleteChallenge(ctx, tt.challengeID)
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
			_, err = s.GetChallenge(ctx, tt.challengeID)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("after delete: error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestDeleteExpiredChallenges(t *testing.T) {
	tests := []struct {
		name        string
		challenges  []struct{ id string; expired bool }
		wantDeleted int64
	}{
		{
			name:        "no challenges",
			wantDeleted: 0,
		},
		{
			name: "only expired challenges",
			challenges: []struct{ id string; expired bool }{
				{"ch1", true},
				{"ch2", true},
			},
			wantDeleted: 2,
		},
		{
			name: "only valid challenges",
			challenges: []struct{ id string; expired bool }{
				{"ch1", false},
				{"ch2", false},
			},
			wantDeleted: 0,
		},
		{
			name: "mix of expired and valid",
			challenges: []struct{ id string; expired bool }{
				{"ch1", true},
				{"ch2", false},
				{"ch3", true},
			},
			wantDeleted: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			ctx := context.Background()

			now := time.Now().Unix()
			for _, ch := range tt.challenges {
				expiresAt := now + 300 // 5 minutes from now
				if ch.expired {
					expiresAt = now - 1 // already expired
				}
				c := makeChallenge(ch.id, "alice", "registration", expiresAt)
				if err := s.CreateChallenge(ctx, c); err != nil {
					t.Fatalf("CreateChallenge(%q): %v", ch.id, err)
				}
			}

			deleted, err := s.DeleteExpiredChallenges(ctx)
			if err != nil {
				t.Fatalf("DeleteExpiredChallenges: %v", err)
			}
			if deleted != tt.wantDeleted {
				t.Errorf("deleted = %d, want %d", deleted, tt.wantDeleted)
			}
		})
	}
}

func TestChallengeRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	want := &Challenge{
		ChallengeID:   "ch1",
		ChallengeData: []byte(`{"session_data":{"challenge":"abc"}}`),
		Username:      "alice",
		ChallengeType: "registration",
		CreatedAt:     time.Now().Unix(),
		ExpiresAt:     time.Now().Add(60 * time.Second).Unix(),
	}

	if err := s.CreateChallenge(ctx, want); err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}

	got, err := s.GetChallenge(ctx, "ch1")
	if err != nil {
		t.Fatalf("GetChallenge: %v", err)
	}

	if got.ChallengeID != want.ChallengeID {
		t.Errorf("ChallengeID = %q, want %q", got.ChallengeID, want.ChallengeID)
	}
	if string(got.ChallengeData) != string(want.ChallengeData) {
		t.Errorf("ChallengeData = %q, want %q", got.ChallengeData, want.ChallengeData)
	}
	if got.Username != want.Username {
		t.Errorf("Username = %q, want %q", got.Username, want.Username)
	}
	if got.ChallengeType != want.ChallengeType {
		t.Errorf("ChallengeType = %q, want %q", got.ChallengeType, want.ChallengeType)
	}
	if got.CreatedAt != want.CreatedAt {
		t.Errorf("CreatedAt = %d, want %d", got.CreatedAt, want.CreatedAt)
	}
	if got.ExpiresAt != want.ExpiresAt {
		t.Errorf("ExpiresAt = %d, want %d", got.ExpiresAt, want.ExpiresAt)
	}
}
