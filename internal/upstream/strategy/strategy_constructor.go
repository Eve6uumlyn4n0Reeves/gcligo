package strategy

import (
	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
)

// NewStrategy constructs a routing strategy with optional refresh callback.
func NewStrategy(cfg *config.Config, mgr *credential.Manager, onRefresh func(string)) *Strategy {
	if onRefresh == nil {
		onRefresh = func(string) {}
	}
	return &Strategy{
		cfg:        cfg,
		credMgr:    mgr,
		onRefresh:  onRefresh,
		sticky:     make(map[string]stickyEntry),
		cooldown:   make(map[string]cooldownEntry),
		pickLogs:   make([]PickLog, 0, 200),
		pickLogCap: 200,
	}
}

// SetOnRefresh allows late-binding the refresh callback for shared strategies.
func (s *Strategy) SetOnRefresh(cb func(string)) {
	if s == nil {
		return
	}
	if cb == nil {
		cb = func(string) {}
	}
	s.mu.Lock()
	s.onRefresh = cb
	s.mu.Unlock()
}
