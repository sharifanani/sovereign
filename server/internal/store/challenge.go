package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Challenge represents a WebAuthn challenge for registration or login.
type Challenge struct {
	ChallengeID   string
	ChallengeData []byte
	Username      string // may be empty for login challenges
	ChallengeType string // "registration" or "login"
	CreatedAt     int64
	ExpiresAt     int64
}

// CreateChallenge inserts a new challenge.
func (s *Store) CreateChallenge(ctx context.Context, c *Challenge) error {
	var username interface{}
	if c.Username != "" {
		username = c.Username
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO challenge (challenge_id, challenge_data, username, challenge_type, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.ChallengeID, c.ChallengeData, username, c.ChallengeType, c.CreatedAt, c.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert challenge: %w", err)
	}
	return nil
}

// GetChallenge returns a challenge by ID. Returns ErrNotFound if not found.
func (s *Store) GetChallenge(ctx context.Context, challengeID string) (*Challenge, error) {
	c := &Challenge{}
	var username sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT challenge_id, challenge_data, username, challenge_type, created_at, expires_at
		 FROM challenge WHERE challenge_id = ?`, challengeID,
	).Scan(&c.ChallengeID, &c.ChallengeData, &username, &c.ChallengeType, &c.CreatedAt, &c.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get challenge: %w", err)
	}
	if username.Valid {
		c.Username = username.String
	}
	return c, nil
}

// DeleteChallenge deletes a challenge by ID. Returns ErrNotFound if not found.
func (s *Store) DeleteChallenge(ctx context.Context, challengeID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM challenge WHERE challenge_id = ?`, challengeID)
	if err != nil {
		return fmt.Errorf("delete challenge: %w", err)
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

// DeleteExpiredChallenges removes all challenges that have expired.
// Returns the number of challenges deleted.
func (s *Store) DeleteExpiredChallenges(ctx context.Context) (int64, error) {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx, `DELETE FROM challenge WHERE expires_at <= ?`, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired challenges: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return n, nil
}
