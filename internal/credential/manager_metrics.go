package credential

import (
	log "github.com/sirupsen/logrus"
)

// MarkSuccess marks a credential as successful and persists its state.
func (m *Manager) MarkSuccess(credID string) {
	var target *Credential
	m.mu.RLock()
	for _, cred := range m.credentials {
		if cred.ID == credID {
			cred.MarkSuccess()
			target = cred
			break
		}
	}
	m.mu.RUnlock()

	if target != nil {
		m.persistCredentialState(target, false)
	}
}

// MarkFailure marks a credential as failed (enhanced with status code) and persists the outcome.
func (m *Manager) MarkFailure(credID string, reason string, statusCode int) {
	var target *Credential
	m.mu.RLock()
	for _, cred := range m.credentials {
		if cred.ID == credID {
			cred.MarkFailureWithConfig(reason, statusCode, m.autoBan)
			cred.mu.RLock()
			weight := cred.FailureWeight
			autoBanned := cred.AutoBanned
			bannedReason := cred.BannedReason
			consecutive := cred.ConsecutiveFails
			cred.mu.RUnlock()
			target = cred

			if autoBanned {
				log.Warnf("Credential %s auto-banned: %s (status: %d, weight: %.2f)", credID, bannedReason, statusCode, weight)
			} else {
				log.Warnf("Credential %s failed: %s (status: %d, consecutive fails: %d, weight: %.2f)", credID, reason, statusCode, consecutive, weight)
			}
			break
		}
	}
	m.mu.RUnlock()

	if target != nil {
		m.persistCredentialState(target, true)
	}
}
