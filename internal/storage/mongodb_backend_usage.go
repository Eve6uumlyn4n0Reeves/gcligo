//go:build !stats_isolation

package storage

import (
	"context"
)

// 从 mongodb_backend.go 拆分：用量相关方法

func (m *MongoDBBackend) IncrementUsage(ctx context.Context, key string, field string, value int64) error {
	return m.storage.IncrementUsage(ctx, key, field, value)
}

func (m *MongoDBBackend) GetUsage(ctx context.Context, key string) (map[string]interface{}, error) {
	return m.storage.GetUsage(ctx, key)
}

func (m *MongoDBBackend) ResetUsage(ctx context.Context, key string) error {
	return m.storage.ResetUsage(ctx, key)
}

func (m *MongoDBBackend) ListUsage(ctx context.Context) (map[string]map[string]interface{}, error) {
	return m.storage.ListUsage(ctx)
}
