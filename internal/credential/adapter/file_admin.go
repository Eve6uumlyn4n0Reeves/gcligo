package adapter

import (
	"context"
	"fmt"
	"time"
)

// EnableCredentials 启用凭证
func (f *FileStorageAdapter) EnableCredentials(ctx context.Context, credIDs []string) error {
	return f.updateCredentialStatus(credIDs, false)
}

// DisableCredentials 禁用凭证
func (f *FileStorageAdapter) DisableCredentials(ctx context.Context, credIDs []string) error {
	return f.updateCredentialStatus(credIDs, true)
}

// DeleteCredentials 批量删除凭证
func (f *FileStorageAdapter) DeleteCredentials(ctx context.Context, credIDs []string) error {
	for _, id := range credIDs {
		if err := f.DeleteCredential(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// GetHealthyCredentials 获取健康凭证
func (f *FileStorageAdapter) GetHealthyCredentials(ctx context.Context) ([]*Credential, error) {
	return f.GetCredentialsByFilter(ctx, &CredentialFilter{
		Disabled:  boolPtr(false),
		MinHealth: float64Ptr(0.5),
	})
}

// GetUnhealthyCredentials 获取不健康凭证
func (f *FileStorageAdapter) GetUnhealthyCredentials(ctx context.Context) ([]*Credential, error) {
	return f.GetCredentialsByFilter(ctx, &CredentialFilter{
		Disabled: boolPtr(true),
	})
}

// ValidateCredential 验证凭证
func (f *FileStorageAdapter) ValidateCredential(ctx context.Context, cred *Credential) error {
	if cred.ID == "" {
		return fmt.Errorf("credential ID cannot be empty")
	}
	if cred.Type == "" {
		return fmt.Errorf("credential type cannot be empty")
	}

	switch cred.Type {
	case "oauth":
		if cred.Token == "" && cred.RefreshToken == "" {
			return fmt.Errorf("OAuth credential must have token or refresh token")
		}
	case "api_key":
		if cred.Token == "" {
			return fmt.Errorf("API key credential must have token")
		}
	case "service_account":
		if cred.ClientID == "" {
			return fmt.Errorf("Service account must have client ID")
		}
	}

	return nil
}

// GetCredentialsByFilter 根据过滤器获取凭证
func (f *FileStorageAdapter) GetCredentialsByFilter(ctx context.Context, filter *CredentialFilter) ([]*Credential, error) {
	creds, err := f.GetAllCredentials(ctx)
	if err != nil || filter == nil {
		return creds, err
	}

	result := make([]*Credential, 0, len(creds))
	for _, cred := range creds {
		if f.matchesFilter(cred, filter) {
			result = append(result, cred)
		}
	}
	return result, nil
}

func (f *FileStorageAdapter) updateCredentialStatus(credIDs []string, disabled bool) error {
	f.mu.Lock()
	for _, id := range credIDs {
		state, ok := f.states[id]
		if !ok {
			continue
		}
		state.Disabled = disabled
		state.UpdatedAt = time.Now()
		if err := f.saveStateFile(state); err != nil {
			f.mu.Unlock()
			return fmt.Errorf("failed to persist state for %s: %w", id, err)
		}
	}
	f.mu.Unlock()

	f.notifyWatchers()

	return nil
}

func (f *FileStorageAdapter) matchesFilter(cred *Credential, filter *CredentialFilter) bool {
	if filter == nil {
		return true
	}

	state := cred.State
	if filter.Disabled != nil && state != nil && state.Disabled != *filter.Disabled {
		return false
	}
	if filter.Type != "" && cred.Type != filter.Type {
		return false
	}
	if filter.MinHealth != nil && state != nil && state.HealthScore < *filter.MinHealth {
		return false
	}
	if filter.MaxError != nil && state != nil && state.ErrorRate > *filter.MaxError {
		return false
	}
	if filter.LastUsed != nil && state != nil {
		if state.LastUsed == nil || state.LastUsed.Before(*filter.LastUsed) {
			return false
		}
	}

	return true
}
