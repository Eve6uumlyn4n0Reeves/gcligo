//go:build !stats_isolation

package storage

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

// 从 mongodb_backend.go 拆分：配置相关方法

func (m *MongoDBBackend) GetConfig(ctx context.Context, key string) (interface{}, error) {
	value, err := m.storage.GetConfig(ctx, key)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, &ErrNotFound{Key: key}
		}
		return nil, err
	}
	return value, nil
}

func (m *MongoDBBackend) SetConfig(ctx context.Context, key string, value interface{}) error {
	return m.storage.SetConfig(ctx, key, value)
}

func (m *MongoDBBackend) DeleteConfig(ctx context.Context, key string) error {
	if err := m.storage.DeleteConfig(ctx, key); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return &ErrNotFound{Key: key}
		}
		return err
	}
	return nil
}

func (m *MongoDBBackend) ListConfigs(ctx context.Context) (map[string]interface{}, error) {
	return m.storage.ListConfigs(ctx)
}
