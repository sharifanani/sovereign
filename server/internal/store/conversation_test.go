package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateConversation(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		creator   string
		members   []string
		wantCount int // expected total members including creator
	}{
		{
			name:      "1:1 conversation",
			title:     "DM",
			creator:   "alice",
			members:   []string{"bob"},
			wantCount: 2,
		},
		{
			name:      "group conversation",
			title:     "Team Chat",
			creator:   "alice",
			members:   []string{"bob", "charlie"},
			wantCount: 3,
		},
		{
			name:      "creator in member list is deduplicated",
			title:     "Dedup",
			creator:   "alice",
			members:   []string{"alice", "bob"},
			wantCount: 2,
		},
		{
			name:      "solo conversation (no additional members)",
			title:     "Notes",
			creator:   "alice",
			members:   []string{},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			ctx := context.Background()
			now := time.Now().Unix()

			// Seed users.
			allUsers := append([]string{tt.creator}, tt.members...)
			for _, uid := range allUsers {
				_ = s.CreateUser(ctx, &User{
					ID: uid, Username: "u-" + uid, DisplayName: uid,
					Role: "member", Enabled: true, CreatedAt: now, UpdatedAt: now,
				})
			}

			conv, err := s.CreateConversation(ctx, tt.title, tt.creator, tt.members)
			if err != nil {
				t.Fatalf("CreateConversation: %v", err)
			}
			if conv.ID == "" {
				t.Error("conv.ID is empty")
			}
			if conv.Title != tt.title {
				t.Errorf("Title = %q, want %q", conv.Title, tt.title)
			}
			if conv.CreatedBy != tt.creator {
				t.Errorf("CreatedBy = %q, want %q", conv.CreatedBy, tt.creator)
			}

			members, err := s.GetMembers(ctx, conv.ID)
			if err != nil {
				t.Fatalf("GetMembers: %v", err)
			}
			if len(members) != tt.wantCount {
				t.Errorf("member count = %d, want %d", len(members), tt.wantCount)
			}

			// Creator should be admin.
			for _, m := range members {
				if m.UserID == tt.creator && m.Role != "admin" {
					t.Errorf("creator role = %q, want admin", m.Role)
				}
			}
		})
	}
}

func TestGetConversation(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Unix()

	_ = s.CreateUser(ctx, &User{
		ID: "alice", Username: "alice", DisplayName: "Alice",
		Role: "member", Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	conv, err := s.CreateConversation(ctx, "Test", "alice", nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := s.GetConversation(ctx, conv.ID)
		if err != nil {
			t.Fatalf("GetConversation: %v", err)
		}
		if got.Title != "Test" {
			t.Errorf("Title = %q, want Test", got.Title)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := s.GetConversation(ctx, "nonexistent")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
}

func TestAddRemoveMember(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Unix()

	for _, uid := range []string{"alice", "bob", "charlie"} {
		_ = s.CreateUser(ctx, &User{
			ID: uid, Username: uid, DisplayName: uid,
			Role: "member", Enabled: true, CreatedAt: now, UpdatedAt: now,
		})
	}

	conv, err := s.CreateConversation(ctx, "Group", "alice", []string{"bob"})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	t.Run("add member", func(t *testing.T) {
		if err := s.AddMember(ctx, conv.ID, "charlie", "member"); err != nil {
			t.Fatalf("AddMember: %v", err)
		}
		members, err := s.GetMembers(ctx, conv.ID)
		if err != nil {
			t.Fatalf("GetMembers: %v", err)
		}
		if len(members) != 3 {
			t.Errorf("member count = %d, want 3", len(members))
		}
	})

	t.Run("add duplicate member returns ErrConflict", func(t *testing.T) {
		err := s.AddMember(ctx, conv.ID, "charlie", "member")
		if !errors.Is(err, ErrConflict) {
			t.Errorf("error = %v, want ErrConflict", err)
		}
	})

	t.Run("remove member", func(t *testing.T) {
		if err := s.RemoveMember(ctx, conv.ID, "charlie"); err != nil {
			t.Fatalf("RemoveMember: %v", err)
		}
		members, err := s.GetMembers(ctx, conv.ID)
		if err != nil {
			t.Fatalf("GetMembers: %v", err)
		}
		if len(members) != 2 {
			t.Errorf("member count = %d, want 2", len(members))
		}
	})

	t.Run("remove nonexistent member returns ErrNotFound", func(t *testing.T) {
		err := s.RemoveMember(ctx, conv.ID, "nonexistent")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
}

func TestGetMembers(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Unix()

	for _, uid := range []string{"alice", "bob"} {
		_ = s.CreateUser(ctx, &User{
			ID: uid, Username: uid, DisplayName: uid,
			Role: "member", Enabled: true, CreatedAt: now, UpdatedAt: now,
		})
	}

	conv, err := s.CreateConversation(ctx, "Test", "alice", []string{"bob"})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	members, err := s.GetMembers(ctx, conv.ID)
	if err != nil {
		t.Fatalf("GetMembers: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("member count = %d, want 2", len(members))
	}

	// Members should be ordered by joined_at.
	if members[0].UserID != "alice" {
		t.Errorf("first member = %s, want alice (creator)", members[0].UserID)
	}
	if members[0].Role != "admin" {
		t.Errorf("creator role = %s, want admin", members[0].Role)
	}
	if members[1].Role != "member" {
		t.Errorf("non-creator role = %s, want member", members[1].Role)
	}
}

func TestGetConversationsForUser(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Unix()

	for _, uid := range []string{"alice", "bob", "charlie"} {
		_ = s.CreateUser(ctx, &User{
			ID: uid, Username: uid, DisplayName: uid,
			Role: "member", Enabled: true, CreatedAt: now, UpdatedAt: now,
		})
	}

	_, err := s.CreateConversation(ctx, "Conv 1", "alice", []string{"bob"})
	if err != nil {
		t.Fatalf("CreateConversation 1: %v", err)
	}
	_, err = s.CreateConversation(ctx, "Conv 2", "alice", []string{"charlie"})
	if err != nil {
		t.Fatalf("CreateConversation 2: %v", err)
	}

	t.Run("alice sees both conversations", func(t *testing.T) {
		convs, err := s.GetConversationsForUser(ctx, "alice")
		if err != nil {
			t.Fatalf("GetConversationsForUser: %v", err)
		}
		if len(convs) != 2 {
			t.Errorf("count = %d, want 2", len(convs))
		}
	})

	t.Run("bob sees only one conversation", func(t *testing.T) {
		convs, err := s.GetConversationsForUser(ctx, "bob")
		if err != nil {
			t.Fatalf("GetConversationsForUser: %v", err)
		}
		if len(convs) != 1 {
			t.Errorf("count = %d, want 1", len(convs))
		}
	})

	t.Run("unknown user sees no conversations", func(t *testing.T) {
		convs, err := s.GetConversationsForUser(ctx, "unknown")
		if err != nil {
			t.Fatalf("GetConversationsForUser: %v", err)
		}
		if len(convs) != 0 {
			t.Errorf("count = %d, want 0", len(convs))
		}
	})
}

func TestIsUserMember(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Unix()

	for _, uid := range []string{"alice", "bob"} {
		_ = s.CreateUser(ctx, &User{
			ID: uid, Username: uid, DisplayName: uid,
			Role: "member", Enabled: true, CreatedAt: now, UpdatedAt: now,
		})
	}

	conv, err := s.CreateConversation(ctx, "Test", "alice", []string{"bob"})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	tests := []struct {
		name   string
		userID string
		want   bool
	}{
		{"member is true", "alice", true},
		{"other member is true", "bob", true},
		{"non-member is false", "charlie", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.IsUserMember(ctx, conv.ID, tt.userID)
			if err != nil {
				t.Fatalf("IsUserMember: %v", err)
			}
			if got != tt.want {
				t.Errorf("IsUserMember = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMemberRole(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Unix()

	for _, uid := range []string{"alice", "bob"} {
		_ = s.CreateUser(ctx, &User{
			ID: uid, Username: uid, DisplayName: uid,
			Role: "member", Enabled: true, CreatedAt: now, UpdatedAt: now,
		})
	}

	conv, err := s.CreateConversation(ctx, "Test", "alice", []string{"bob"})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	t.Run("creator is admin", func(t *testing.T) {
		role, err := s.GetMemberRole(ctx, conv.ID, "alice")
		if err != nil {
			t.Fatalf("GetMemberRole: %v", err)
		}
		if role != "admin" {
			t.Errorf("role = %s, want admin", role)
		}
	})

	t.Run("other member is member", func(t *testing.T) {
		role, err := s.GetMemberRole(ctx, conv.ID, "bob")
		if err != nil {
			t.Fatalf("GetMemberRole: %v", err)
		}
		if role != "member" {
			t.Errorf("role = %s, want member", role)
		}
	})

	t.Run("nonexistent returns ErrNotFound", func(t *testing.T) {
		_, err := s.GetMemberRole(ctx, conv.ID, "nonexistent")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
}

func TestTransferAdmin(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Unix()

	for _, uid := range []string{"alice", "bob", "charlie"} {
		_ = s.CreateUser(ctx, &User{
			ID: uid, Username: uid, DisplayName: uid,
			Role: "member", Enabled: true, CreatedAt: now, UpdatedAt: now,
		})
	}

	conv, err := s.CreateConversation(ctx, "Test", "alice", []string{"bob", "charlie"})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Transfer admin from alice.
	if err := s.TransferAdmin(ctx, conv.ID, "alice"); err != nil {
		t.Fatalf("TransferAdmin: %v", err)
	}

	// The next oldest member (bob) should now be admin.
	role, err := s.GetMemberRole(ctx, conv.ID, "bob")
	if err != nil {
		t.Fatalf("GetMemberRole: %v", err)
	}
	if role != "admin" {
		t.Errorf("bob's role = %s, want admin", role)
	}
}
