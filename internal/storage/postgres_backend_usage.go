//go:build !stats_isolation

package storage

import (
	"context"
	"database/sql"
	"errors"
)

// 从 postgres_backend.go 拆分：用量相关方法

func (p *PostgresBackend) IncrementUsage(ctx context.Context, key string, field string, value int64) error {
	return p.storage.IncrementUsage(ctx, key, field, value)
}

func (p *PostgresBackend) GetUsage(ctx context.Context, key string) (map[string]interface{}, error) {
	usage, err := p.storage.GetUsage(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &ErrNotFound{Key: key}
		}
		return nil, err
	}
	return usage, nil
}

func (p *PostgresBackend) ResetUsage(ctx context.Context, key string) error {
	return p.storage.ResetUsage(ctx, key)
}

func (p *PostgresBackend) ListUsage(ctx context.Context) (map[string]map[string]interface{}, error) {
	return p.storage.ListUsage(ctx)
}
