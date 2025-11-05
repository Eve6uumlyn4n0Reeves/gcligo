package storage

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestRedisBackendBeginTransactionUnsupported(t *testing.T) {
	t.Parallel()
	var backend RedisBackend
	if _, err := backend.BeginTransaction(context.Background()); err == nil {
		t.Fatalf("expected ErrNotSupported")
	}
}

func TestRedisBackendBatchOperationsRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mr, err := miniredis.Run()
	if err != nil {
		t.Skipf("miniredis unavailable: %v", err)
	}
	t.Cleanup(mr.Close)

	rb, err := NewRedisBackend(mr.Addr(), "", 0, "gcli2api:")
	require.NoError(t, err)
	require.NoError(t, rb.Initialize(ctx))
	t.Cleanup(func() { _ = rb.Close() })

	payload := map[string]map[string]interface{}{
		"cred-1": {
			"client_id":     "id-1",
			"client_secret": "secret-1",
			"refresh_token": "ref-1",
			"token_uri":     "uri-1",
			"project_id":    "proj-1",
		},
		"cred-2": {
			"client_id":     "id-2",
			"client_secret": "secret-2",
			"refresh_token": "ref-2",
			"token_uri":     "uri-2",
			"project_id":    "proj-2",
		},
	}
	require.NoError(t, rb.BatchSetCredentials(ctx, payload))

	got, err := rb.BatchGetCredentials(ctx, []string{"cred-1", "cred-2", "missing"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "id-1", got["cred-1"]["client_id"])
	require.Equal(t, "proj-2", got["cred-2"]["project_id"])

	require.NoError(t, rb.BatchDeleteCredentials(ctx, []string{"cred-1"}))

	remaining, err := rb.BatchGetCredentials(ctx, []string{"cred-1", "cred-2"})
	require.NoError(t, err)
	_, exists := remaining["cred-1"]
	require.False(t, exists)
	require.Equal(t, "id-2", remaining["cred-2"]["client_id"])
}

func TestRedisBackendBatchSetCredentialsInvalidPayload(t *testing.T) {
	t.Parallel()
	rb := &RedisBackend{}
	err := rb.BatchSetCredentials(context.Background(), map[string]map[string]interface{}{
		"cred-bad": {
			"client_id": make(chan int),
		},
	})
	require.Error(t, err)
}
