//go:build !stats_isolation

package storage

import "context"

// 从 mongodb_backend.go 拆分：批量相关方法

func (m *MongoDBBackend) BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	if len(ids) == 0 {
		return map[string]map[string]interface{}{}, nil
	}

	raw, err := m.storage.BulkGetCredentials(ctx, ids)
	if err != nil {
		return nil, err
	}

	return m.adapter.BatchUnmarshalCredentials(raw)
}

func (m *MongoDBBackend) BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error {
	items, err := m.adapter.BatchMarshalCredentials(data)
	if err != nil {
		return err
	}
	return m.storage.BulkUpsertCredentials(ctx, items)
}

func (m *MongoDBBackend) BatchDeleteCredentials(ctx context.Context, ids []string) error {
	return m.storage.BulkDeleteCredentials(ctx, ids)
}
