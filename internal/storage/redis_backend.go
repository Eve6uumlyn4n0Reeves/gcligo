//go:build !stats_isolation

package storage

import (
	"context"
	"fmt"
	"time"

	"gcli2api-go/internal/monitoring"
	storagecommon "gcli2api-go/internal/storage/common"
	"github.com/redis/go-redis/v9"
)

// ✅ RedisBackend implements Storage interface using Redis
type RedisBackend struct {
	client  *redis.Client
	prefix  string
	adapter storagecommon.BackendAdapter
	// 嵌入通用的"不支持"操作实现，减少重复代码
	UnsupportedTransactionOps
}

// NewRedisBackend creates a new Redis storage backend
func NewRedisBackend(addr, password string, db int, prefix string) (*RedisBackend, error) {
	if prefix == "" {
		prefix = "gcli2api:"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	return &RedisBackend{
		client:  client,
		prefix:  prefix,
		adapter: storagecommon.NewBackendAdapter(),
	}, nil
}

// Initialize tests Redis connection
func (r *RedisBackend) Initialize(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}
	return nil
}

// Close closes Redis connection
func (r *RedisBackend) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// Health checks redis availability
func (r *RedisBackend) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// GetCredential retrieves a credential
func (r *RedisBackend) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	key := r.prefix + "cred:" + id
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, &ErrNotFound{Key: id}
		}
		return nil, err
	}
	return r.adapter.UnmarshalCredential(data)
}

// SetCredential stores a credential
func (r *RedisBackend) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	key := r.prefix + "cred:" + id
	payload, err := r.adapter.MarshalCredential(data)
	if err != nil {
		return fmt.Errorf("failed to encode credential %s: %w", id, err)
	}
	return r.client.Set(ctx, key, payload, 0).Err()
}

// DeleteCredential removes a credential
func (r *RedisBackend) DeleteCredential(ctx context.Context, id string) error {
	key := r.prefix + "cred:" + id
	return r.client.Del(ctx, key).Err()
}

// ListCredentials lists all credential IDs
func (r *RedisBackend) ListCredentials(ctx context.Context) ([]string, error) {
	pattern := r.prefix + "cred:*"
	var ids []string

	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		id := key[len(r.prefix+"cred:"):]
		ids = append(ids, id)
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

// IncrementUsage increments a usage counter
// moved to redis_backend_usage.go: IncrementUsage

// GetUsage retrieves usage statistics
// moved to redis_backend_usage.go: GetUsage

// ResetUsage clears usage statistics
// moved to redis_backend_usage.go: ResetUsage

// ListUsage lists all usage records
// moved to redis_backend_usage.go: ListUsage

// Config operations
// moved to redis_backend_usage.go: GetConfig

// moved to redis_backend_usage.go: SetConfig

// moved to redis_backend_usage.go: DeleteConfig

// moved to redis_backend_usage.go: ListConfigs

// Cache operations (not supported)
// moved to redis_backend_config_cache.go: GetCache

// moved to redis_backend_config_cache.go: SetCache

// moved to redis_backend_config_cache.go: DeleteCache

// Batch operations for performance
// moved to redis_backend_batch.go: BatchGetCredentials

// moved to redis_backend_batch.go: BatchSetCredentials

// moved to redis_backend_batch.go: BatchDeleteCredentials

// Transaction support (not implemented yet)
// 使用嵌入的 UnsupportedTransactionOps 提供默认实现，无需重复代码

// ExportData exports all data for backup
func (r *RedisBackend) ExportData(ctx context.Context) (map[string]interface{}, error) {
	return exportDataCommon(ctx, "redis", r)
}

// ImportData imports data from backup
func (r *RedisBackend) ImportData(ctx context.Context, data map[string]interface{}) error {
	return importDataCommon(ctx, r, data)
}

// PoolStats returns snapshot statistics about the Redis connection pool.
func (r *RedisBackend) PoolStats(ctx context.Context) (monitoring.StoragePoolStats, error) {
	if r.client == nil {
		return monitoring.StoragePoolStats{}, fmt.Errorf("redis client not initialized")
	}
	stats := r.client.PoolStats()
	active := int64(stats.TotalConns - stats.IdleConns)
	if active < 0 {
		active = 0
	}
	return monitoring.StoragePoolStats{
		Active: active,
		Idle:   int64(stats.IdleConns),
		Hits:   int64(stats.Hits),
		Misses: int64(stats.Misses),
	}, nil
}

// GetStorageStats returns storage statistics
func (r *RedisBackend) GetStorageStats(ctx context.Context) (StorageStats, error) {
	stats, err := storageStatsCommon(ctx, "redis", r)
	if err != nil {
		return stats, err
	}

	// Get Redis info for connection count
	info, err := r.client.Info(ctx, "clients").Result()
	if err == nil {
		// 简单解析连接数（实际应用中可能需要更复杂的解析）
		stats.Details = map[string]interface{}{
			"redis_info": info,
		}
	}

	return stats, nil
}
