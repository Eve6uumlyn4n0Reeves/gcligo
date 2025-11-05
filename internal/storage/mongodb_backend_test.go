package storage

import (
	"context"
	"testing"

	storagecommon "gcli2api-go/internal/storage/common"
)

func TestMongoBatchGetCredentialsEmpty(t *testing.T) {
	t.Parallel()
	m := &MongoDBBackend{
		storage: nil,
		adapter: storagecommon.NewBackendAdapter(),
	}
	out, err := m.BatchGetCredentials(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty map, got %v", out)
	}
}

func TestMongoBatchSetCredentialsInvalidPayload(t *testing.T) {
	t.Parallel()
	m := &MongoDBBackend{
		adapter: storagecommon.NewBackendAdapter(),
	}
	err := m.BatchSetCredentials(context.Background(), map[string]map[string]interface{}{
		"cred": {
			"client_id": make(chan int),
		},
	})
	if err == nil {
		t.Fatalf("expected error for invalid payload")
	}
}
