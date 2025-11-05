package credential

import (
	"context"
	"sync"
)

// RefreshCoordinator coalesces concurrent refresh operations per credential.
type RefreshCoordinator interface {
	Do(ctx context.Context, credID string, fn func(ctx context.Context) error) error
}

// InflightCoordinator is a simple singleflight-like coordinator without external deps.
type InflightCoordinator struct {
	mu       sync.Mutex
	inflight map[string]*flight
}

type flight struct {
	wg  sync.WaitGroup
	err error
}

func NewInflightCoordinator() *InflightCoordinator {
	return &InflightCoordinator{inflight: make(map[string]*flight)}
}

func (c *InflightCoordinator) Do(ctx context.Context, credID string, fn func(ctx context.Context) error) error {
	if credID == "" {
		return fn(ctx)
	}
	c.mu.Lock()
	if f := c.inflight[credID]; f != nil {
		// another goroutine is refreshing; wait for it
		c.mu.Unlock()
		done := make(chan struct{})
		go func() { f.wg.Wait(); close(done) }()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return f.err
		}
	}
	f := &flight{}
	f.wg.Add(1)
	c.inflight[credID] = f
	c.mu.Unlock()

	// execute
	err := fn(ctx)
	f.err = err
	f.wg.Done()

	c.mu.Lock()
	delete(c.inflight, credID)
	c.mu.Unlock()
	return err
}
