package credential

// Acquire obtains a concurrency slot for the given credential ID.
// Returns a release function that must be called to free the slot.
// If no limit is configured or credID is empty, a no-op release is returned.
func (m *Manager) Acquire(credID string) func() {
	if m == nil || m.maxConcPerCred <= 0 || credID == "" {
		return func() {}
	}
	sem := m.getSemaphore(credID)
	sem <- struct{}{}
	return func() { <-sem }
}

func (m *Manager) getSemaphore(credID string) chan struct{} {
	m.semMu.Lock()
	defer m.semMu.Unlock()
	if ch, ok := m.sems[credID]; ok && ch != nil {
		return ch
	}
	size := m.maxConcPerCred
	if size <= 0 {
		size = 1
	}
	ch := make(chan struct{}, size)
	m.sems[credID] = ch
	return ch
}

// HasCapacity returns true if the credential has available concurrency slots
// or if no per-credential limit is configured.
func (m *Manager) HasCapacity(credID string) bool {
	if m == nil || m.maxConcPerCred <= 0 || credID == "" {
		return true
	}
	sem := m.getSemaphore(credID)
	return len(sem) < cap(sem)
}
