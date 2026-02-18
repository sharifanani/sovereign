package store

import (
	"context"
	"database/sql"
	"fmt"
)

// User represents a registered user on this Sovereign server.
type User struct {
	ID          string
	Username    string
	DisplayName string
	Role        string
	Enabled     bool
	CreatedAt   int64
	UpdatedAt   int64
}

// CreateUser inserts a new user. Returns ErrConflict if the username is taken.
func (s *Store) CreateUser(ctx context.Context, u *User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user (id, username, display_name, role, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.DisplayName, u.Role, u.Enabled, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("username %q: %w", u.Username, ErrConflict)
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetUserByID returns a user by ID. Returns ErrNotFound if not found.
func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, display_name, role, enabled, created_at, updated_at
		 FROM user WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Enabled, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// GetUserByUsername returns a user by username. Returns ErrNotFound if not found.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, display_name, role, enabled, created_at, updated_at
		 FROM user WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Enabled, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return u, nil
}

// UpdateUser updates a user's display_name, role, enabled, and updated_at fields.
// Returns ErrNotFound if the user does not exist.
func (s *Store) UpdateUser(ctx context.Context, u *User) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE user SET display_name = ?, role = ?, enabled = ?, updated_at = ?
		 WHERE id = ?`,
		u.DisplayName, u.Role, u.Enabled, u.UpdatedAt, u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
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

// ListUsers returns all users ordered by username.
func (s *Store) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, display_name, role, enabled, created_at, updated_at
		 FROM user ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Enabled, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}
