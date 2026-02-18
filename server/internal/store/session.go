package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Session represents an active user session.
// The raw session token is never stored; only its SHA-256 hash.
type Session struct {
	ID           string
	UserID       string
	CredentialID string // may be empty if not tracked
	TokenHash    []byte
	CreatedAt    int64
	ExpiresAt    int64
	LastSeenAt   int64
}

// CreateSession inserts a new session.
func (s *Store) CreateSession(ctx context.Context, sess *Session) error {
	var credID interface{}
	if sess.CredentialID != "" {
		credID = sess.CredentialID
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO session (id, user_id, credential_id, token_hash, created_at, expires_at, last_seen_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.UserID, credID, sess.TokenHash, sess.CreatedAt, sess.ExpiresAt, sess.LastSeenAt,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// GetSessionByTokenHash returns a session by its token hash. Returns ErrNotFound if not found.
func (s *Store) GetSessionByTokenHash(ctx context.Context, tokenHash []byte) (*Session, error) {
	sess := &Session{}
	var credID sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, credential_id, token_hash, created_at, expires_at, last_seen_at
		 FROM session WHERE token_hash = ?`, tokenHash,
	).Scan(&sess.ID, &sess.UserID, &credID, &sess.TokenHash, &sess.CreatedAt, &sess.ExpiresAt, &sess.LastSeenAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get session by token hash: %w", err)
	}
	if credID.Valid {
		sess.CredentialID = credID.String
	}
	return sess, nil
}

// UpdateSessionLastUsed updates the last_seen_at timestamp for a session.
// Returns ErrNotFound if the session does not exist.
func (s *Store) UpdateSessionLastUsed(ctx context.Context, id string) error {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx,
		`UPDATE session SET last_seen_at = ? WHERE id = ?`, now, id,
	)
	if err != nil {
		return fmt.Errorf("update session last used: %w", err)
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

// DeleteSession deletes a session by ID. Returns ErrNotFound if not found.
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM session WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
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

// DeleteExpiredSessions removes all sessions that have expired.
// Returns the number of sessions deleted.
func (s *Store) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx, `DELETE FROM session WHERE expires_at <= ?`, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return n, nil
}
