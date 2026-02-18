package mls

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sovereign-im/sovereign/server/internal/store"
)

// Default key package expiry (30 days).
const defaultKeyPackageExpiry = 30 * 24 * time.Hour

// Errors for MLS operations.
var (
	ErrNoKeyPackage    = errors.New("no key package available")
	ErrInvalidPayload  = errors.New("invalid key package payload")
	ErrNotMember       = errors.New("not a member of conversation")
	ErrRecipientNotFound = errors.New("recipient not found")
)

// Service manages MLS key packages and message routing.
// The server is a delivery service — it stores and forwards opaque blobs
// without performing MLS crypto.
type Service struct {
	store *store.Store
}

// NewService creates a new MLS service.
func NewService(s *store.Store) *Service {
	return &Service{store: s}
}

// UploadKeyPackage validates basic structure and stores a key package for the user.
func (s *Service) UploadKeyPackage(ctx context.Context, userID string, data []byte) error {
	if len(data) == 0 {
		return ErrInvalidPayload
	}
	// The server treats key packages as opaque blobs. We only validate
	// that the data is non-empty. Cryptographic validation happens on clients.
	expiresAt := time.Now().Add(defaultKeyPackageExpiry).Unix()
	_, err := s.store.StoreKeyPackage(ctx, userID, data, expiresAt)
	if err != nil {
		return fmt.Errorf("upload key package: %w", err)
	}
	return nil
}

// FetchKeyPackage consumes and returns one key package for the target user.
func (s *Service) FetchKeyPackage(ctx context.Context, targetUserID string) ([]byte, error) {
	kp, err := s.store.ConsumeKeyPackage(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrNoKeyPackage
		}
		return nil, fmt.Errorf("fetch key package: %w", err)
	}
	return kp.KeyPackageData, nil
}

// CountKeyPackages returns the number of available key packages for a user.
func (s *Service) CountKeyPackages(ctx context.Context, userID string) (int, error) {
	return s.store.CountKeyPackages(ctx, userID)
}

// CleanupExpiredKeyPackages removes expired key packages.
func (s *Service) CleanupExpiredKeyPackages(ctx context.Context) (int64, error) {
	return s.store.DeleteExpiredKeyPackages(ctx)
}
