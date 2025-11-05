//go:build !stats_isolation

package storage

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// 从 redis_backend.go 拆分：批量操作（Batch 部分）

func (r *RedisBackend) BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	result := make(map[string]map[string]interface{})
	pipe := r.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd)
	for _, id := range ids {
		key := r.prefix + "cred:" + id
		cmds[id] = pipe.Get(ctx, key)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}
	for id, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return nil, err
		}
		decoded, err := r.adapter.UnmarshalCredential([]byte(data))
		if err != nil {
			return nil, err
		}
		result[id] = decoded
	}
	return result, nil
}

func (r *RedisBackend) BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error {
	items, err := r.adapter.BatchMarshalCredentials(data)
	if err != nil {
		return err
	}
	pipe := r.client.Pipeline()
	for id, payload := range items {
		key := r.prefix + "cred:" + id
		pipe.Set(ctx, key, payload, 0)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisBackend) BatchDeleteCredentials(ctx context.Context, ids []string) error {
	pipe := r.client.Pipeline()
	for _, id := range ids {
		pipe.Del(ctx, r.prefix+"cred:"+id)
	}
	_, err := pipe.Exec(ctx)
	return err
}
