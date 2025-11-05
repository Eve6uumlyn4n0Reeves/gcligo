package adapter

import (
	"context"
	"fmt"
	"time"
)

// GetAllCredentialStates 获取所有凭证状态
func (f *FileStorageAdapter) GetAllCredentialStates(ctx context.Context) (map[string]*CredentialState, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	states := make(map[string]*CredentialState, len(f.states))
	for id, state := range f.states {
		copy := *state
		states[id] = &copy
	}

	return states, nil
}

// UpdateCredentialStates 批量更新凭证状态
func (f *FileStorageAdapter) UpdateCredentialStates(ctx context.Context, states map[string]*CredentialState) error {
	f.mu.Lock()

	for id, state := range states {
		if state == nil {
			continue
		}
		if state.CreatedAt.IsZero() {
			state.CreatedAt = time.Now()
		}
		state.UpdatedAt = time.Now()

		if err := f.saveStateFile(state); err != nil {
			f.mu.Unlock()
			return fmt.Errorf("failed to save state file for %s: %w", id, err)
		}

		f.states[id] = state
	}

	f.mu.Unlock()
	f.notifyWatchers()

	return nil
}

// DiscoverCredentials 发现凭证
func (f *FileStorageAdapter) DiscoverCredentials(ctx context.Context) ([]*Credential, error) {
	return f.GetAllCredentials(ctx)
}

// RefreshCredential 刷新凭证
func (f *FileStorageAdapter) RefreshCredential(ctx context.Context, credID string) (*Credential, error) {
	return f.LoadCredential(ctx, credID)
}

// UpdateUsageStats 更新使用统计
func (f *FileStorageAdapter) UpdateUsageStats(ctx context.Context, credID string, stats map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	state, ok := f.states[credID]
	if !ok {
		state = &CredentialState{
			ID:          credID,
			HealthScore: 1.0,
			CreatedAt:   time.Now(),
		}
	}

	if state.UsageStats == nil {
		state.UsageStats = make(map[string]interface{})
	}
	for k, v := range stats {
		state.UsageStats[k] = v
	}
	state.UpdatedAt = time.Now()

	if err := f.saveStateFile(state); err != nil {
		return fmt.Errorf("failed to persist usage stats: %w", err)
	}

	f.states[credID] = state

	return nil
}

// GetUsageStats 获取使用统计
func (f *FileStorageAdapter) GetUsageStats(ctx context.Context, credID string) (map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	state, ok := f.states[credID]
	if !ok || state.UsageStats == nil {
		return make(map[string]interface{}), nil
	}

	stats := make(map[string]interface{}, len(state.UsageStats))
	for k, v := range state.UsageStats {
		stats[k] = v
	}

	return stats, nil
}

// GetUsageStatsSummary 获取使用统计汇总
func (f *FileStorageAdapter) GetUsageStatsSummary(ctx context.Context) (map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var (
		total    int
		disabled int
		success  int64
		failure  int64
		health   float64
		errorSum float64
	)

	for _, state := range f.states {
		total++
		if state.Disabled {
			disabled++
		}
		success += int64(state.SuccessCount)
		failure += int64(state.FailureCount)
		health += state.HealthScore
		errorSum += state.ErrorRate
	}

	summary := map[string]interface{}{
		"total_credentials":    total,
		"active_credentials":   total - disabled,
		"disabled_credentials": disabled,
		"total_success":        success,
		"total_failure":        failure,
		"avg_health_score":     0.0,
		"avg_error_rate":       0.0,
	}

	if total > 0 {
		summary["avg_health_score"] = health / float64(total)
		summary["avg_error_rate"] = errorSum / float64(total)
	}

	return summary, nil
}
