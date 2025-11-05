package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// StoreCredential 存储凭证
func (r *RedisStorageAdapter) StoreCredential(ctx context.Context, cred *Credential) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 设置创建时间
	now := time.Now()
	if cred.State == nil {
		cred.State = &CredentialState{
			ID:          cred.ID,
			HealthScore: 1.0,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	cred.State.UpdatedAt = now

	// 序列化凭证
	credData, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("failed to marshal credential: %w", err)
	}

	// 存储凭证
	credKey := r.credKey(cred.ID)
	if err := r.client.Set(ctx, credKey, credData, r.keyTTL).Err(); err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	// 存储状态
	stateKey := r.stateKey(cred.ID)
	stateData, err := json.Marshal(cred.State)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := r.client.Set(ctx, stateKey, stateData, r.stateKeyTTL).Err(); err != nil {
		return fmt.Errorf("failed to store state: %w", err)
	}

	// 添加到凭证集合
	if err := r.client.SAdd(ctx, r.credSetKey(), cred.ID).Err(); err != nil {
		return fmt.Errorf("failed to add to credential set: %w", err)
	}

	// 通知观察者
	go r.notifyWatchers()

	return nil
}

// LoadCredential 加载凭证
func (r *RedisStorageAdapter) LoadCredential(ctx context.Context, id string) (*Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	credKey := r.credKey(id)
	credData, err := r.client.Get(ctx, credKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("credential not found: %s", id)
		}
		return nil, fmt.Errorf("failed to load credential: %w", err)
	}

	var cred Credential
	if err := json.Unmarshal(credData, &cred); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credential: %w", err)
	}

	// 加载状态
	stateKey := r.stateKey(id)
	stateData, err := r.client.Get(ctx, stateKey).Bytes()
	if err != nil {
		if err != redis.Nil {
			return nil, fmt.Errorf("failed to load state: %w", err)
		}
		// 如果状态不存在，创建默认状态
		cred.State = &CredentialState{
			ID:          cred.ID,
			HealthScore: 1.0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	} else {
		if err := json.Unmarshal(stateData, &cred.State); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	return &cred, nil
}

// DeleteCredential 删除凭证
func (r *RedisStorageAdapter) DeleteCredential(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 删除凭证
	credKey := r.credKey(id)
	if err := r.client.Del(ctx, credKey).Err(); err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	// 删除状态
	stateKey := r.stateKey(id)
	if err := r.client.Del(ctx, stateKey).Err(); err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	// 从集合中移除
	if err := r.client.SRem(ctx, r.credSetKey(), id).Err(); err != nil {
		return fmt.Errorf("failed to remove from credential set: %w", err)
	}

	// 通知观察者
	go r.notifyWatchers()

	return nil
}

// UpdateCredential 更新凭证
func (r *RedisStorageAdapter) UpdateCredential(ctx context.Context, cred *Credential) error {
	return r.StoreCredential(ctx, cred)
}

// GetAllCredentials 获取所有凭证
func (r *RedisStorageAdapter) GetAllCredentials(ctx context.Context) ([]*Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 获取所有凭证ID
	credIDs, err := r.client.SMembers(ctx, r.credSetKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get credential IDs: %w", err)
	}

	if len(credIDs) == 0 {
		return []*Credential{}, nil
	}

	// 批量获取凭证
	credentials := make([]*Credential, 0, len(credIDs))
	for _, id := range credIDs {
		cred, err := r.LoadCredential(ctx, id)
		if err != nil {
			// 跳过加载失败的凭证
			continue
		}
		credentials = append(credentials, cred)
	}

	return credentials, nil
}

// DiscoverCredentials 发现凭证
func (r *RedisStorageAdapter) DiscoverCredentials(ctx context.Context) ([]*Credential, error) {
	// Redis中不需要发现，所有凭证都在集合中
	return r.GetAllCredentials(ctx)
}

// RefreshCredential 刷新凭证
func (r *RedisStorageAdapter) RefreshCredential(ctx context.Context, credID string) (*Credential, error) {
	// 重新加载凭证，相当于刷新TTL
	cred, err := r.LoadCredential(ctx, credID)
	if err != nil {
		return nil, err
	}

	// 重新存储以刷新TTL
	if err := r.StoreCredential(ctx, cred); err != nil {
		return nil, err
	}

	return cred, nil
}

// ValidateCredential 验证凭证
func (r *RedisStorageAdapter) ValidateCredential(ctx context.Context, cred *Credential) error {
	// 基础验证
	if cred.ID == "" {
		return fmt.Errorf("credential ID cannot be empty")
	}
	if cred.Type == "" {
		return fmt.Errorf("credential type cannot be empty")
	}

	// 根据类型验证
	switch cred.Type {
	case "oauth":
		if cred.AccessToken == "" {
			return fmt.Errorf("OAuth credential must have access token")
		}
	case "api_key":
		if cred.APIKey == "" {
			return fmt.Errorf("API key credential must have API key")
		}
	default:
		return fmt.Errorf("unsupported credential type: %s", cred.Type)
	}

	return nil
}
