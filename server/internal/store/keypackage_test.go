package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStoreKeyPackage(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		userID    string
		data      []byte
		expiresAt int64
	}{
		{
			name:      "valid key package",
			userID:    "alice",
			data:      []byte("key-package-data-1"),
			expiresAt: time.Now().Add(24 * time.Hour).Unix(),
		},
		{
			name:      "empty data",
			userID:    "alice",
			data:      []byte{},
			expiresAt: time.Now().Add(24 * time.Hour).Unix(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := s.StoreKeyPackage(ctx, tt.userID, tt.data, tt.expiresAt)
			if err != nil {
				t.Fatalf("StoreKeyPackage: %v", err)
			}
			if id == "" {
				t.Error("returned ID is empty")
			}
		})
	}
}

func TestConsumeKeyPackage(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	t.Run("consumes and deletes a key package", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour).Unix()
		_, err := s.StoreKeyPackage(ctx, "alice", []byte("kp-data"), expiresAt)
		if err != nil {
			t.Fatalf("StoreKeyPackage: %v", err)
		}

		kp, err := s.ConsumeKeyPackage(ctx, "alice")
		if err != nil {
			t.Fatalf("ConsumeKeyPackage: %v", err)
		}
		if string(kp.KeyPackageData) != "kp-data" {
			t.Errorf("data = %q, want kp-data", kp.KeyPackageData)
		}
		if kp.UserID != "alice" {
			t.Errorf("userID = %s, want alice", kp.UserID)
		}

		// Second consume should fail — single-use.
		_, err = s.ConsumeKeyPackage(ctx, "alice")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("second consume: error = %v, want ErrNotFound", err)
		}
	})

	t.Run("no key package available returns ErrNotFound", func(t *testing.T) {
		_, err := s.ConsumeKeyPackage(ctx, "nonexistent")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})

	t.Run("expired key package is not consumed", func(t *testing.T) {
		// Store with past expiry.
		expiresAt := time.Now().Add(-1 * time.Hour).Unix()
		_, err := s.StoreKeyPackage(ctx, "bob", []byte("expired-kp"), expiresAt)
		if err != nil {
			t.Fatalf("StoreKeyPackage: %v", err)
		}

		_, err = s.ConsumeKeyPackage(ctx, "bob")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expired consume: error = %v, want ErrNotFound", err)
		}
	})

	t.Run("consumes oldest key package first", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour).Unix()
		_, err := s.StoreKeyPackage(ctx, "charlie", []byte("kp-first"), expiresAt)
		if err != nil {
			t.Fatalf("StoreKeyPackage 1: %v", err)
		}
		_, err = s.StoreKeyPackage(ctx, "charlie", []byte("kp-second"), expiresAt)
		if err != nil {
			t.Fatalf("StoreKeyPackage 2: %v", err)
		}

		kp, err := s.ConsumeKeyPackage(ctx, "charlie")
		if err != nil {
			t.Fatalf("ConsumeKeyPackage: %v", err)
		}
		if string(kp.KeyPackageData) != "kp-first" {
			t.Errorf("consumed = %q, want kp-first (oldest)", kp.KeyPackageData)
		}
	})
}

func TestCountKeyPackages(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	t.Run("zero for no key packages", func(t *testing.T) {
		count, err := s.CountKeyPackages(ctx, "alice")
		if err != nil {
			t.Fatalf("CountKeyPackages: %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})

	t.Run("counts non-expired key packages", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			if _, err := s.StoreKeyPackage(ctx, "alice", []byte("kp"), expiresAt); err != nil {
				t.Fatalf("StoreKeyPackage: %v", err)
			}
		}
		// Also store an expired one.
		if _, err := s.StoreKeyPackage(ctx, "alice", []byte("expired"), time.Now().Add(-1*time.Hour).Unix()); err != nil {
			t.Fatalf("StoreKeyPackage expired: %v", err)
		}

		count, err := s.CountKeyPackages(ctx, "alice")
		if err != nil {
			t.Fatalf("CountKeyPackages: %v", err)
		}
		if count != 3 {
			t.Errorf("count = %d, want 3 (excluding expired)", count)
		}
	})
}

func TestDeleteExpiredKeyPackages(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Store 2 expired and 1 valid key packages.
	expired := time.Now().Add(-1 * time.Hour).Unix()
	valid := time.Now().Add(24 * time.Hour).Unix()

	for i := 0; i < 2; i++ {
		if _, err := s.StoreKeyPackage(ctx, "alice", []byte("expired"), expired); err != nil {
			t.Fatalf("StoreKeyPackage expired: %v", err)
		}
	}
	if _, err := s.StoreKeyPackage(ctx, "alice", []byte("valid"), valid); err != nil {
		t.Fatalf("StoreKeyPackage valid: %v", err)
	}

	deleted, err := s.DeleteExpiredKeyPackages(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredKeyPackages: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2", deleted)
	}

	// Only the valid one should remain.
	count, err := s.CountKeyPackages(ctx, "alice")
	if err != nil {
		t.Fatalf("CountKeyPackages: %v", err)
	}
	if count != 1 {
		t.Errorf("remaining = %d, want 1", count)
	}
}
