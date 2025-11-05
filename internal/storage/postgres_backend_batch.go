//go:build !stats_isolation

package storage

import "context"

// 从 postgres_backend.go 拆分：批量相关方法

func (p *PostgresBackend) BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	if len(ids) == 0 {
		return map[string]map[string]interface{}{}, nil
	}

	creds, err := p.storage.BatchGetCredentials(ctx, ids)
	if err != nil {
		return nil, err
	}

	return p.adapter.BatchCredentialsFromStruct(creds)
}

func (p *PostgresBackend) BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error {
	items, err := p.adapter.BatchCredentialsToStruct(data)
	if err != nil {
		return err
	}
	return p.storage.BatchSaveCredentials(ctx, items)
}

func (p *PostgresBackend) BatchDeleteCredentials(ctx context.Context, ids []string) error {
	return p.storage.BatchDeleteCredentials(ctx, ids)
}
