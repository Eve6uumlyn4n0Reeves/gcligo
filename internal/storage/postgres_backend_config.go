//go:build !stats_isolation

package storage

import (
	"context"
	"database/sql"
	"errors"
)

// 从 postgres_backend.go 拆分：配置相关方法

func (p *PostgresBackend) GetConfig(ctx context.Context, key string) (interface{}, error) {
	value, err := p.storage.GetConfig(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &ErrNotFound{Key: key}
		}
		return nil, err
	}
	return value, nil
}

func (p *PostgresBackend) SetConfig(ctx context.Context, key string, value interface{}) error {
	return p.storage.SetConfig(ctx, key, value)
}

func (p *PostgresBackend) DeleteConfig(ctx context.Context, key string) error {
	if err := p.storage.DeleteConfig(ctx, key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &ErrNotFound{Key: key}
		}
		return err
	}
	return nil
}

func (p *PostgresBackend) ListConfigs(ctx context.Context) (map[string]interface{}, error) {
	return p.storage.ListConfigs(ctx)
}
