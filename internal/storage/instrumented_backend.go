package storage

import (
	"context"
	"time"

	"gcli2api-go/internal/monitoring"
	"gcli2api-go/internal/monitoring/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// StoragePoolStatsProvider can optionally expose pool statistics for a backend.
type StoragePoolStatsProvider interface {
	PoolStats(context.Context) (monitoring.StoragePoolStats, error)
}

// WithInstrumentation wraps a backend with tracing and metrics instrumentation.
func WithInstrumentation(inner Backend, metrics *monitoring.EnhancedMetrics, label string) Backend {
	if inner == nil || metrics == nil {
		return inner
	}
	if label == "" {
		label = "unknown"
	}
	return &instrumentedBackend{
		Backend: inner,
		metrics: metrics,
		label:   label,
	}
}

type instrumentedBackend struct {
	Backend
	metrics *monitoring.EnhancedMetrics
	label   string
}

func (i *instrumentedBackend) GetConfig(ctx context.Context, key string) (interface{}, error) {
	var result interface{}
	err := i.instrument(ctx, "get_config", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.Backend.GetConfig(ctx, key)
		return innerErr
	})
	return result, err
}

func (i *instrumentedBackend) SetConfig(ctx context.Context, key string, value interface{}) error {
	return i.instrument(ctx, "set_config", func(ctx context.Context) error {
		return i.Backend.SetConfig(ctx, key, value)
	})
}

func (i *instrumentedBackend) DeleteConfig(ctx context.Context, key string) error {
	return i.instrument(ctx, "delete_config", func(ctx context.Context) error {
		return i.Backend.DeleteConfig(ctx, key)
	})
}

func (i *instrumentedBackend) ListConfigs(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := i.instrument(ctx, "list_configs", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.Backend.ListConfigs(ctx)
		return innerErr
	})
	return result, err
}

func (i *instrumentedBackend) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := i.instrument(ctx, "get_credential", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.Backend.GetCredential(ctx, id)
		return innerErr
	})
	return result, err
}

func (i *instrumentedBackend) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	return i.instrument(ctx, "set_credential", func(ctx context.Context) error {
		return i.Backend.SetCredential(ctx, id, data)
	})
}

func (i *instrumentedBackend) DeleteCredential(ctx context.Context, id string) error {
	return i.instrument(ctx, "delete_credential", func(ctx context.Context) error {
		return i.Backend.DeleteCredential(ctx, id)
	})
}

func (i *instrumentedBackend) ListCredentials(ctx context.Context) ([]string, error) {
	var result []string
	err := i.instrument(ctx, "list_credentials", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.Backend.ListCredentials(ctx)
		return innerErr
	})
	return result, err
}

func (i *instrumentedBackend) IncrementUsage(ctx context.Context, key string, field string, delta int64) error {
	return i.instrument(ctx, "increment_usage", func(ctx context.Context) error {
		return i.Backend.IncrementUsage(ctx, key, field, delta)
	})
}

func (i *instrumentedBackend) GetUsage(ctx context.Context, key string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := i.instrument(ctx, "get_usage", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.Backend.GetUsage(ctx, key)
		return innerErr
	})
	return result, err
}

func (i *instrumentedBackend) ResetUsage(ctx context.Context, key string) error {
	return i.instrument(ctx, "reset_usage", func(ctx context.Context) error {
		return i.Backend.ResetUsage(ctx, key)
	})
}

func (i *instrumentedBackend) ListUsage(ctx context.Context) (map[string]map[string]interface{}, error) {
	var result map[string]map[string]interface{}
	err := i.instrument(ctx, "list_usage", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.Backend.ListUsage(ctx)
		return innerErr
	})
	return result, err
}

func (i *instrumentedBackend) GetCache(ctx context.Context, key string) ([]byte, error) {
	var result []byte
	err := i.instrument(ctx, "get_cache", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.Backend.GetCache(ctx, key)
		return innerErr
	})
	return result, err
}

func (i *instrumentedBackend) SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return i.instrument(ctx, "set_cache", func(ctx context.Context) error {
		return i.Backend.SetCache(ctx, key, value, ttl)
	})
}

func (i *instrumentedBackend) DeleteCache(ctx context.Context, key string) error {
	return i.instrument(ctx, "delete_cache", func(ctx context.Context) error {
		return i.Backend.DeleteCache(ctx, key)
	})
}

func (i *instrumentedBackend) BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	var result map[string]map[string]interface{}
	err := i.instrument(ctx, "batch_get_credentials", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.Backend.BatchGetCredentials(ctx, ids)
		return innerErr
	})
	return result, err
}

func (i *instrumentedBackend) BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error {
	return i.instrument(ctx, "batch_set_credentials", func(ctx context.Context) error {
		return i.Backend.BatchSetCredentials(ctx, data)
	})
}

func (i *instrumentedBackend) BatchDeleteCredentials(ctx context.Context, ids []string) error {
	return i.instrument(ctx, "batch_delete_credentials", func(ctx context.Context) error {
		return i.Backend.BatchDeleteCredentials(ctx, ids)
	})
}

func (i *instrumentedBackend) BeginTransaction(ctx context.Context) (Transaction, error) {
	var tx Transaction
	err := i.instrument(ctx, "begin_transaction", func(ctx context.Context) error {
		var innerErr error
		tx, innerErr = i.Backend.BeginTransaction(ctx)
		return innerErr
	})
	if err != nil || tx == nil {
		return tx, err
	}
	return &instrumentedTransaction{Transaction: tx, backend: i}, nil
}

func (i *instrumentedBackend) ExportData(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := i.instrument(ctx, "export_data", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.Backend.ExportData(ctx)
		return innerErr
	})
	return result, err
}

func (i *instrumentedBackend) ImportData(ctx context.Context, data map[string]interface{}) error {
	return i.instrument(ctx, "import_data", func(ctx context.Context) error {
		return i.Backend.ImportData(ctx, data)
	})
}

func (i *instrumentedBackend) GetStorageStats(ctx context.Context) (StorageStats, error) {
	var result StorageStats
	err := i.instrument(ctx, "get_storage_stats", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.Backend.GetStorageStats(ctx)
		return innerErr
	})
	return result, err
}

func (i *instrumentedBackend) instrument(ctx context.Context, operation string, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := tracing.StartSpan(ctx, "storage", i.label+"/"+operation)
	span.SetAttributes(
		attribute.String("storage.backend", i.label),
		attribute.String("storage.operation", operation),
	)
	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()

	if i.metrics != nil {
		i.metrics.RecordStorageOperation(i.label, operation, duration, err)
		if provider, ok := i.Backend.(StoragePoolStatsProvider); ok {
			if stats, statsErr := provider.PoolStats(ctx); statsErr == nil {
				i.metrics.UpdateStoragePoolStats(i.label, stats)
			}
		}
	}
	return err
}

type instrumentedTransaction struct {
	Transaction
	backend *instrumentedBackend
}

func (t *instrumentedTransaction) GetConfig(ctx context.Context, key string) (interface{}, error) {
	var result interface{}
	err := t.backend.instrument(ctx, "tx_get_config", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = t.Transaction.GetConfig(ctx, key)
		return innerErr
	})
	return result, err
}

func (t *instrumentedTransaction) SetConfig(ctx context.Context, key string, value interface{}) error {
	return t.backend.instrument(ctx, "tx_set_config", func(ctx context.Context) error {
		return t.Transaction.SetConfig(ctx, key, value)
	})
}

func (t *instrumentedTransaction) DeleteConfig(ctx context.Context, key string) error {
	return t.backend.instrument(ctx, "tx_delete_config", func(ctx context.Context) error {
		return t.Transaction.DeleteConfig(ctx, key)
	})
}

func (t *instrumentedTransaction) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := t.backend.instrument(ctx, "tx_get_credential", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = t.Transaction.GetCredential(ctx, id)
		return innerErr
	})
	return result, err
}

func (t *instrumentedTransaction) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	return t.backend.instrument(ctx, "tx_set_credential", func(ctx context.Context) error {
		return t.Transaction.SetCredential(ctx, id, data)
	})
}

func (t *instrumentedTransaction) DeleteCredential(ctx context.Context, id string) error {
	return t.backend.instrument(ctx, "tx_delete_credential", func(ctx context.Context) error {
		return t.Transaction.DeleteCredential(ctx, id)
	})
}

func (t *instrumentedTransaction) Commit(ctx context.Context) error {
	return t.backend.instrument(ctx, "tx_commit", func(ctx context.Context) error {
		return t.Transaction.Commit(ctx)
	})
}

func (t *instrumentedTransaction) Rollback(ctx context.Context) error {
	return t.backend.instrument(ctx, "tx_rollback", func(ctx context.Context) error {
		return t.Transaction.Rollback(ctx)
	})
}
