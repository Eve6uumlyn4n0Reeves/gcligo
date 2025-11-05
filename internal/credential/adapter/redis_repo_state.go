package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// GetAllCredentialStates 获取所有凭证状态
func (r *RedisStorageAdapter) GetAllCredentialStates(ctx context.Context) (map[string]*CredentialState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 获取所有凭证ID
	credIDs, err := r.client.SMembers(ctx, r.credSetKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get credential IDs: %w", err)
	}

	states := make(map[string]*CredentialState)
	for _, id := range credIDs {
		stateKey := r.stateKey(id)
		stateData, err := r.client.Get(ctx, stateKey).Bytes()
		if err != nil {
			if err != redis.Nil {
				continue
			}
			// 创建默认状态
			states[id] = &CredentialState{
				ID:          id,
				HealthScore: 1.0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
		} else {
			var state CredentialState
			if err := json.Unmarshal(stateData, &state); err != nil {
				continue
			}
			states[id] = &state
		}
	}

	return states, nil
}

// UpdateCredentialStates 批量更新凭证状态
func (r *RedisStorageAdapter) UpdateCredentialStates(ctx context.Context, states map[string]*CredentialState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 使用管道批量更新
	pipe := r.client.Pipeline()

	for id, state := range states {
		// 更新时间戳
		state.UpdatedAt = time.Now()

		// 序列化状态
		stateData, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("failed to marshal state for %s: %w", id, err)
		}

		// 添加到管道
		stateKey := r.stateKey(id)
		pipe.Set(ctx, stateKey, stateData, r.stateKeyTTL)
	}

	// 执行管道
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update states: %w", err)
	}

	return nil
}

// UpdateUsageStats 更新使用统计
func (r *RedisStorageAdapter) UpdateUsageStats(ctx context.Context, credID string, stats map[string]interface{}) error {
	stateKey := r.stateKey(credID)

	// 使用Redis HINCRBY 原子操作更新计数器
	pipe := r.client.Pipeline()

	for key, value := range stats {
		switch v := value.(type) {
		case int:
			pipe.HIncrBy(ctx, stateKey+":stats", key, int64(v))
		case int64:
			pipe.HIncrBy(ctx, stateKey+":stats", key, v)
		case float64:
			// 对于浮点数，使用字符串存储
			pipe.HSet(ctx, stateKey+":stats", key, fmt.Sprintf("%.6f", v))
		case string:
			pipe.HSet(ctx, stateKey+":stats", key, v)
		}
	}

	// 设置统计数据的过期时间
	pipe.Expire(ctx, stateKey+":stats", r.stateKeyTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update usage stats: %w", err)
	}

	return nil
}

// GetUsageStats 获取使用统计
func (r *RedisStorageAdapter) GetUsageStats(ctx context.Context, credID string) (map[string]interface{}, error) {
	stateKey := r.stateKey(credID) + ":stats"

	result, err := r.client.HGetAll(ctx, stateKey).Result()
	if err != nil {
		if err == redis.Nil {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	stats := make(map[string]interface{})
	for key, value := range result {
		// 尝试解析为数字
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			stats[key] = intVal
		} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			stats[key] = floatVal
		} else {
			stats[key] = value
		}
	}

	return stats, nil
}

// GetUsageStatsSummary 获取使用统计汇总
func (r *RedisStorageAdapter) GetUsageStatsSummary(ctx context.Context) (map[string]interface{}, error) {
	// 获取所有凭证
	credentials, err := r.GetAllCredentials(ctx)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"total_credentials": len(credentials),
		"total_requests":    int64(0),
		"total_tokens":      int64(0),
		"total_errors":      int64(0),
	}

	// 汇总所有凭证的统计
	for _, cred := range credentials {
		stats, err := r.GetUsageStats(ctx, cred.ID)
		if err != nil {
			continue
		}

		if requests, ok := stats["requests"].(int64); ok {
			summary["total_requests"] = summary["total_requests"].(int64) + requests
		}
		if tokens, ok := stats["tokens"].(int64); ok {
			summary["total_tokens"] = summary["total_tokens"].(int64) + tokens
		}
		if errors, ok := stats["errors"].(int64); ok {
			summary["total_errors"] = summary["total_errors"].(int64) + errors
		}
	}

	return summary, nil
}
