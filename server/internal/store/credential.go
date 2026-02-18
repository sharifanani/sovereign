package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Credential represents a WebAuthn/Passkey credential.
type Credential struct {
	ID           string
	UserID       string
	CredentialID []byte // WebAuthn credential ID (external identifier)
	PublicKey    []byte
	SignCount    int64
	CreatedAt    int64
	LastUsedAt   *int64 // nil if never used after creation
}

// CreateCredential inserts a new credential.
func (s *Store) CreateCredential(ctx context.Context, c *Credential) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO credential (id, user_id, credential_id, public_key, sign_count, created_at, last_used_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.UserID, c.CredentialID, c.PublicKey, c.SignCount, c.CreatedAt, c.LastUsedAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("credential: %w", ErrConflict)
		}
		return fmt.Errorf("insert credential: %w", err)
	}
	return nil
}

// GetCredentialByID returns a credential by its internal ID. Returns ErrNotFound if not found.
func (s *Store) GetCredentialByID(ctx context.Context, id string) (*Credential, error) {
	c := &Credential{}
	var lastUsedAt sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, credential_id, public_key, sign_count, created_at, last_used_at
		 FROM credential WHERE id = ?`, id,
	).Scan(&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &c.SignCount, &c.CreatedAt, &lastUsedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get credential by id: %w", err)
	}
	if lastUsedAt.Valid {
		c.LastUsedAt = &lastUsedAt.Int64
	}
	return c, nil
}

// GetCredentialsByUserID returns all credentials for a user.
func (s *Store) GetCredentialsByUserID(ctx context.Context, userID string) ([]*Credential, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, credential_id, public_key, sign_count, created_at, last_used_at
		 FROM credential WHERE user_id = ? ORDER BY created_at`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get credentials by user id: %w", err)
	}
	defer rows.Close()

	var creds []*Credential
	for rows.Next() {
		c := &Credential{}
		var lastUsedAt sql.NullInt64
		if err := rows.Scan(&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &c.SignCount, &c.CreatedAt, &lastUsedAt); err != nil {
			return nil, fmt.Errorf("scan credential: %w", err)
		}
		if lastUsedAt.Valid {
			c.LastUsedAt = &lastUsedAt.Int64
		}
		creds = append(creds, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate credentials: %w", err)
	}
	return creds, nil
}

// UpdateSignCount updates the sign count and last_used_at for a credential.
// Returns ErrNotFound if the credential does not exist.
func (s *Store) UpdateSignCount(ctx context.Context, id string, signCount int64) error {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx,
		`UPDATE credential SET sign_count = ?, last_used_at = ? WHERE id = ?`,
		signCount, now, id,
	)
	if err != nil {
		return fmt.Errorf("update sign count: %w", err)
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

// DeleteCredential deletes a credential by ID. Returns ErrNotFound if not found.
func (s *Store) DeleteCredential(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM credential WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete credential: %w", err)
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
