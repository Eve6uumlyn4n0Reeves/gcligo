package credential

import "fmt"

// mutateCredential safely locks a credential before applying the mutate function.
// It ensures all mutations acquire locks in the same order to avoid deadlocks.
func (m *Manager) mutateCredential(credID string, mutate func(*Credential) error) (*Credential, error) {
	if credID == "" {
		return nil, fmt.Errorf("credential id is required")
	}

	m.mu.RLock()
	var target *Credential
	for _, cred := range m.credentials {
		if cred != nil && cred.ID == credID {
			target = cred
			break
		}
	}
	m.mu.RUnlock()

	if target == nil {
		return nil, fmt.Errorf("credential %s not found", credID)
	}

	target.mu.Lock()
	defer target.mu.Unlock()

	if mutate != nil {
		if err := mutate(target); err != nil {
			return nil, err
		}
	}

	return target, nil
}

// removeCredential detaches a credential and its source from the manager slice/maps under a single lock.
func (m *Manager) removeCredential(credID string) (*Credential, CredentialSource, error) {
	if credID == "" {
		return nil, nil, fmt.Errorf("credential id is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, cred := range m.credentials {
		if cred != nil && cred.ID == credID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return nil, nil, fmt.Errorf("credential %s not found", credID)
	}

	target := m.credentials[idx]
	var src CredentialSource
	if existing, ok := m.credSource[credID]; ok {
		src = existing
	}

	m.credentials = append(m.credentials[:idx], m.credentials[idx+1:]...)
	delete(m.credSource, credID)
	return target, src, nil
}

// hasCredential checks if manager already tracks a credential id (used in tests).
func (m *Manager) hasCredential(credID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, cred := range m.credentials {
		if cred != nil && cred.ID == credID {
			return true
		}
	}
	return false
}

// cloneCredential safely clones a credential without leaking locks.
func cloneCredential(c *Credential) *Credential {
	if c == nil {
		return nil
	}
	return c.Clone()
}
