package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"gcli2api-go/internal/monitoring"
	store "gcli2api-go/internal/storage"
)

// simple in-memory backend with optional tx support
type memBackend struct {
	cfg       map[string]interface{}
	tx        bool
	commitErr error
}

func newMem(tx bool) *memBackend                           { return &memBackend{cfg: map[string]interface{}{}, tx: tx} }
func (m *memBackend) Initialize(ctx context.Context) error { return nil }
func (m *memBackend) Close() error                         { return nil }
func (m *memBackend) Health(ctx context.Context) error     { return nil }
func (m *memBackend) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	return nil, &store.ErrNotSupported{Operation: "cred"}
}
func (m *memBackend) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	return &store.ErrNotSupported{Operation: "cred"}
}
func (m *memBackend) DeleteCredential(ctx context.Context, id string) error {
	return &store.ErrNotSupported{Operation: "cred"}
}
func (m *memBackend) ListCredentials(ctx context.Context) ([]string, error) {
	return nil, &store.ErrNotSupported{Operation: "cred"}
}
func (m *memBackend) GetConfig(ctx context.Context, key string) (interface{}, error) {
	v, ok := m.cfg[key]
	if !ok {
		return nil, &store.ErrNotFound{Key: key}
	}
	return v, nil
}
func (m *memBackend) SetConfig(ctx context.Context, key string, value interface{}) error {
	m.cfg[key] = value
	return nil
}
func (m *memBackend) DeleteConfig(ctx context.Context, key string) error {
	delete(m.cfg, key)
	return nil
}
func (m *memBackend) ListConfigs(ctx context.Context) (map[string]interface{}, error) {
	return m.cfg, nil
}
func (m *memBackend) IncrementUsage(ctx context.Context, key string, field string, delta int64) error {
	return &store.ErrNotSupported{Operation: "usage"}
}
func (m *memBackend) GetUsage(ctx context.Context, key string) (map[string]interface{}, error) {
	return nil, &store.ErrNotSupported{Operation: "usage"}
}
func (m *memBackend) ResetUsage(ctx context.Context, key string) error {
	return &store.ErrNotSupported{Operation: "usage"}
}
func (m *memBackend) ListUsage(ctx context.Context) (map[string]map[string]interface{}, error) {
	return nil, &store.ErrNotSupported{Operation: "usage"}
}
func (m *memBackend) GetCache(ctx context.Context, key string) ([]byte, error) {
	return nil, &store.ErrNotSupported{Operation: "cache"}
}
func (m *memBackend) SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return &store.ErrNotSupported{Operation: "cache"}
}
func (m *memBackend) DeleteCache(ctx context.Context, key string) error {
	return &store.ErrNotSupported{Operation: "cache"}
}
func (m *memBackend) BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	return nil, &store.ErrNotSupported{Operation: "batch"}
}
func (m *memBackend) BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error {
	return &store.ErrNotSupported{Operation: "batch"}
}
func (m *memBackend) BatchDeleteCredentials(ctx context.Context, ids []string) error {
	return &store.ErrNotSupported{Operation: "batch"}
}
func (m *memBackend) BeginTransaction(ctx context.Context) (store.Transaction, error) {
	if !m.tx {
		return nil, &store.ErrNotSupported{Operation: "tx"}
	}
	return &memTx{m: m, pending: map[string]interface{}{}, commitErr: m.commitErr}, nil
}
func (m *memBackend) ExportData(ctx context.Context) (map[string]interface{}, error)    { return nil, nil }
func (m *memBackend) ImportData(ctx context.Context, data map[string]interface{}) error { return nil }
func (m *memBackend) GetStorageStats(ctx context.Context) (store.StorageStats, error) {
	return store.StorageStats{Backend: "mem"}, nil
}

type memTx struct {
	m         *memBackend
	pending   map[string]interface{}
	commitErr error
}

func (t *memTx) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	return nil, &store.ErrNotSupported{Operation: "cred"}
}
func (t *memTx) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	return &store.ErrNotSupported{Operation: "cred"}
}
func (t *memTx) DeleteCredential(ctx context.Context, id string) error {
	return &store.ErrNotSupported{Operation: "cred"}
}
func (t *memTx) GetConfig(ctx context.Context, key string) (interface{}, error) {
	if v, ok := t.pending[key]; ok {
		return v, nil
	}
	return t.m.GetConfig(ctx, key)
}
func (t *memTx) SetConfig(ctx context.Context, key string, value interface{}) error {
	t.pending[key] = value
	return nil
}
func (t *memTx) DeleteConfig(ctx context.Context, key string) error { t.pending[key] = nil; return nil }
func (t *memTx) Commit(ctx context.Context) error {
	if t.commitErr != nil {
		return t.commitErr
	}
	for k, v := range t.pending {
		if v == nil {
			delete(t.m.cfg, k)
		} else {
			t.m.cfg[k] = v
		}
	}
	return nil
}
func (t *memTx) Rollback(ctx context.Context) error { t.pending = map[string]interface{}{}; return nil }

func TestAssemblyService_Apply_NoTx(t *testing.T) {
	s := NewAssemblyService(nil, newMem(false), monitoring.NewEnhancedMetrics(), nil)
	plan := map[string]any{"models": map[string]any{"openai": []any{map[string]any{"id": "gemini-2.5-pro"}}}}
	_ = s.st.SetConfig(context.Background(), "assembly_plan:it", plan)
	if err := s.ApplyPlan(context.Background(), "it"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := s.st.GetConfig(context.Background(), "model_registry_openai"); err != nil {
		t.Fatalf("expect set: %v", err)
	}
}

func TestAssemblyService_Apply_WithTx(t *testing.T) {
	metrics := monitoring.NewEnhancedMetrics()
	s := NewAssemblyService(nil, newMem(true), metrics, nil)
	plan := map[string]any{"models": map[string]any{"openai": []any{map[string]any{"id": "gemini-2.5-pro"}}}}
	_ = s.st.SetConfig(context.Background(), "assembly_plan:it", plan)
	if err := s.ApplyPlan(context.Background(), "it"); err != nil {
		t.Fatalf("apply tx: %v", err)
	}
	if _, err := s.st.GetConfig(context.Background(), "model_registry_openai"); err != nil {
		t.Fatalf("expect set: %v", err)
	}

	snap := metrics.GetSnapshot()
	txAny, ok := snap["transactions"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected transactions in snapshot")
	}
	commits, ok := txAny["commits"].(map[string]int64)
	if !ok {
		t.Fatalf("expected commits map")
	}
	attempts, ok := txAny["attempts"].(map[string]int64)
	if !ok {
		t.Fatalf("expected attempts map")
	}
	if attempts["unknown"] != 1 || commits["unknown"] != 1 {
		t.Fatalf("unexpected transaction metrics: attempts=%v commits=%v", attempts, commits)
	}
}

func TestAssemblyService_Apply_TxFailureMetrics(t *testing.T) {
	metrics := monitoring.NewEnhancedMetrics()
	backend := newMem(true)
	backend.commitErr = errors.New("boom")

	s := NewAssemblyService(nil, backend, metrics, nil)
	plan := map[string]any{"models": map[string]any{"openai": []any{map[string]any{"id": "gemini-2.5-pro"}}}}
	_ = s.st.SetConfig(context.Background(), "assembly_plan:it", plan)
	if err := s.ApplyPlan(context.Background(), "it"); err == nil {
		t.Fatalf("expected error when transaction commit fails")
	}

	snap := metrics.GetSnapshot()
	txAny := snap["transactions"].(map[string]interface{})
	attempts := txAny["attempts"].(map[string]int64)
	failures := txAny["failures"].(map[string]int64)
	if attempts["unknown"] != 1 || failures["unknown"] != 1 {
		t.Fatalf("unexpected transaction failure metrics: attempts=%v failures=%v", attempts, failures)
	}
}

func TestAssemblyService_Rollback_NoBackup(t *testing.T) {
	s := NewAssemblyService(nil, newMem(false), monitoring.NewEnhancedMetrics(), nil)
	if err := s.RollbackPlan(context.Background(), "none"); err == nil {
		t.Fatalf("expected error for missing backup")
	}
}
