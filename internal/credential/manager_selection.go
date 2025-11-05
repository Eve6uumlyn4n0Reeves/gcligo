package credential

import (
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"
)

// GetCredential returns the next available credential using round-robin with health checks.
func (m *Manager) GetCredential() (*Credential, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.credentials) == 0 {
		return nil, fmt.Errorf("no credentials available")
	}

	// First pass: try to find a healthy credential starting from current index.
	startIndex := m.currentIndex
	attempts := 0

	for attempts < len(m.credentials) {
		cred := m.credentials[m.currentIndex]

		// Check if credential should rotate.
		if cred.ShouldRotate(m.rotationThreshold) {
			log.Infof("Rotating credential %s (reached %d calls)", cred.ID, cred.CallsSinceRotation)
			cred.ResetCallCount()
			m.currentIndex = (m.currentIndex + 1) % len(m.credentials)
			continue
		}

		// Check if credential is healthy.
		if cred.IsHealthy() {
			return cred.Clone(), nil
		}

		m.currentIndex = (m.currentIndex + 1) % len(m.credentials)
		attempts++
	}

	// Second pass: try to find the best credential by score (even if unhealthy).
	m.currentIndex = startIndex
	bestCred := m.findBestCredential()
	if bestCred != nil {
		log.Warnf("Using degraded credential %s (score: %.2f)", bestCred.ID, bestCred.GetScore())
		return bestCred.Clone(), nil
	}

	return nil, fmt.Errorf("all credentials are unavailable")
}

// GetAlternateCredential returns a healthy credential different from excludeID if possible.
// Falls back to any non-disabled credential when no healthy alternate is available.
func (m *Manager) GetAlternateCredential(excludeID string) (*Credential, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.credentials) == 0 {
		return nil, fmt.Errorf("no credentials available")
	}

	// First pass: healthy and not excluded.
	for i := 0; i < len(m.credentials); i++ {
		idx := (m.currentIndex + 1 + i) % len(m.credentials)
		cred := m.credentials[idx]
		if cred.ID == excludeID || cred.Disabled {
			continue
		}
		if cred.IsHealthy() {
			m.currentIndex = idx
			return cred.Clone(), nil
		}
	}

	// Second pass: any not excluded and not disabled.
	for i := 0; i < len(m.credentials); i++ {
		idx := (m.currentIndex + 1 + i) % len(m.credentials)
		cred := m.credentials[idx]
		if cred.ID == excludeID || cred.Disabled {
			continue
		}
		m.currentIndex = idx
		return cred.Clone(), nil
	}

	// No alternate available.
	return nil, fmt.Errorf("no alternate credential available")
}

// findBestCredential finds the credential with the highest score.
func (m *Manager) findBestCredential() *Credential {
	if len(m.credentials) == 0 {
		return nil
	}

	type scoredCred struct {
		cred  *Credential
		score float64
	}

	scored := make([]scoredCred, 0, len(m.credentials))
	for _, cred := range m.credentials {
		if !cred.Disabled {
			scored = append(scored, scoredCred{
				cred:  cred,
				score: cred.GetScore(),
			})
		}
	}

	if len(scored) == 0 {
		return nil
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	return scored[0].cred
}
