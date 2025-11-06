package usage

import (
	"context"
)

// Storage defines the interface for persisting usage statistics
type Storage interface {
	// LoadStats loads usage statistics from storage
	LoadStats(ctx context.Context) (*Stats, error)

	// SaveStats saves usage statistics to storage
	SaveStats(ctx context.Context, stats *Stats) error

	// LoadCredentialUsage loads usage for a specific credential
	LoadCredentialUsage(ctx context.Context, credentialID string) (*CredentialUsage, error)

	// SaveCredentialUsage saves usage for a specific credential
	SaveCredentialUsage(ctx context.Context, usage *CredentialUsage) error
}

// NoOpStorage is a storage implementation that does nothing (for testing)
type NoOpStorage struct{}

// LoadStats implements Storage
func (n *NoOpStorage) LoadStats(ctx context.Context) (*Stats, error) {
	return NewStats(), nil
}

// SaveStats implements Storage
func (n *NoOpStorage) SaveStats(ctx context.Context, stats *Stats) error {
	return nil
}

// LoadCredentialUsage implements Storage
func (n *NoOpStorage) LoadCredentialUsage(ctx context.Context, credentialID string) (*CredentialUsage, error) {
	return nil, nil
}

// SaveCredentialUsage implements Storage
func (n *NoOpStorage) SaveCredentialUsage(ctx context.Context, usage *CredentialUsage) error {
	return nil
}

