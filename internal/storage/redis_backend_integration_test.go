package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRedisBackend_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("redis integration test skipped in short mode")
	}

	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7.2-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("redis container unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "6379/tcp")
	require.NoError(t, err)
	addr := fmt.Sprintf("%s:%s", host, port.Port())

	backend, err := NewRedisBackend(addr, "", 0, "it:")
	require.NoError(t, err)

	require.NoError(t, backend.Initialize(ctx))
	t.Cleanup(func() {
		_ = backend.Close()
	})

	t.Run("config CRUD", func(t *testing.T) {
		require.NoError(t, backend.SetConfig(ctx, "cfg:test", map[string]any{"flag": true}))

		val, err := backend.GetConfig(ctx, "cfg:test")
		require.NoError(t, err)
		m, ok := val.(map[string]any)
		require.True(t, ok)
		require.Equal(t, true, m["flag"])

		configs, err := backend.ListConfigs(ctx)
		require.NoError(t, err)
		require.Contains(t, configs, "cfg:test")

		require.NoError(t, backend.DeleteConfig(ctx, "cfg:test"))
		_, err = backend.GetConfig(ctx, "cfg:test")
		require.Error(t, err)
	})

	t.Run("credential CRUD", func(t *testing.T) {
		payload := map[string]any{"token": "redis-secret"}
		require.NoError(t, backend.SetCredential(ctx, "cred-1", payload))

		got, err := backend.GetCredential(ctx, "cred-1")
		require.NoError(t, err)
		require.Equal(t, "redis-secret", got["token"])

		all, err := backend.ListCredentials(ctx)
		require.NoError(t, err)
		require.Contains(t, all, "cred-1")

		require.NoError(t, backend.DeleteCredential(ctx, "cred-1"))
		_, err = backend.GetCredential(ctx, "cred-1")
		require.Error(t, err)
	})
}
