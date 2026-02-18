package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// KeyPackage represents an opaque MLS key package blob stored for a user.
type KeyPackage struct {
	ID             string
	UserID         string
	KeyPackageData []byte
	CreatedAt      int64
	ExpiresAt      int64
}

// StoreKeyPackage saves a key package for a user.
func (s *Store) StoreKeyPackage(ctx context.Context, userID string, data []byte, expiresAt int64) (string, error) {
	id := NewULID()
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO key_packages (id, user_id, key_package_data, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		id, userID, data, now, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("store key package: %w", err)
	}
	return id, nil
}

// ConsumeKeyPackage fetches one key package for a user and deletes it (single-use).
// Returns ErrNotFound if no key packages are available.
func (s *Store) ConsumeKeyPackage(ctx context.Context, userID string) (*KeyPackage, error) {
	var kp KeyPackage
	now := time.Now().Unix()

	err := s.InTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			`SELECT id, user_id, key_package_data, created_at, expires_at
			 FROM key_packages
			 WHERE user_id = ? AND expires_at > ?
			 ORDER BY created_at ASC LIMIT 1`,
			userID, now,
		).Scan(&kp.ID, &kp.UserID, &kp.KeyPackageData, &kp.CreatedAt, &kp.ExpiresAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("select key package: %w", err)
		}

		_, err = tx.ExecContext(ctx,
			`DELETE FROM key_packages WHERE id = ?`, kp.ID,
		)
		if err != nil {
			return fmt.Errorf("delete consumed key package: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &kp, nil
}

// CountKeyPackages returns the number of available (non-expired) key packages for a user.
func (s *Store) CountKeyPackages(ctx context.Context, userID string) (int, error) {
	var count int
	now := time.Now().Unix()
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM key_packages WHERE user_id = ? AND expires_at > ?`,
		userID, now,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count key packages: %w", err)
	}
	return count, nil
}

// DeleteExpiredKeyPackages removes key packages that have passed their expiry.
// Returns the number of deleted key packages.
func (s *Store) DeleteExpiredKeyPackages(ctx context.Context) (int64, error) {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM key_packages WHERE expires_at <= ?`, now,
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired key packages: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return n, nil
}
