package strategy

import (
	"time"

	mon "gcli2api-go/internal/monitoring"
)

// OnResult 在请求完成后记录结果，用于冷却与恢复。
func (s *Strategy) OnResult(credID string, status int) {
	if credID == "" {
		return
	}
	if status > 0 && status < 400 {
		s.mu.Lock()
		if ce, ok := s.cooldown[credID]; ok {
			if ce.strikes <= 1 {
				delete(s.cooldown, credID)
			} else {
				ce.strikes--
				ce.until = time.Now()
				s.cooldown[credID] = ce
			}
		}
		s.mu.Unlock()
		return
	}
	shouldCooldown := status == 429 || status == 403 || (status >= 500 && status <= 599)
	if !shouldCooldown {
		return
	}
	base := time.Duration(s.cfg.RouterCooldownBaseMS) * time.Millisecond
	if base <= 0 {
		base = 2 * time.Second
	}
	max := time.Duration(s.cfg.RouterCooldownMaxMS) * time.Millisecond
	if max <= 0 {
		max = 60 * time.Second
	}
	s.mu.Lock()
	ce := s.cooldown[credID]
	ce.strikes++
	dur := base << (ce.strikes - 1)
	if dur > max {
		dur = max
	}
	ce.until = time.Now().Add(dur)
	s.cooldown[credID] = ce
	s.mu.Unlock()
	mon.RoutingCooldownEventsTotal.WithLabelValues(toStatusLabel(status)).Inc()
	mon.RoutingCooldownSize.Set(float64(len(s.cooldown)))
}

func (s *Strategy) isCooledDown(credID string) bool {
	s.mu.RLock()
	ce, ok := s.cooldown[credID]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	return time.Now().Before(ce.until)
}

// ClearCooldown removes cooldown entry for a credential id.
func (s *Strategy) ClearCooldown(credID string) bool {
	if credID == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cooldown[credID]; ok {
		delete(s.cooldown, credID)
		mon.RoutingCooldownSize.Set(float64(len(s.cooldown)))
		return true
	}
	return false
}

// SetCooldown sets cooldown state directly (for restore), with safety checks.
func (s *Strategy) SetCooldown(credID string, strikes int, until time.Time) {
	if credID == "" || strikes <= 0 {
		return
	}
	s.mu.Lock()
	s.cooldown[credID] = cooldownEntry{strikes: strikes, until: until}
	sz := len(s.cooldown)
	s.mu.Unlock()
	mon.RoutingCooldownSize.Set(float64(sz))
}

// Snapshot returns sticky count and a slice of cooldown infos.
func (s *Strategy) Snapshot() (int, []CooldownInfo) {
	s.mu.RLock()
	stickyCount := len(s.sticky)
	infos := make([]CooldownInfo, 0, len(s.cooldown))
	now := time.Now()
	for id, ce := range s.cooldown {
		rem := int64(0)
		if now.Before(ce.until) {
			rem = int64(ce.until.Sub(now).Seconds())
		}
		infos = append(infos, CooldownInfo{CredID: id, Strikes: ce.strikes, Until: ce.until, RemainingSec: rem})
		if rem > 0 {
			mon.RoutingCooldownRemainingSeconds.Observe(float64(rem))
		} else {
			mon.RoutingCooldownRemainingSeconds.Observe(0)
		}
	}
	s.mu.RUnlock()
	return stickyCount, infos
}
