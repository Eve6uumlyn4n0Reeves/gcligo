package storage

import (
	"testing"
)

func TestErrNotFound(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"Simple key", "test-key", "key not found: test-key"},
		{"Empty key", "", "key not found: "},
		{"Complex key", "user:123:profile", "key not found: user:123:profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ErrNotFound{Key: tt.key}
			if err.Error() != tt.expected {
				t.Errorf("ErrNotFound.Error() = %q, want %q", err.Error(), tt.expected)
			}
		})
	}
}

func TestErrNotSupported(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		expected  string
	}{
		{"Simple operation", "transaction", "operation not supported: transaction"},
		{"Empty operation", "", "operation not supported: "},
		{"Complex operation", "batch_update", "operation not supported: batch_update"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ErrNotSupported{Operation: tt.operation}
			if err.Error() != tt.expected {
				t.Errorf("ErrNotSupported.Error() = %q, want %q", err.Error(), tt.expected)
			}
		})
	}
}
