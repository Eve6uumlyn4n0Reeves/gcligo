package strategy

import (
	"time"

	mon "gcli2api-go/internal/monitoring"
)

func (s *Strategy) setSticky(key, credID string, ttl time.Duration) {
	if key == "" || credID == "" {
		return
	}
	s.mu.Lock()
	s.sticky[key] = stickyEntry{credID: credID, expires: time.Now().Add(ttl)}
	s.mu.Unlock()
	mon.RoutingStickySize.Set(float64(len(s.sticky)))
}

func (s *Strategy) getSticky(key string) (string, bool) {
	if key == "" {
		return "", false
	}
	s.mu.RLock()
	se, ok := s.sticky[key]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Now().After(se.expires) {
		s.mu.Lock()
		delete(s.sticky, key)
		sz := len(s.sticky)
		s.mu.Unlock()
		mon.RoutingStickySize.Set(float64(sz))
		return "", false
	}
	return se.credID, true
}
