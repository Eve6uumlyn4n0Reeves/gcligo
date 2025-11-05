package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMongoDBBackend_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("mongodb integration test skipped in short mode")
	}

	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:7.0",
			ExposedPorts: []string{"27017/tcp"},
			WaitingFor:   wait.ForListeningPort("27017/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("mongodb container unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "27017/tcp")
	require.NoError(t, err)

	uri := fmt.Sprintf("mongodb://%s:%s", host, port.Port())
	backend, err := NewMongoDBBackend(uri, "it_tests")
	require.NoError(t, err)

	require.NoError(t, backend.Initialize(ctx))
	t.Cleanup(func() {
		_ = backend.Close()
	})

	t.Run("credential CRUD", func(t *testing.T) {
		payload := map[string]any{"token": "mongo-secret"}
		require.NoError(t, backend.SetCredential(ctx, "cred-1", payload))

		got, err := backend.GetCredential(ctx, "cred-1")
		require.NoError(t, err)
		require.Equal(t, "mongo-secret", got["token"])

		all, err := backend.ListCredentials(ctx)
		require.NoError(t, err)
		require.Contains(t, all, "cred-1")

		require.NoError(t, backend.DeleteCredential(ctx, "cred-1"))
		_, err = backend.GetCredential(ctx, "cred-1")
		require.Error(t, err)
	})

	t.Run("cache operations", func(t *testing.T) {
		require.NoError(t, backend.SetCache(ctx, "cache-key", []byte("value"), 0))
		data, err := backend.GetCache(ctx, "cache-key")
		require.NoError(t, err)
		require.Equal(t, []byte("value"), data)
		require.NoError(t, backend.DeleteCache(ctx, "cache-key"))
	})
}
