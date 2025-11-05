package credential

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// ✅ StartAutoRecovery starts automatic recovery of banned credentials
func (m *Manager) StartAutoRecovery(ctx context.Context) {
	if !m.autoRecoveryEnabled {
		log.Info("Auto-recovery is disabled")
		return
	}

	interval := m.autoRecoveryInterval
	if interval <= 0 {
		interval = 10 * time.Minute
	}

	if m.stopRecovery == nil {
		m.stopRecovery = make(chan struct{})
	}

	m.recoveryTicker = time.NewTicker(interval)
	log.Infof("Starting auto-recovery service (interval: %v)", interval)

	go func() {
		for {
			select {
			case <-m.recoveryTicker.C:
				m.tryRecoverBannedCredentials(ctx)
			case <-m.stopRecovery:
				m.recoveryTicker.Stop()
				return
			case <-ctx.Done():
				m.recoveryTicker.Stop()
				return
			}
		}
	}()
}

// ✅ StopAutoRecovery stops the auto-recovery service
func (m *Manager) StopAutoRecovery() {
	if m.recoveryTicker != nil {
		close(m.stopRecovery)
		m.recoveryTicker.Stop()
		m.recoveryTicker = nil
		m.stopRecovery = make(chan struct{})
		log.Info("Auto-recovery service stopped")
	}
}

// ✅ tryRecoverBannedCredentials attempts to recover auto-banned credentials
func (m *Manager) tryRecoverBannedCredentials(ctx context.Context) {
	m.mu.RLock()
	creds := make([]*Credential, len(m.credentials))
	copy(creds, m.credentials)
	m.mu.RUnlock()

	recoveredCount := 0
	for _, cred := range creds {
		if cred.CanRecover() {
			// Try to recover the credential
			m.recoverCredential(ctx, cred.ID)
			recoveredCount++
		}
	}

	if recoveredCount > 0 {
		log.Infof("Auto-recovery: recovered %d banned credential(s)", recoveredCount)
	}
}

// ✅ recoverCredential recovers a specific credential
func (m *Manager) recoverCredential(ctx context.Context, credID string) error {
	m.mu.RLock()
	var target *Credential
	for _, cred := range m.credentials {
		if cred.ID == credID {
			target = cred
			break
		}
	}
	m.mu.RUnlock()

	if target == nil {
		return fmt.Errorf("credential %s not found", credID)
	}

	if target.Type == "oauth" && target.IsExpired() && target.RefreshToken != "" {
		if err := m.RefreshCredential(ctx, credID); err != nil {
			log.Errorf("Failed to refresh credential %s during recovery: %v", credID, err)
			return err
		}
	}

	target.Recover()
	log.Infof("Recovered credential %s (was banned for: %s)", credID, target.BannedReason)
	m.persistCredentialState(target, true)
	return nil
}

// ✅ ForceRecoverAll force recovers all banned credentials (admin function)
func (m *Manager) ForceRecoverAll(ctx context.Context) int {
	m.mu.RLock()
	creds := make([]*Credential, len(m.credentials))
	copy(creds, m.credentials)
	m.mu.RUnlock()

	recoveredCount := 0
	for _, cred := range creds {
		cred.mu.RLock()
		autoBanned := cred.AutoBanned
		cred.mu.RUnlock()

		if autoBanned {
			if err := m.recoverCredential(ctx, cred.ID); err == nil {
				recoveredCount++
			}
		}
	}

	log.Infof("Force recovery: recovered %d credential(s)", recoveredCount)
	return recoveredCount
}

// ✅ ForceRecoverOne force recovers a specific credential by id (admin function)
func (m *Manager) ForceRecoverOne(ctx context.Context, credID string) error {
	return m.recoverCredential(ctx, credID)
}

// ✅ GetCredentialStats returns detailed statistics for all credentials
func (m *Manager) GetCredentialStats() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make([]map[string]interface{}, 0, len(m.credentials))

	for _, cred := range m.credentials {
		cred.mu.RLock()
		stat := map[string]interface{}{
			"id":                cred.ID,
			"email":             cred.Email,
			"project_id":        cred.ProjectID,
			"type":              cred.Type,
			"disabled":          cred.Disabled,
			"auto_banned":       cred.AutoBanned,
			"banned_reason":     cred.BannedReason,
			"ban_until":         cred.BanUntil,
			"health_score":      cred.GetScore(),
			"total_requests":    cred.TotalRequests,
			"success_count":     cred.SuccessCount,
			"failure_count":     cred.FailureCount,
			"consecutive_fails": cred.ConsecutiveFails,
			"last_success":      cred.LastSuccess,
			"last_failure":      cred.LastFailure,
			"last_error_code":   cred.LastErrorCode,
			"error_code_counts": cred.ErrorCodeCounts,
			"daily_usage":       cred.DailyUsage,
			"daily_limit":       cred.DailyLimit,
			"quota_reset_time":  cred.QuotaResetTime,
			"success_rate":      float64(0),
			"failure_weight":    cred.FailureWeight,
		}
		if cred.TotalRequests > 0 {
			stat["success_rate"] = float64(cred.SuccessCount) / float64(cred.TotalRequests)
		}
		cred.mu.RUnlock()
		stats = append(stats, stat)
	}
	return stats
}
