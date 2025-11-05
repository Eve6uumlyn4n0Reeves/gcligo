package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"gcli2api-go/internal/monitoring"
)

// mockBackend implements Backend interface for testing
type mockBackend struct {
	getConfigFunc        func(ctx context.Context, key string) (interface{}, error)
	setConfigFunc        func(ctx context.Context, key string, value interface{}) error
	deleteConfigFunc     func(ctx context.Context, key string) error
	listConfigsFunc      func(ctx context.Context) (map[string]interface{}, error)
	getCredentialFunc    func(ctx context.Context, id string) (map[string]interface{}, error)
	setCredentialFunc    func(ctx context.Context, id string, data map[string]interface{}) error
	deleteCredentialFunc func(ctx context.Context, id string) error
	listCredentialsFunc  func(ctx context.Context) ([]string, error)
	healthFunc           func(ctx context.Context) error
	closeFunc            func() error
	initializeFunc       func(ctx context.Context) error
}

func (m *mockBackend) Initialize(ctx context.Context) error {
	if m.initializeFunc != nil {
		return m.initializeFunc(ctx)
	}
	return nil
}

func (m *mockBackend) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockBackend) Health(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}

func (m *mockBackend) GetConfig(ctx context.Context, key string) (interface{}, error) {
	if m.getConfigFunc != nil {
		return m.getConfigFunc(ctx, key)
	}
	return nil, &ErrNotFound{Key: key}
}

func (m *mockBackend) SetConfig(ctx context.Context, key string, value interface{}) error {
	if m.setConfigFunc != nil {
		return m.setConfigFunc(ctx, key, value)
	}
	return nil
}

func (m *mockBackend) DeleteConfig(ctx context.Context, key string) error {
	if m.deleteConfigFunc != nil {
		return m.deleteConfigFunc(ctx, key)
	}
	return nil
}

func (m *mockBackend) ListConfigs(ctx context.Context) (map[string]interface{}, error) {
	if m.listConfigsFunc != nil {
		return m.listConfigsFunc(ctx)
	}
	return make(map[string]interface{}), nil
}

func (m *mockBackend) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	if m.getCredentialFunc != nil {
		return m.getCredentialFunc(ctx, id)
	}
	return nil, &ErrNotFound{Key: id}
}

func (m *mockBackend) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	if m.setCredentialFunc != nil {
		return m.setCredentialFunc(ctx, id, data)
	}
	return nil
}

func (m *mockBackend) DeleteCredential(ctx context.Context, id string) error {
	if m.deleteCredentialFunc != nil {
		return m.deleteCredentialFunc(ctx, id)
	}
	return nil
}

func (m *mockBackend) ListCredentials(ctx context.Context) ([]string, error) {
	if m.listCredentialsFunc != nil {
		return m.listCredentialsFunc(ctx)
	}
	return []string{}, nil
}

func (m *mockBackend) IncrementUsage(ctx context.Context, key string, field string, delta int64) error {
	return nil
}

func (m *mockBackend) GetUsage(ctx context.Context, key string) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

func (m *mockBackend) ResetUsage(ctx context.Context, key string) error {
	return nil
}

func (m *mockBackend) ListUsage(ctx context.Context) (map[string]map[string]interface{}, error) {
	return make(map[string]map[string]interface{}), nil
}

func (m *mockBackend) GetCache(ctx context.Context, key string) ([]byte, error) {
	return nil, &ErrNotSupported{Operation: "cache"}
}

func (m *mockBackend) SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return &ErrNotSupported{Operation: "cache"}
}

func (m *mockBackend) DeleteCache(ctx context.Context, key string) error {
	return &ErrNotSupported{Operation: "cache"}
}

func (m *mockBackend) BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	return make(map[string]map[string]interface{}), nil
}

func (m *mockBackend) BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error {
	return nil
}

func (m *mockBackend) BatchDeleteCredentials(ctx context.Context, ids []string) error {
	return nil
}

func (m *mockBackend) BeginTransaction(ctx context.Context) (Transaction, error) {
	return nil, &ErrNotSupported{Operation: "transaction"}
}

func (m *mockBackend) ExportData(ctx context.Context) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

func (m *mockBackend) ImportData(ctx context.Context, data map[string]interface{}) error {
	return nil
}

func (m *mockBackend) GetStorageStats(ctx context.Context) (StorageStats, error) {
	return StorageStats{}, nil
}

func TestWithInstrumentation(t *testing.T) {
	_ = context.Background() // For future use

	t.Run("Nil backend returns nil", func(t *testing.T) {
		result := WithInstrumentation(nil, nil, "test")
		if result != nil {
			t.Error("Expected nil for nil backend")
		}
	})

	t.Run("Nil metrics returns original backend", func(t *testing.T) {
		mock := &mockBackend{}
		result := WithInstrumentation(mock, nil, "test")
		if result != mock {
			t.Error("Expected original backend when metrics is nil")
		}
	})

	t.Run("Empty label defaults to unknown", func(t *testing.T) {
		mock := &mockBackend{}
		metrics := monitoring.NewEnhancedMetrics()
		result := WithInstrumentation(mock, metrics, "")

		instrumented, ok := result.(*instrumentedBackend)
		if !ok {
			t.Fatal("Expected instrumentedBackend")
		}

		if instrumented.label != "unknown" {
			t.Errorf("Expected label 'unknown', got %q", instrumented.label)
		}
	})

	t.Run("Wraps backend with instrumentation", func(t *testing.T) {
		mock := &mockBackend{}
		metrics := monitoring.NewEnhancedMetrics()
		result := WithInstrumentation(mock, metrics, "test")

		instrumented, ok := result.(*instrumentedBackend)
		if !ok {
			t.Fatal("Expected instrumentedBackend")
		}

		if instrumented.Backend != mock {
			t.Error("Expected wrapped backend to be original mock")
		}

		if instrumented.label != "test" {
			t.Errorf("Expected label 'test', got %q", instrumented.label)
		}
	})
}

func TestInstrumentedBackend_Operations(t *testing.T) {
	ctx := context.Background()

	t.Run("GetConfig success", func(t *testing.T) {
		mock := &mockBackend{
			getConfigFunc: func(ctx context.Context, key string) (interface{}, error) {
				return "value", nil
			},
		}
		metrics := monitoring.NewEnhancedMetrics()
		instrumented := WithInstrumentation(mock, metrics, "test")

		result, err := instrumented.GetConfig(ctx, "key")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != "value" {
			t.Errorf("Expected 'value', got %v", result)
		}
	})

	t.Run("GetConfig error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		mock := &mockBackend{
			getConfigFunc: func(ctx context.Context, key string) (interface{}, error) {
				return nil, expectedErr
			},
		}
		metrics := monitoring.NewEnhancedMetrics()
		instrumented := WithInstrumentation(mock, metrics, "test")

		_, err := instrumented.GetConfig(ctx, "key")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("SetCredential success", func(t *testing.T) {
		called := false
		mock := &mockBackend{
			setCredentialFunc: func(ctx context.Context, id string, data map[string]interface{}) error {
				called = true
				return nil
			},
		}
		metrics := monitoring.NewEnhancedMetrics()
		instrumented := WithInstrumentation(mock, metrics, "test")

		err := instrumented.SetCredential(ctx, "id", map[string]interface{}{"key": "value"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !called {
			t.Error("Expected SetCredential to be called")
		}
	})

	t.Run("ListCredentials success", func(t *testing.T) {
		mock := &mockBackend{
			listCredentialsFunc: func(ctx context.Context) ([]string, error) {
				return []string{"id1", "id2"}, nil
			},
		}
		metrics := monitoring.NewEnhancedMetrics()
		instrumented := WithInstrumentation(mock, metrics, "test")

		result, err := instrumented.ListCredentials(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 credentials, got %d", len(result))
		}
	})
}
