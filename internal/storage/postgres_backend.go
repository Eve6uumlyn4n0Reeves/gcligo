//go:build !stats_isolation

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"gcli2api-go/internal/monitoring"
	storagecommon "gcli2api-go/internal/storage/common"
	"gcli2api-go/internal/storage/postgres"
)

// ✅ PostgresBackend wraps PostgreSQL storage implementation
type PostgresBackend struct {
	storage *postgres.PostgresStorage
	adapter storagecommon.BackendAdapter
	// 嵌入通用的"不支持"操作实现，减少重复代码
	storagecommon.UnsupportedCacheOps
}

// NewPostgresBackend creates a PostgreSQL storage backend
func NewPostgresBackend(dsn string) (*PostgresBackend, error) {
	storage, err := postgres.NewPostgresStorage(dsn)
	if err != nil {
		return nil, err
	}

	return &PostgresBackend{
		storage: storage,
		adapter: storagecommon.NewBackendAdapter(),
	}, nil
}

// Initialize initializes PostgreSQL connection
func (p *PostgresBackend) Initialize(ctx context.Context) error {
	return p.storage.Initialize(ctx)
}

// Close closes PostgreSQL connection
func (p *PostgresBackend) Close() error {
	return p.storage.Close()
}

// Health checks connectivity via a simple metadata read
func (p *PostgresBackend) Health(ctx context.Context) error {
	_, err := p.storage.ListCredentials(ctx)
	return err
}

// GetCredential retrieves a credential
func (p *PostgresBackend) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	cred, err := p.storage.GetCredential(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &ErrNotFound{Key: id}
		}
		return nil, err
	}
	return p.adapter.CredentialFromStruct(cred)
}

// SetCredential stores a credential
func (p *PostgresBackend) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	cred, err := p.adapter.CredentialToStruct(data)
	if err != nil {
		return fmt.Errorf("invalid credential json: %w", err)
	}
	return p.storage.SaveCredential(ctx, id, cred)
}

// DeleteCredential removes a credential
func (p *PostgresBackend) DeleteCredential(ctx context.Context, id string) error {
	return p.storage.DeleteCredential(ctx, id)
}

// ListCredentials lists all credentials
func (p *PostgresBackend) ListCredentials(ctx context.Context) ([]string, error) {
	return p.storage.ListCredentials(ctx)
}

// IncrementUsage increments usage counter
// moved to postgres_backend_usage.go: IncrementUsage

// GetUsage retrieves usage statistics
// moved to postgres_backend_usage.go: GetUsage

// ResetUsage resets usage statistics
// moved to postgres_backend_usage.go: ResetUsage

// ListUsage returns all usage records (not supported)
// moved to postgres_backend_usage.go: ListUsage

// Config operations (not supported)
// moved to postgres_backend_config.go: GetConfig
// moved to postgres_backend_config.go: SetConfig
// moved to postgres_backend_config.go: DeleteConfig
// moved to postgres_backend_config.go: ListConfigs

// Cache operations (not supported)
// 使用嵌入的 UnsupportedCacheOps 提供默认实现，无需重复代码

// Batch operations for performance
// moved to postgres_backend_batch.go: BatchGetCredentials

// moved to postgres_backend_batch.go: BatchSetCredentials

// moved to postgres_backend_batch.go: BatchDeleteCredentials

// ExportData exports all data for backup
func (p *PostgresBackend) ExportData(ctx context.Context) (map[string]interface{}, error) {
	return exportDataCommon(ctx, "postgres", p)
}

// ImportData imports data from backup
func (p *PostgresBackend) ImportData(ctx context.Context, data map[string]interface{}) error {
	return importDataCommon(ctx, p, data)
}

// GetStorageStats returns storage statistics
func (p *PostgresBackend) GetStorageStats(ctx context.Context) (StorageStats, error) {
	return storageStatsCommon(ctx, "postgres", p)
}

// PoolStats returns snapshot statistics about the PostgreSQL connection pool.
func (p *PostgresBackend) PoolStats(ctx context.Context) (monitoring.StoragePoolStats, error) {
	if p == nil || p.storage == nil {
		return monitoring.StoragePoolStats{}, fmt.Errorf("postgres storage not initialized")
	}
	active, idle, misses := p.storage.PoolStats()
	return monitoring.StoragePoolStats{
		Active: active,
		Idle:   idle,
		Hits:   0,      // PostgreSQL driver doesn't provide cache hits
		Misses: misses, // WaitCount represents connection wait events
	}, nil
}
