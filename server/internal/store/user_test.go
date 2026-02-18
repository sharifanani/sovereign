package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func makeUser(id, username string) *User {
	now := time.Now().Unix()
	return &User{
		ID:          id,
		Username:    username,
		DisplayName: "Display " + username,
		Role:        "member",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name    string
		users   []*User // users to create in order
		wantErr error   // expected error on the last user
	}{
		{
			name:  "success",
			users: []*User{makeUser("u1", "alice")},
		},
		{
			name: "duplicate username returns ErrConflict",
			users: []*User{
				makeUser("u1", "alice"),
				makeUser("u2", "alice"),
			},
			wantErr: ErrConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			ctx := context.Background()

			var err error
			for _, u := range tt.users {
				err = s.CreateUser(ctx, u)
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

func TestGetUserByID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   bool // whether to create the user first
		wantErr error
	}{
		{
			name:  "found",
			id:    "u1",
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
			ctx := context.Background()

			if tt.setup {
				if err := s.CreateUser(ctx, makeUser("u1", "alice")); err != nil {
					t.Fatalf("CreateUser: %v", err)
				}
			}

			got, err := s.GetUserByID(ctx, tt.id)
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

func TestGetUserByUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		setup    bool
		wantErr  error
	}{
		{
			name:     "found",
			username: "alice",
			setup:    true,
		},
		{
			name:     "not found",
			username: "nonexistent",
			wantErr:  ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			ctx := context.Background()

			if tt.setup {
				if err := s.CreateUser(ctx, makeUser("u1", "alice")); err != nil {
					t.Fatalf("CreateUser: %v", err)
				}
			}

			got, err := s.GetUserByUsername(ctx, tt.username)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Username != tt.username {
				t.Errorf("Username = %q, want %q", got.Username, tt.username)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		name    string
		setup   bool
		wantErr error
	}{
		{
			name:  "success",
			setup: true,
		},
		{
			name:    "not found",
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			ctx := context.Background()

			if tt.setup {
				if err := s.CreateUser(ctx, makeUser("u1", "alice")); err != nil {
					t.Fatalf("CreateUser: %v", err)
				}
			}

			updated := &User{
				ID:          "u1",
				DisplayName: "Alice Updated",
				Role:        "admin",
				Enabled:     false,
				UpdatedAt:   time.Now().Unix(),
			}

			err := s.UpdateUser(ctx, updated)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := s.GetUserByID(ctx, "u1")
			if err != nil {
				t.Fatalf("GetUserByID: %v", err)
			}
			if got.DisplayName != "Alice Updated" {
				t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Alice Updated")
			}
			if got.Role != "admin" {
				t.Errorf("Role = %q, want %q", got.Role, "admin")
			}
			if got.Enabled {
				t.Error("Enabled = true, want false")
			}
		})
	}
}

func TestListUsers(t *testing.T) {
	tests := []struct {
		name      string
		users     []*User
		wantCount int
	}{
		{
			name:      "empty",
			wantCount: 0,
		},
		{
			name:      "single user",
			users:     []*User{makeUser("u1", "alice")},
			wantCount: 1,
		},
		{
			name: "multiple users ordered by username",
			users: []*User{
				makeUser("u2", "charlie"),
				makeUser("u1", "alice"),
				makeUser("u3", "bob"),
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			ctx := context.Background()

			for _, u := range tt.users {
				if err := s.CreateUser(ctx, u); err != nil {
					t.Fatalf("CreateUser(%q): %v", u.Username, err)
				}
			}

			got, err := s.ListUsers(ctx)
			if err != nil {
				t.Fatalf("ListUsers: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("len = %d, want %d", len(got), tt.wantCount)
			}

			// Verify alphabetical ordering
			if len(got) >= 2 {
				for i := 1; i < len(got); i++ {
					if got[i-1].Username >= got[i].Username {
						t.Errorf("users not in order: %q >= %q", got[i-1].Username, got[i].Username)
					}
				}
			}
		})
	}
}

func TestUserRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	now := time.Now().Unix()
	want := &User{
		ID:          "u1",
		Username:    "alice",
		DisplayName: "Alice Wonderland",
		Role:        "admin",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.CreateUser(ctx, want); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	got, err := s.GetUserByID(ctx, "u1")
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}

	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.Username != want.Username {
		t.Errorf("Username = %q, want %q", got.Username, want.Username)
	}
	if got.DisplayName != want.DisplayName {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, want.DisplayName)
	}
	if got.Role != want.Role {
		t.Errorf("Role = %q, want %q", got.Role, want.Role)
	}
	if got.Enabled != want.Enabled {
		t.Errorf("Enabled = %v, want %v", got.Enabled, want.Enabled)
	}
	if got.CreatedAt != want.CreatedAt {
		t.Errorf("CreatedAt = %d, want %d", got.CreatedAt, want.CreatedAt)
	}
	if got.UpdatedAt != want.UpdatedAt {
		t.Errorf("UpdatedAt = %d, want %d", got.UpdatedAt, want.UpdatedAt)
	}
}
