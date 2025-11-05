package strategy

// recordPick stores a routing pick log with capacity trimming.
func (s *Strategy) recordPick(pl PickLog) {
	s.mu.Lock()
	if len(s.pickLogs) >= s.pickLogCap {
		copy(s.pickLogs, s.pickLogs[1:])
		s.pickLogs[len(s.pickLogs)-1] = pl
	} else {
		s.pickLogs = append(s.pickLogs, pl)
	}
	s.mu.Unlock()
}

// Picks returns recent pick logs up to limit.
func (s *Strategy) Picks(limit int) []PickLog {
	if limit <= 0 {
		limit = 50
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.pickLogs)
	if n == 0 {
		return nil
	}
	if limit > n {
		limit = n
	}
	out := make([]PickLog, limit)
	copy(out, s.pickLogs[n-limit:])
	return out
}
