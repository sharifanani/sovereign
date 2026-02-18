package store

import (
	"context"
	"sort"
	"testing"
	"time"
)

// seedConversationWithMembers creates a conversation with the given members for message tests.
func seedConversationWithMembers(t *testing.T, s *Store, convID string, creator string, members []string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().Unix()

	// Create users if they don't exist.
	for _, uid := range append([]string{creator}, members...) {
		_ = s.CreateUser(ctx, &User{
			ID:          uid,
			Username:    "user-" + uid,
			DisplayName: "User " + uid,
			Role:        "member",
			Enabled:     true,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	// Create conversation directly via SQL (we have a known ID).
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO conversations (id, title, created_by, created_at) VALUES (?, ?, ?, ?)`,
		convID, "Test Conv", creator, now,
	)
	if err != nil {
		t.Fatalf("insert conversation: %v", err)
	}

	// Add creator as admin.
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO group_members (group_id, user_id, role, joined_at) VALUES (?, ?, 'admin', ?)`,
		convID, creator, now,
	)
	if err != nil {
		t.Fatalf("add creator: %v", err)
	}

	// Add other members.
	for _, uid := range members {
		if uid == creator {
			continue
		}
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO group_members (group_id, user_id, role, joined_at) VALUES (?, ?, 'member', ?)`,
			convID, uid, now,
		)
		if err != nil {
			t.Fatalf("add member %s: %v", uid, err)
		}
	}
}

func TestInsertMessage(t *testing.T) {
	tests := []struct {
		name        string
		groupID     string
		senderID    string
		payload     []byte
		messageType int
		epoch       int
		wantErr     bool
	}{
		{
			name:        "valid message",
			groupID:     "group-1",
			senderID:    "alice",
			payload:     []byte("encrypted data"),
			messageType: MsgTypeApplication,
			epoch:       1,
		},
		{
			name:        "empty payload",
			groupID:     "group-1",
			senderID:    "alice",
			payload:     []byte{},
			messageType: MsgTypeApplication,
			epoch:       0,
		},
		{
			name:        "commit message type",
			groupID:     "group-1",
			senderID:    "alice",
			payload:     []byte("commit data"),
			messageType: MsgTypeCommit,
			epoch:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			ctx := context.Background()
			seedConversationWithMembers(t, s, tt.groupID, tt.senderID, []string{"bob"})

			msgID, serverTS, err := s.InsertMessage(ctx, tt.groupID, tt.senderID, tt.payload, tt.messageType, tt.epoch)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("InsertMessage: %v", err)
			}

			if msgID == "" {
				t.Error("msgID is empty")
			}
			if serverTS == 0 {
				t.Error("serverTS is zero")
			}

			// Verify message was stored.
			msgs, err := s.GetMessagesByGroup(ctx, tt.groupID, "", 10, false)
			if err != nil {
				t.Fatalf("GetMessagesByGroup: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("expected 1 message, got %d", len(msgs))
			}
			if msgs[0].ID != msgID {
				t.Errorf("ID = %s, want %s", msgs[0].ID, msgID)
			}
			if msgs[0].MessageType != tt.messageType {
				t.Errorf("MessageType = %d, want %d", msgs[0].MessageType, tt.messageType)
			}
		})
	}
}

func TestInsertMessageCreatesDeliveryStatus(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedConversationWithMembers(t, s, "group-1", "alice", []string{"bob", "charlie"})

	msgID, _, err := s.InsertMessage(ctx, "group-1", "alice", []byte("hello"), MsgTypeApplication, 0)
	if err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}

	// Delivery status should exist for bob and charlie (not alice).
	for _, recipientID := range []string{"bob", "charlie"} {
		dr, err := s.GetDeliveryStatus(ctx, msgID, recipientID)
		if err != nil {
			t.Fatalf("GetDeliveryStatus(%s): %v", recipientID, err)
		}
		if dr.Status != DeliveryPending {
			t.Errorf("Status for %s = %d, want %d (pending)", recipientID, dr.Status, DeliveryPending)
		}
	}

	// No delivery status for the sender.
	_, err = s.GetDeliveryStatus(ctx, msgID, "alice")
	if err == nil {
		t.Error("expected ErrNotFound for sender delivery status, got nil")
	}
}

func TestGetMessagesByGroup(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedConversationWithMembers(t, s, "group-1", "alice", []string{"bob"})

	// Insert 5 messages. ULIDs use rand.Reader so order within the same
	// millisecond is not guaranteed — sort after insertion to match DB ordering.
	var ids []string
	for i := 0; i < 5; i++ {
		id, _, err := s.InsertMessage(ctx, "group-1", "alice", []byte("msg"), MsgTypeApplication, 0)
		if err != nil {
			t.Fatalf("InsertMessage %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	sort.Strings(ids) // match DB ORDER BY id

	t.Run("no cursor returns most recent", func(t *testing.T) {
		msgs, err := s.GetMessagesByGroup(ctx, "group-1", "", 3, false)
		if err != nil {
			t.Fatalf("GetMessagesByGroup: %v", err)
		}
		if len(msgs) != 3 {
			t.Fatalf("expected 3, got %d", len(msgs))
		}
		// Most recent first (DESC by id).
		if msgs[0].ID != ids[4] {
			t.Errorf("first message = %s, want %s", msgs[0].ID, ids[4])
		}
	})

	t.Run("forward cursor", func(t *testing.T) {
		msgs, err := s.GetMessagesByGroup(ctx, "group-1", ids[1], 10, true)
		if err != nil {
			t.Fatalf("GetMessagesByGroup forward: %v", err)
		}
		// Should get messages after ids[1] (ids[2], ids[3], ids[4]).
		if len(msgs) != 3 {
			t.Fatalf("expected 3, got %d", len(msgs))
		}
		if msgs[0].ID != ids[2] {
			t.Errorf("first forward message = %s, want %s", msgs[0].ID, ids[2])
		}
	})

	t.Run("backward cursor", func(t *testing.T) {
		msgs, err := s.GetMessagesByGroup(ctx, "group-1", ids[3], 10, false)
		if err != nil {
			t.Fatalf("GetMessagesByGroup backward: %v", err)
		}
		// Should get messages before ids[3] (ids[0], ids[1], ids[2]).
		if len(msgs) != 3 {
			t.Fatalf("expected 3, got %d", len(msgs))
		}
	})

	t.Run("empty group returns nil", func(t *testing.T) {
		msgs, err := s.GetMessagesByGroup(ctx, "nonexistent-group", "", 10, false)
		if err != nil {
			t.Fatalf("GetMessagesByGroup: %v", err)
		}
		if len(msgs) != 0 {
			t.Errorf("expected 0 messages, got %d", len(msgs))
		}
	})

	t.Run("limit clamped to 50 when invalid", func(t *testing.T) {
		msgs, err := s.GetMessagesByGroup(ctx, "group-1", "", 0, false)
		if err != nil {
			t.Fatalf("GetMessagesByGroup: %v", err)
		}
		// Should return up to 50 (we have 5).
		if len(msgs) != 5 {
			t.Errorf("expected 5, got %d", len(msgs))
		}
	})
}

func TestGetPendingMessages(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedConversationWithMembers(t, s, "group-1", "alice", []string{"bob"})

	// Insert a message from alice (creates pending delivery for bob).
	msgID, _, err := s.InsertMessage(ctx, "group-1", "alice", []byte("hello"), MsgTypeApplication, 0)
	if err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}

	t.Run("returns pending messages for recipient", func(t *testing.T) {
		msgs, err := s.GetPendingMessages(ctx, "bob")
		if err != nil {
			t.Fatalf("GetPendingMessages: %v", err)
		}
		if len(msgs) != 1 {
			t.Fatalf("expected 1 pending, got %d", len(msgs))
		}
		if msgs[0].ID != msgID {
			t.Errorf("message ID = %s, want %s", msgs[0].ID, msgID)
		}
	})

	t.Run("no pending messages for sender", func(t *testing.T) {
		msgs, err := s.GetPendingMessages(ctx, "alice")
		if err != nil {
			t.Fatalf("GetPendingMessages: %v", err)
		}
		if len(msgs) != 0 {
			t.Errorf("expected 0 pending for sender, got %d", len(msgs))
		}
	})

	t.Run("after delivery update, no longer pending", func(t *testing.T) {
		if err := s.UpdateDeliveryStatus(ctx, msgID, "bob", DeliveryDelivered); err != nil {
			t.Fatalf("UpdateDeliveryStatus: %v", err)
		}
		msgs, err := s.GetPendingMessages(ctx, "bob")
		if err != nil {
			t.Fatalf("GetPendingMessages: %v", err)
		}
		if len(msgs) != 0 {
			t.Errorf("expected 0 pending after delivery, got %d", len(msgs))
		}
	})
}

func TestUpdateDeliveryStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		wantDelivery bool
		wantRead     bool
	}{
		{
			name:         "pending to delivered",
			status:       DeliveryDelivered,
			wantDelivery: true,
			wantRead:     false,
		},
		{
			name:         "pending to read",
			status:       DeliveryRead,
			wantDelivery: true,
			wantRead:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			ctx := context.Background()
			seedConversationWithMembers(t, s, "group-1", "alice", []string{"bob"})

			msgID, _, err := s.InsertMessage(ctx, "group-1", "alice", []byte("hello"), MsgTypeApplication, 0)
			if err != nil {
				t.Fatalf("InsertMessage: %v", err)
			}

			if err := s.UpdateDeliveryStatus(ctx, msgID, "bob", tt.status); err != nil {
				t.Fatalf("UpdateDeliveryStatus: %v", err)
			}

			dr, err := s.GetDeliveryStatus(ctx, msgID, "bob")
			if err != nil {
				t.Fatalf("GetDeliveryStatus: %v", err)
			}
			if dr.Status != tt.status {
				t.Errorf("Status = %d, want %d", dr.Status, tt.status)
			}
			if tt.wantDelivery && dr.DeliveredAt == nil {
				t.Error("DeliveredAt is nil, want non-nil")
			}
			if tt.wantRead && dr.ReadAt == nil {
				t.Error("ReadAt is nil, want non-nil")
			}
		})
	}

	t.Run("nonexistent message returns ErrNotFound", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()
		err := s.UpdateDeliveryStatus(ctx, "nonexistent", "bob", DeliveryDelivered)
		if err != ErrNotFound {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
}

func TestGetMessageSenderID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedConversationWithMembers(t, s, "group-1", "alice", []string{"bob"})

	msgID, _, err := s.InsertMessage(ctx, "group-1", "alice", []byte("hello"), MsgTypeApplication, 0)
	if err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}

	t.Run("returns sender for existing message", func(t *testing.T) {
		senderID, err := s.GetMessageSenderID(ctx, msgID)
		if err != nil {
			t.Fatalf("GetMessageSenderID: %v", err)
		}
		if senderID != "alice" {
			t.Errorf("senderID = %s, want alice", senderID)
		}
	})

	t.Run("returns ErrNotFound for nonexistent message", func(t *testing.T) {
		_, err := s.GetMessageSenderID(ctx, "nonexistent")
		if err != ErrNotFound {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
}

func TestDeleteExpiredMessages(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedConversationWithMembers(t, s, "group-1", "alice", []string{"bob"})

	// Insert a message with a very old created_at.
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (id, group_id, sender_id, server_timestamp, payload, payload_size, message_type, epoch, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"old-msg", "group-1", "alice", 100, []byte("old"), 3, 0, 0, 1000, // created_at = 1000 (very old)
	)
	if err != nil {
		t.Fatalf("insert old message: %v", err)
	}

	// Insert a recent message.
	_, _, err = s.InsertMessage(ctx, "group-1", "alice", []byte("new"), MsgTypeApplication, 0)
	if err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}

	// Delete messages older than now.
	cutoff := time.Now().Unix()
	deleted, err := s.DeleteExpiredMessages(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteExpiredMessages: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify only the new message remains.
	msgs, err := s.GetMessagesByGroup(ctx, "group-1", "", 10, false)
	if err != nil {
		t.Fatalf("GetMessagesByGroup: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 remaining message, got %d", len(msgs))
	}
}
