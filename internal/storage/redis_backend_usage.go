//go:build !stats_isolation

package storage

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
)

// 从 redis_backend.go 拆分：用量与配置/缓存相关方法（Usage 部分）

// IncrementUsage increments a usage counter
func (r *RedisBackend) IncrementUsage(ctx context.Context, key string, field string, value int64) error {
	hashKey := r.prefix + "usage:" + key
	return r.client.HIncrBy(ctx, hashKey, field, value).Err()
}

// GetUsage retrieves usage statistics
func (r *RedisBackend) GetUsage(ctx context.Context, key string) (map[string]interface{}, error) {
	hashKey := r.prefix + "usage:" + key
	data, err := r.client.HGetAll(ctx, hashKey).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]interface{})
	for k, v := range data {
		result[k] = v
	}
	return result, nil
}

// ResetUsage clears usage statistics
func (r *RedisBackend) ResetUsage(ctx context.Context, key string) error {
	hashKey := r.prefix + "usage:" + key
	return r.client.Del(ctx, hashKey).Err()
}

// ListUsage lists all usage records
func (r *RedisBackend) ListUsage(ctx context.Context) (map[string]map[string]interface{}, error) {
	pattern := r.prefix + "usage:*"
	result := make(map[string]map[string]interface{})
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		usageKey := strings.TrimPrefix(key, r.prefix+"usage:")
		data, err := r.client.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		entry := make(map[string]interface{}, len(data))
		for k, v := range data {
			entry[k] = v
		}
		result[usageKey] = entry
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// Config operations
func (r *RedisBackend) GetConfig(ctx context.Context, key string) (interface{}, error) {
	value, err := r.client.HGet(ctx, r.prefix+"config", key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, &ErrNotFound{Key: key}
		}
		return nil, err
	}
	if doc, err := r.adapter.UnmarshalDocument(key, value); err == nil && len(doc) > 0 {
		return doc, nil
	}
	if out, err := r.adapter.UnmarshalValue(value); err == nil {
		return out, nil
	}
	return string(value), nil
}

func (r *RedisBackend) SetConfig(ctx context.Context, key string, value interface{}) error {
	var payload []byte
	var err error
	if doc, ok := value.(map[string]interface{}); ok {
		payload, err = r.adapter.MarshalDocument(key, doc)
	} else {
		payload, err = r.adapter.MarshalValue(value)
	}
	if err != nil {
		return err
	}
	return r.client.HSet(ctx, r.prefix+"config", key, payload).Err()
}

func (r *RedisBackend) DeleteConfig(ctx context.Context, key string) error {
	res, err := r.client.HDel(ctx, r.prefix+"config", key).Result()
	if err != nil {
		return err
	}
	if res == 0 {
		return &ErrNotFound{Key: key}
	}
	return nil
}

func (r *RedisBackend) ListConfigs(ctx context.Context) (map[string]interface{}, error) {
	data, err := r.client.HGetAll(ctx, r.prefix+"config").Result()
	if err != nil {
		return nil, err
	}
	configs := make(map[string]interface{}, len(data))
	for k, v := range data {
		payload := []byte(v)
		if doc, err := r.adapter.UnmarshalDocument(k, payload); err == nil && len(doc) > 0 {
			configs[k] = doc
			continue
		}
		if decoded, err := r.adapter.UnmarshalValue(payload); err == nil {
			configs[k] = decoded
		} else {
			configs[k] = v
		}
	}
	return configs, nil
}
