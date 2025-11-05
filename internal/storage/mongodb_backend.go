//go:build !stats_isolation

package storage

import (
	"context"
	"fmt"

	"gcli2api-go/internal/monitoring"
	storagecommon "gcli2api-go/internal/storage/common"
	"gcli2api-go/internal/storage/mongodb"
)

// ✅ MongoDBBackend wraps MongoDB storage implementation
type MongoDBBackend struct {
	storage *mongodb.MongoDBStorage
	adapter storagecommon.BackendAdapter
	// 嵌入通用的"不支持"操作实现，减少重复代码
	storagecommon.UnsupportedCacheOps
	UnsupportedTransactionOps
}

// NewMongoDBBackend creates a MongoDB storage backend
func NewMongoDBBackend(uri, dbName string) (*MongoDBBackend, error) {
	storage, err := mongodb.NewMongoDBStorage(uri, dbName)
	if err != nil {
		return nil, err
	}

	return &MongoDBBackend{
		storage: storage,
		adapter: storagecommon.NewBackendAdapter(),
	}, nil
}

// Initialize initializes MongoDB connection
func (m *MongoDBBackend) Initialize(ctx context.Context) error { return m.storage.Initialize(ctx) }

// Close closes MongoDB connection
func (m *MongoDBBackend) Close() error { return m.storage.Close() }

// PoolStats exposes basic pool metrics to monitoring instrumentation.
func (m *MongoDBBackend) PoolStats(ctx context.Context) (monitoring.StoragePoolStats, error) {
	if m == nil || m.storage == nil {
		return monitoring.StoragePoolStats{}, fmt.Errorf("mongodb storage not initialized")
	}
	active, idle, err := m.storage.PoolStats(ctx)
	if err != nil {
		return monitoring.StoragePoolStats{}, err
	}
	return monitoring.StoragePoolStats{Active: active, Idle: idle, Hits: 0, Misses: 0}, nil
}

// Health pings storage by listing credentials
func (m *MongoDBBackend) Health(ctx context.Context) error {
	_, err := m.storage.ListCredentials(ctx)
	return err
}

// GetCredential retrieves a credential as a generic map
func (m *MongoDBBackend) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	data, err := m.storage.GetCredential(ctx, id)
	if err != nil {
		return nil, err
	}
	return m.adapter.UnmarshalCredential(data)
}

// SetCredential stores a credential from a generic map
func (m *MongoDBBackend) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	b, err := m.adapter.MarshalCredential(data)
	if err != nil {
		return err
	}
	return m.storage.SetCredential(ctx, id, b)
}

// DeleteCredential removes a credential
func (m *MongoDBBackend) DeleteCredential(ctx context.Context, id string) error {
	return m.storage.DeleteCredential(ctx, id)
}

// ListCredentials lists all credential IDs
func (m *MongoDBBackend) ListCredentials(ctx context.Context) ([]string, error) {
	return m.storage.ListCredentials(ctx)
}

// IncrementUsage increments usage counter
// moved to mongodb_backend_usage.go: IncrementUsage

// GetUsage retrieves usage statistics
// moved to mongodb_backend_usage.go: GetUsage

// ResetUsage resets usage statistics
// moved to mongodb_backend_usage.go: ResetUsage

// ListUsage returns empty map (not yet implemented in low-level storage)
// moved to mongodb_backend_usage.go: ListUsage

// Config operations
// moved to mongodb_backend_config.go: GetConfig
// moved to mongodb_backend_config.go: SetConfig
// moved to mongodb_backend_config.go: DeleteConfig
// moved to mongodb_backend_config.go: ListConfigs

// Cache operations (not supported)
// 使用嵌入的 UnsupportedCacheOps 提供默认实现，无需重复代码

// Batch operations for performance
// moved to mongodb_backend_batch.go: BatchGetCredentials

// moved to mongodb_backend_batch.go: BatchSetCredentials

// moved to mongodb_backend_batch.go: BatchDeleteCredentials

// Transaction support (not supported for MongoDB in this implementation)
// 使用嵌入的 UnsupportedTransactionOps 提供默认实现，无需重复代码

// ExportData exports all data for backup
func (m *MongoDBBackend) ExportData(ctx context.Context) (map[string]interface{}, error) {
	return exportDataCommon(ctx, "mongodb", m)
}

// ImportData imports data from backup
func (m *MongoDBBackend) ImportData(ctx context.Context, data map[string]interface{}) error {
	return importDataCommon(ctx, m, data)
}

// GetStorageStats returns storage statistics
func (m *MongoDBBackend) GetStorageStats(ctx context.Context) (StorageStats, error) {
	return storageStatsCommon(ctx, "mongodb", m)
}
