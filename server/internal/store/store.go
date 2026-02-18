package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Sentinel errors for store operations.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

// Store provides the data access layer over SQLite.
type Store struct {
	db *sql.DB
}

// New opens a SQLite database at the given path and runs migrations.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Single connection ensures PRAGMAs persist and avoids
	// SQLite write contention issues.
	db.SetMaxOpenConns(1)

	if err := configurePragmas(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("configure database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB.
func (s *Store) DB() *sql.DB {
	return s.db
}

// InTx executes fn within a database transaction. If fn returns an error,
// the transaction is rolled back; otherwise it is committed.
func (s *Store) InTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}
	return tx.Commit()
}

func configurePragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA cache_size = -64000",
		"PRAGMA temp_store = MEMORY",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("exec %q: %w", p, err)
		}
	}
	return nil
}

// migrate runs all pending database migrations in order.
func (s *Store) migrate() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version    INTEGER PRIMARY KEY,
		applied_at INTEGER NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	var currentVersion int
	err = s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("get current version: %w", err)
	}

	for i, m := range migrations {
		version := i + 1
		if version <= currentVersion {
			continue
		}

		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", version, err)
		}

		if err := m(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", version, err)
		}

		_, err = tx.Exec("INSERT INTO schema_version (version, applied_at) VALUES (?, ?)",
			version, time.Now().Unix())
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}

	return nil
}

// migrations is an ordered list of migration functions.
var migrations = []func(*sql.Tx) error{
	migrateV1,
	migrateV2,
}

// migrateV1 creates the initial schema for auth (Phase B).
func migrateV1(tx *sql.Tx) error {
	stmts := []string{
		`CREATE TABLE user (
			id           TEXT PRIMARY KEY,
			username     TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			role         TEXT NOT NULL DEFAULT 'member',
			enabled      INTEGER NOT NULL DEFAULT 1,
			created_at   INTEGER NOT NULL,
			updated_at   INTEGER NOT NULL
		)`,
		`CREATE UNIQUE INDEX idx_user_username ON user (username)`,

		`CREATE TABLE credential (
			id            TEXT PRIMARY KEY,
			user_id       TEXT NOT NULL,
			credential_id BLOB NOT NULL,
			public_key    BLOB NOT NULL,
			sign_count    INTEGER NOT NULL DEFAULT 0,
			created_at    INTEGER NOT NULL,
			last_used_at  INTEGER,
			FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE
		)`,
		`CREATE INDEX idx_credential_user_id ON credential (user_id)`,
		`CREATE UNIQUE INDEX idx_credential_credential_id ON credential (credential_id)`,

		`CREATE TABLE session (
			id           TEXT PRIMARY KEY,
			user_id      TEXT NOT NULL,
			credential_id TEXT,
			token_hash   BLOB NOT NULL,
			created_at   INTEGER NOT NULL,
			expires_at   INTEGER NOT NULL,
			last_seen_at INTEGER NOT NULL,
			FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE,
			FOREIGN KEY (credential_id) REFERENCES credential (id) ON DELETE SET NULL
		)`,
		`CREATE INDEX idx_session_user_id ON session (user_id)`,
		`CREATE UNIQUE INDEX idx_session_token_hash ON session (token_hash)`,
		`CREATE INDEX idx_session_expires_at ON session (expires_at)`,

		`CREATE TABLE challenge (
			challenge_id   TEXT PRIMARY KEY,
			challenge_data BLOB NOT NULL,
			username       TEXT,
			challenge_type TEXT NOT NULL,
			created_at     INTEGER NOT NULL,
			expires_at     INTEGER NOT NULL
		)`,
		`CREATE INDEX idx_challenge_expires_at ON challenge (expires_at)`,
	}

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:min(len(stmt), 60)], err)
		}
	}
	return nil
}

// migrateV2 creates the schema for messaging (Phase C).
func migrateV2(tx *sql.Tx) error {
	stmts := []string{
		// Messages table: stores encrypted message blobs
		`CREATE TABLE messages (
			id               TEXT PRIMARY KEY,
			group_id         TEXT NOT NULL,
			sender_id        TEXT NOT NULL,
			server_timestamp INTEGER NOT NULL,
			payload          BLOB NOT NULL,
			payload_size     INTEGER NOT NULL,
			message_type     INTEGER NOT NULL DEFAULT 0,
			epoch            INTEGER NOT NULL DEFAULT 0,
			created_at       INTEGER NOT NULL
		)`,
		`CREATE INDEX idx_messages_group_timestamp ON messages(group_id, server_timestamp)`,
		`CREATE INDEX idx_messages_sender ON messages(sender_id, server_timestamp)`,
		`CREATE INDEX idx_messages_created_at ON messages(created_at)`,
		`CREATE INDEX idx_messages_group_size ON messages(group_id, payload_size)`,

		// Delivery tracking: per-recipient delivery state
		`CREATE TABLE delivery_status (
			message_id   TEXT NOT NULL,
			recipient_id TEXT NOT NULL,
			status       INTEGER NOT NULL DEFAULT 0,
			delivered_at INTEGER,
			read_at      INTEGER,
			PRIMARY KEY (message_id, recipient_id),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX idx_delivery_pending ON delivery_status(recipient_id, status)`,

		// Group membership: tracks which users are in which groups
		`CREATE TABLE group_members (
			group_id  TEXT NOT NULL,
			user_id   TEXT NOT NULL,
			role      TEXT NOT NULL DEFAULT 'member',
			joined_at INTEGER NOT NULL,
			PRIMARY KEY (group_id, user_id)
		)`,
		`CREATE INDEX idx_group_members_user ON group_members(user_id)`,

		// Conversations metadata
		`CREATE TABLE conversations (
			id         TEXT PRIMARY KEY,
			title      TEXT NOT NULL DEFAULT '',
			created_by TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)`,

		// Key packages: opaque MLS key package blobs
		`CREATE TABLE key_packages (
			id               TEXT PRIMARY KEY,
			user_id          TEXT NOT NULL,
			key_package_data BLOB NOT NULL,
			created_at       INTEGER NOT NULL,
			expires_at       INTEGER NOT NULL
		)`,
		`CREATE INDEX idx_key_packages_user ON key_packages(user_id)`,
		`CREATE INDEX idx_key_packages_expires ON key_packages(expires_at)`,
	}

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:min(len(stmt), 60)], err)
		}
	}
	return nil
}

// isUniqueConstraintError returns true if the error is a SQLite UNIQUE constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
