package monitoring

import "sync"

var defaultMetrics struct {
	mu  sync.RWMutex
	ref *EnhancedMetrics
}

// SetDefaultMetrics registers the shared EnhancedMetrics instance for process-wide introspection.
func SetDefaultMetrics(m *EnhancedMetrics) {
	defaultMetrics.mu.Lock()
	defaultMetrics.ref = m
	defaultMetrics.mu.Unlock()
}

// DefaultMetrics returns the registered EnhancedMetrics instance, if any.
func DefaultMetrics() *EnhancedMetrics {
	defaultMetrics.mu.RLock()
	defer defaultMetrics.mu.RUnlock()
	return defaultMetrics.ref
}
