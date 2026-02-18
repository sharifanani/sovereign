package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

// newTestStore creates an in-memory Store for testing.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("New(:memory:) error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNewStore(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
	}{
		{
			name:   "in-memory database opens successfully",
			dbPath: ":memory:",
		},
		{
			name:    "invalid path returns error",
			dbPath:  "/nonexistent/dir/test.db",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(tt.dbPath)
			if tt.wantErr {
				if err == nil {
					s.Close()
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer s.Close()
		})
	}
}

func TestMigrationIdempotent(t *testing.T) {
	// Run New twice on the same file path — second call should not fail
	// because migrations check schema_version.
	s1, err := New("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("first New() error: %v", err)
	}
	defer s1.Close()

	// Re-running migrate on the same DB should be idempotent.
	if err := s1.migrate(); err != nil {
		t.Fatalf("second migrate() error: %v", err)
	}
}

func TestTablesExist(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	tables := []string{"user", "credential", "session", "challenge", "schema_version"}
	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			var name string
			err := s.db.QueryRowContext(ctx,
				"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
			).Scan(&name)
			if err != nil {
				t.Fatalf("table %q not found: %v", table, err)
			}
			if name != table {
				t.Errorf("expected table %q, got %q", table, name)
			}
		})
	}
}

func TestSchemaVersion(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	var version int
	err := s.db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	if version != len(migrations) {
		t.Errorf("schema version = %d, want %d", version, len(migrations))
	}
}

func TestInTx(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	t.Run("commit on success", func(t *testing.T) {
		err := s.InTx(ctx, func(tx *sql.Tx) error {
			_, err := tx.Exec(
				`INSERT INTO user (id, username, display_name, role, enabled, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				"tx-user", "tx-test", "TX Test", "member", 1, 1000, 1000,
			)
			return err
		})
		if err != nil {
			t.Fatalf("InTx: %v", err)
		}
		got, err := s.GetUserByID(ctx, "tx-user")
		if err != nil {
			t.Fatalf("GetUserByID: %v", err)
		}
		if got.Username != "tx-test" {
			t.Errorf("Username = %q, want %q", got.Username, "tx-test")
		}
	})

	t.Run("rollback on error", func(t *testing.T) {
		wantErr := errors.New("test error")
		err := s.InTx(ctx, func(tx *sql.Tx) error {
			_, _ = tx.Exec(
				`INSERT INTO user (id, username, display_name, role, enabled, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				"tx-user-rollback", "tx-rollback", "TX Rollback", "member", 1, 1000, 1000,
			)
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Errorf("error = %v, want %v", err, wantErr)
		}
		// User should not exist after rollback
		_, err = s.GetUserByID(ctx, "tx-user-rollback")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("after rollback: error = %v, want ErrNotFound", err)
		}
	})
}
