package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Conversation represents a conversation (1:1 or group).
type Conversation struct {
	ID        string
	Title     string
	CreatedBy string
	CreatedAt int64
}

// GroupMember represents a user's membership in a group.
type GroupMember struct {
	GroupID  string
	UserID   string
	Role     string
	JoinedAt int64
}

// CreateConversation creates a new conversation and adds the creator as an admin member.
// Additional member IDs are added with the "member" role.
func (s *Store) CreateConversation(ctx context.Context, title, createdBy string, memberIDs []string) (*Conversation, error) {
	conv := &Conversation{
		ID:        NewULID(),
		Title:     title,
		CreatedBy: createdBy,
		CreatedAt: time.Now().Unix(),
	}

	err := s.InTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO conversations (id, title, created_by, created_at) VALUES (?, ?, ?, ?)`,
			conv.ID, conv.Title, conv.CreatedBy, conv.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert conversation: %w", err)
		}

		now := time.Now().Unix()

		// Add creator as admin.
		_, err = tx.ExecContext(ctx,
			`INSERT INTO group_members (group_id, user_id, role, joined_at) VALUES (?, ?, 'admin', ?)`,
			conv.ID, createdBy, now,
		)
		if err != nil {
			return fmt.Errorf("add creator to group: %w", err)
		}

		// Add other members.
		for _, memberID := range memberIDs {
			if memberID == createdBy {
				continue
			}
			_, err = tx.ExecContext(ctx,
				`INSERT INTO group_members (group_id, user_id, role, joined_at) VALUES (?, ?, 'member', ?)`,
				conv.ID, memberID, now,
			)
			if err != nil {
				return fmt.Errorf("add member %s: %w", memberID, err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return conv, nil
}

// GetConversation returns a conversation by ID. Returns ErrNotFound if not found.
func (s *Store) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	conv := &Conversation{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, created_by, created_at FROM conversations WHERE id = ?`, id,
	).Scan(&conv.ID, &conv.Title, &conv.CreatedBy, &conv.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	return conv, nil
}

// AddMember adds a user to a conversation.
func (s *Store) AddMember(ctx context.Context, groupID, userID, role string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO group_members (group_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		groupID, userID, role, time.Now().Unix(),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("member %s in group %s: %w", userID, groupID, ErrConflict)
		}
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

// RemoveMember removes a user from a conversation.
func (s *Store) RemoveMember(ctx context.Context, groupID, userID string) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM group_members WHERE group_id = ? AND user_id = ?`,
		groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetMembers returns all members of a conversation.
func (s *Store) GetMembers(ctx context.Context, groupID string) ([]*GroupMember, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT group_id, user_id, role, joined_at FROM group_members WHERE group_id = ? ORDER BY joined_at`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	defer rows.Close()

	var members []*GroupMember
	for rows.Next() {
		m := &GroupMember{}
		if err := rows.Scan(&m.GroupID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return members, nil
}

// GetConversationsForUser returns all conversations a user is a member of.
func (s *Store) GetConversationsForUser(ctx context.Context, userID string) ([]*Conversation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.title, c.created_by, c.created_at
		 FROM conversations c
		 JOIN group_members gm ON gm.group_id = c.id
		 WHERE gm.user_id = ?
		 ORDER BY c.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get conversations for user: %w", err)
	}
	defer rows.Close()

	var convs []*Conversation
	for rows.Next() {
		c := &Conversation{}
		if err := rows.Scan(&c.ID, &c.Title, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		convs = append(convs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}
	return convs, nil
}

// IsUserMember checks if a user is a member of a conversation.
func (s *Store) IsUserMember(ctx context.Context, groupID, userID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?`,
		groupID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return count > 0, nil
}

// GetMemberRole returns the role of a user in a conversation. Returns ErrNotFound
// if the user is not a member.
func (s *Store) GetMemberRole(ctx context.Context, groupID, userID string) (string, error) {
	var role string
	err := s.db.QueryRowContext(ctx,
		`SELECT role FROM group_members WHERE group_id = ? AND user_id = ?`,
		groupID, userID,
	).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("get member role: %w", err)
	}
	return role, nil
}

// TransferAdmin assigns the admin role to the longest-standing member in the group.
// This is used when the current admin leaves.
func (s *Store) TransferAdmin(ctx context.Context, groupID, leavingUserID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE group_members SET role = 'admin'
		 WHERE group_id = ? AND user_id = (
			SELECT user_id FROM group_members
			WHERE group_id = ? AND user_id != ?
			ORDER BY joined_at ASC LIMIT 1
		 )`,
		groupID, groupID, leavingUserID,
	)
	if err != nil {
		return fmt.Errorf("transfer admin: %w", err)
	}
	return nil
}
