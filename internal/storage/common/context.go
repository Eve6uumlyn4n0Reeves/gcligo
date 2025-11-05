package common

import (
	"context"
	"time"
)

// WithStorageTimeout adds a default timeout to a context if one doesn't already exist
// This ensures all storage operations have a reasonable timeout to prevent hanging
func WithStorageTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 10 * time.Second // Default 10 second timeout
	}

	// If context already has a deadline, use it
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}

	// Otherwise, add timeout
	return context.WithTimeout(ctx, timeout)
}

// WithStorageTimeoutDefault adds a 10 second timeout if no deadline exists
func WithStorageTimeoutDefault(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithStorageTimeout(ctx, 10*time.Second)
}

// WithStorageTimeoutShort adds a 5 second timeout for quick operations
func WithStorageTimeoutShort(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithStorageTimeout(ctx, 5*time.Second)
}

// WithStorageTimeoutLong adds a 30 second timeout for long operations
func WithStorageTimeoutLong(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithStorageTimeout(ctx, 30*time.Second)
}
