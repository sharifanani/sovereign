package mls

import (
	"context"
	"errors"
	"testing"

	"github.com/sovereign-im/sovereign/server/internal/store"
)

func newTestService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return NewService(s), s
}

func TestUploadKeyPackage(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		data    []byte
		wantErr error
	}{
		{
			name:   "valid key package",
			userID: "alice",
			data:   []byte("key-package-blob"),
		},
		{
			name:    "empty data returns ErrInvalidPayload",
			userID:  "alice",
			data:    []byte{},
			wantErr: ErrInvalidPayload,
		},
		{
			name:    "nil data returns ErrInvalidPayload",
			userID:  "alice",
			data:    nil,
			wantErr: ErrInvalidPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t)
			ctx := context.Background()

			err := svc.UploadKeyPackage(ctx, tt.userID, tt.data)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UploadKeyPackage: %v", err)
			}
		})
	}
}

func TestFetchKeyPackage(t *testing.T) {
	t.Run("returns uploaded key package data", func(t *testing.T) {
		svc, _ := newTestService(t)
		ctx := context.Background()

		if err := svc.UploadKeyPackage(ctx, "alice", []byte("kp-data")); err != nil {
			t.Fatalf("UploadKeyPackage: %v", err)
		}

		data, err := svc.FetchKeyPackage(ctx, "alice")
		if err != nil {
			t.Fatalf("FetchKeyPackage: %v", err)
		}
		if string(data) != "kp-data" {
			t.Errorf("data = %q, want kp-data", data)
		}
	})

	t.Run("no key package returns ErrNoKeyPackage", func(t *testing.T) {
		svc, _ := newTestService(t)
		ctx := context.Background()

		_, err := svc.FetchKeyPackage(ctx, "nobody")
		if !errors.Is(err, ErrNoKeyPackage) {
			t.Errorf("error = %v, want ErrNoKeyPackage", err)
		}
	})

	t.Run("key package is single-use (consumed on fetch)", func(t *testing.T) {
		svc, _ := newTestService(t)
		ctx := context.Background()

		if err := svc.UploadKeyPackage(ctx, "alice", []byte("single-use")); err != nil {
			t.Fatalf("UploadKeyPackage: %v", err)
		}

		// First fetch succeeds.
		_, err := svc.FetchKeyPackage(ctx, "alice")
		if err != nil {
			t.Fatalf("first FetchKeyPackage: %v", err)
		}

		// Second fetch fails — consumed.
		_, err = svc.FetchKeyPackage(ctx, "alice")
		if !errors.Is(err, ErrNoKeyPackage) {
			t.Errorf("second fetch error = %v, want ErrNoKeyPackage", err)
		}
	})

	t.Run("fetches from correct user", func(t *testing.T) {
		svc, _ := newTestService(t)
		ctx := context.Background()

		if err := svc.UploadKeyPackage(ctx, "alice", []byte("alice-kp")); err != nil {
			t.Fatalf("UploadKeyPackage alice: %v", err)
		}
		if err := svc.UploadKeyPackage(ctx, "bob", []byte("bob-kp")); err != nil {
			t.Fatalf("UploadKeyPackage bob: %v", err)
		}

		data, err := svc.FetchKeyPackage(ctx, "bob")
		if err != nil {
			t.Fatalf("FetchKeyPackage bob: %v", err)
		}
		if string(data) != "bob-kp" {
			t.Errorf("data = %q, want bob-kp", data)
		}
	})
}

func TestCountKeyPackages(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	t.Run("zero initially", func(t *testing.T) {
		count, err := svc.CountKeyPackages(ctx, "alice")
		if err != nil {
			t.Fatalf("CountKeyPackages: %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})

	t.Run("counts after uploads", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			if err := svc.UploadKeyPackage(ctx, "alice", []byte("kp")); err != nil {
				t.Fatalf("UploadKeyPackage: %v", err)
			}
		}
		count, err := svc.CountKeyPackages(ctx, "alice")
		if err != nil {
			t.Fatalf("CountKeyPackages: %v", err)
		}
		if count != 5 {
			t.Errorf("count = %d, want 5", count)
		}
	})

	t.Run("decrements after fetch", func(t *testing.T) {
		_, err := svc.FetchKeyPackage(ctx, "alice")
		if err != nil {
			t.Fatalf("FetchKeyPackage: %v", err)
		}
		count, err := svc.CountKeyPackages(ctx, "alice")
		if err != nil {
			t.Fatalf("CountKeyPackages: %v", err)
		}
		if count != 4 {
			t.Errorf("count = %d, want 4", count)
		}
	})
}

func TestCleanupExpiredKeyPackages(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// Upload some valid key packages.
	for i := 0; i < 3; i++ {
		if err := svc.UploadKeyPackage(ctx, "alice", []byte("valid-kp")); err != nil {
			t.Fatalf("UploadKeyPackage: %v", err)
		}
	}

	// Cleanup should delete 0 (all are fresh with 30-day expiry).
	deleted, err := svc.CleanupExpiredKeyPackages(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredKeyPackages: %v", err)
	}
	if deleted != 0 {
		t.Errorf("deleted = %d, want 0 (all valid)", deleted)
	}
}
