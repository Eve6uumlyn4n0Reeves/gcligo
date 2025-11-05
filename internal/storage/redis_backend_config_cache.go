//go:build !stats_isolation

package storage

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// 从 redis_backend.go 拆分：缓存操作（Cache 部分）

func (r *RedisBackend) GetCache(ctx context.Context, key string) ([]byte, error) {
	ckey := r.prefix + "cache:" + key
	data, err := r.client.Get(ctx, ckey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, &ErrNotFound{Key: key}
		}
		return nil, err
	}
	return data, nil
}

func (r *RedisBackend) SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ckey := r.prefix + "cache:" + key
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return r.client.Set(ctx, ckey, value, ttl).Err()
}

func (r *RedisBackend) DeleteCache(ctx context.Context, key string) error {
	ckey := r.prefix + "cache:" + key
	if res, err := r.client.Del(ctx, ckey).Result(); err != nil {
		return err
	} else if res == 0 {
		return &ErrNotFound{Key: key}
	}
	return nil
}
