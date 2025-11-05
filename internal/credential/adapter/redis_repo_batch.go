package adapter

import (
	"context"
	"fmt"
)

// EnableCredentials 启用凭证
func (r *RedisStorageAdapter) EnableCredentials(ctx context.Context, credIDs []string) error {
	return r.updateCredentialStatus(ctx, credIDs, false)
}

// DisableCredentials 禁用凭证
func (r *RedisStorageAdapter) DisableCredentials(ctx context.Context, credIDs []string) error {
	return r.updateCredentialStatus(ctx, credIDs, true)
}

// DeleteCredentials 批量删除凭证
func (r *RedisStorageAdapter) DeleteCredentials(ctx context.Context, credIDs []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pipe := r.client.Pipeline()

	for _, id := range credIDs {
		// 删除凭证
		pipe.Del(ctx, r.credKey(id))
		// 删除状态
		pipe.Del(ctx, r.stateKey(id))
		// 删除统计
		pipe.Del(ctx, r.stateKey(id)+":stats")
		// 从集合中移除
		pipe.SRem(ctx, r.credSetKey(), id)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	// 通知观察者
	go r.notifyWatchers()

	return nil
}

// GetHealthyCredentials 获取健康凭证
func (r *RedisStorageAdapter) GetHealthyCredentials(ctx context.Context) ([]*Credential, error) {
	return r.GetCredentialsByFilter(ctx, &CredentialFilter{
		Disabled:  boolPtr(false),
		MinHealth: float64Ptr(0.5),
	})
}

// GetUnhealthyCredentials 获取不健康凭证
func (r *RedisStorageAdapter) GetUnhealthyCredentials(ctx context.Context) ([]*Credential, error) {
	return r.GetCredentialsByFilter(ctx, &CredentialFilter{
		Disabled: boolPtr(true),
	})
}

// GetCredentialsByFilter 根据过滤器获取凭证
func (r *RedisStorageAdapter) GetCredentialsByFilter(ctx context.Context, filter *CredentialFilter) ([]*Credential, error) {
	credentials, err := r.GetAllCredentials(ctx)
	if err != nil {
		return nil, err
	}

	if filter == nil {
		return credentials, nil
	}

	// 应用过滤器
	filtered := make([]*Credential, 0)
	for _, cred := range credentials {
		if filter.Disabled != nil && cred.State != nil {
			if cred.State.Disabled != *filter.Disabled {
				continue
			}
		}

		if filter.MinHealth != nil && cred.State != nil {
			if cred.State.HealthScore < *filter.MinHealth {
				continue
			}
		}

		if filter.MaxHealth != nil && cred.State != nil {
			if cred.State.HealthScore > *filter.MaxHealth {
				continue
			}
		}

		if filter.Type != "" && cred.Type != filter.Type {
			continue
		}

		filtered = append(filtered, cred)
	}

	return filtered, nil
}

// updateCredentialStatus 更新凭证状态
func (r *RedisStorageAdapter) updateCredentialStatus(ctx context.Context, credIDs []string, disabled bool) error {
	pipe := r.client.Pipeline()

	for _, id := range credIDs {
		// 更新状态的 Disabled 字段
		stateKey := r.stateKey(id)
		pipe.HSet(ctx, stateKey, "disabled", disabled)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update credential status: %w", err)
	}

	return nil
}
