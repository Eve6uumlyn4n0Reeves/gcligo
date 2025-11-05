// go:build !stats_isolation

package storage

import (
	"testing"

	"gcli2api-go/internal/config"
)

func TestDetectBackendLabel(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		backend  Backend
		expected string
	}{
		{
			name:     "PostgresBackend",
			cfg:      nil,
			backend:  &PostgresBackend{},
			expected: "postgres",
		},
		{
			name:     "MongoDBBackend",
			cfg:      nil,
			backend:  &MongoDBBackend{},
			expected: "mongodb",
		},
		{
			name:     "RedisBackend",
			cfg:      nil,
			backend:  &RedisBackend{},
			expected: "redis",
		},
		{
			name:     "FileBackend",
			cfg:      nil,
			backend:  &FileBackend{},
			expected: "file",
		},
		{
			name:     "GitBackend",
			cfg:      nil,
			backend:  &GitBackend{},
			expected: "git",
		},
		{
			name: "Config override postgres",
			cfg: &config.Config{
				StorageBackend: "postgres",
			},
			backend:  &FileBackend{},
			expected: "postgres",
		},
		{
			name: "Config override mongodb",
			cfg: &config.Config{
				StorageBackend: "mongodb",
			},
			backend:  &FileBackend{},
			expected: "mongodb",
		},
		{
			name: "Config with auto",
			cfg: &config.Config{
				StorageBackend: "auto",
			},
			backend:  &PostgresBackend{},
			expected: "postgres",
		},
		{
			name: "Config with empty string",
			cfg: &config.Config{
				StorageBackend: "",
			},
			backend:  &MongoDBBackend{},
			expected: "mongodb",
		},
		{
			name: "Config with whitespace",
			cfg: &config.Config{
				StorageBackend: "  ",
			},
			backend:  &RedisBackend{},
			expected: "redis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectBackendLabel(tt.cfg, tt.backend)
			if result != tt.expected {
				t.Errorf("DetectBackendLabel() = %q, want %q", result, tt.expected)
			}
		})
	}
}
