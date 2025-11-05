package storage

import (
	"context"
	"testing"

	storagecommon "gcli2api-go/internal/storage/common"
)

func TestPostgresBatchGetCredentialsEmpty(t *testing.T) {
	t.Parallel()
	p := &PostgresBackend{
		storage: nil,
		adapter: storagecommon.NewBackendAdapter(),
	}
	out, err := p.BatchGetCredentials(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty result, got %v", out)
	}
}

func TestPostgresBatchSetCredentialsInvalidData(t *testing.T) {
	t.Parallel()
	p := &PostgresBackend{
		adapter: storagecommon.NewBackendAdapter(),
	}
	err := p.BatchSetCredentials(context.Background(), map[string]map[string]interface{}{
		"cred": {
			"client_id":     "id",
			"client_secret": make(chan int),
			"refresh_token": "ref",
			"token_uri":     "uri",
			"project_id":    "proj",
		},
	})
	if err == nil {
		t.Fatalf("expected error for invalid payload")
	}
}
