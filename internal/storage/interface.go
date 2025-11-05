package storage

import (
	"context"
	"time"
)

// Backend defines the interface for storage implementations
type Backend interface {
	// Initialize sets up the storage backend
	Initialize(ctx context.Context) error

	// Close closes the storage backend
	Close() error

	// Health checks if the storage backend is healthy
	Health(ctx context.Context) error

	// Credential operations
	GetCredential(ctx context.Context, id string) (map[string]interface{}, error)
	SetCredential(ctx context.Context, id string, data map[string]interface{}) error
	DeleteCredential(ctx context.Context, id string) error
	ListCredentials(ctx context.Context) ([]string, error)

	// Config operations
	GetConfig(ctx context.Context, key string) (interface{}, error)
	SetConfig(ctx context.Context, key string, value interface{}) error
	DeleteConfig(ctx context.Context, key string) error
	ListConfigs(ctx context.Context) (map[string]interface{}, error)

	// Usage stats operations
	IncrementUsage(ctx context.Context, key string, field string, delta int64) error
	GetUsage(ctx context.Context, key string) (map[string]interface{}, error)
	ResetUsage(ctx context.Context, key string) error
	ListUsage(ctx context.Context) (map[string]map[string]interface{}, error)

	// Cache operations (optional, can return ErrNotSupported)
	GetCache(ctx context.Context, key string) ([]byte, error)
	SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error
	DeleteCache(ctx context.Context, key string) error

	// Batch operations for performance
	BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error)
	BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error
	BatchDeleteCredentials(ctx context.Context, ids []string) error

	// Transaction support (optional, can return ErrNotSupported)
	BeginTransaction(ctx context.Context) (Transaction, error)

	// Backup and migration support
	ExportData(ctx context.Context) (map[string]interface{}, error)
	ImportData(ctx context.Context, data map[string]interface{}) error

	// Storage metrics and monitoring
	GetStorageStats(ctx context.Context) (StorageStats, error)
}

// ErrNotFound is returned when a key is not found
type ErrNotFound struct {
	Key string
}

func (e *ErrNotFound) Error() string {
	return "key not found: " + e.Key
}

// ErrNotSupported is returned when an operation is not supported
type ErrNotSupported struct {
	Operation string
}

func (e *ErrNotSupported) Error() string {
	return "operation not supported: " + e.Operation
}

// Transaction interface for atomic operations
type Transaction interface {
	// Credential operations within transaction
	GetCredential(ctx context.Context, id string) (map[string]interface{}, error)
	SetCredential(ctx context.Context, id string, data map[string]interface{}) error
	DeleteCredential(ctx context.Context, id string) error

	// Config operations within transaction
	GetConfig(ctx context.Context, key string) (interface{}, error)
	SetConfig(ctx context.Context, key string, value interface{}) error
	DeleteConfig(ctx context.Context, key string) error

	// Transaction control
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// StorageStats provides storage backend statistics
type StorageStats struct {
	Backend          string                 `json:"backend"`
	Healthy          bool                   `json:"healthy"`
	ConnectionCount  int                    `json:"connection_count,omitempty"`
	CredentialCount  int                    `json:"credential_count"`
	ConfigCount      int                    `json:"config_count"`
	UsageRecordCount int                    `json:"usage_record_count"`
	TotalSize        int64                  `json:"total_size_bytes,omitempty"`
	LastBackup       *time.Time             `json:"last_backup,omitempty"`
	Performance      *PerformanceStats      `json:"performance,omitempty"`
	Details          map[string]interface{} `json:"details,omitempty"`
}

// PerformanceStats tracks storage performance metrics
type PerformanceStats struct {
	AverageReadLatency  time.Duration `json:"average_read_latency"`
	AverageWriteLatency time.Duration `json:"average_write_latency"`
	OperationsPerSecond float64       `json:"operations_per_second"`
	ErrorRate           float64       `json:"error_rate"`
	LastMeasurement     time.Time     `json:"last_measurement"`
}

// ConfigMutation represents a single configuration change in a batch apply operation.
type ConfigMutation struct {
	Key    string
	Value  interface{}
	Delete bool
}

// BatchApplyOptions controls the behavior of ApplyConfigBatch implementations.
type BatchApplyOptions struct {
	// IdempotencyKey provides replay protection; identical keys must yield idempotent behavior.
	IdempotencyKey string
	// TTL determines how long coordinator metadata should be retained for deduplication.
	TTL time.Duration
	// Stage is an optional label (e.g. "apply", "rollback") for observability.
	Stage string
}

// ConfigBatchApplier is implemented by backends that support two-phase, idempotent config mutations.
type ConfigBatchApplier interface {
	ApplyConfigBatch(ctx context.Context, mutations []ConfigMutation, opts BatchApplyOptions) error
}

// PlanAuditEntry represents a recorded plan apply attempt.
type PlanAuditEntry struct {
	Backend       string     `json:"backend"`
	Key           string     `json:"key"`
	Stage         string     `json:"stage,omitempty"`
	Status        string     `json:"status"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CommittedAt   *time.Time `json:"committed_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty"`
	DurationMS    int64      `json:"duration_ms,omitempty"`
	MutationCount int        `json:"mutation_count,omitempty"`
	PayloadHash   string     `json:"payload_hash,omitempty"`
	Error         string     `json:"error,omitempty"`
	Source        string     `json:"source"`
	RecordedAt    time.Time  `json:"recorded_at"`
}

// PlanAuditExporter may expose plan apply audit entries.
type PlanAuditExporter interface {
	ExportPlanAudit(ctx context.Context) ([]PlanAuditEntry, error)
}
