package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgresBackend_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("postgres integration test skipped in short mode")
	}

	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "itdb",
				"POSTGRES_USER":     "ituser",
				"POSTGRES_PASSWORD": "itpass",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("postgres container unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://ituser:itpass@%s:%s/itdb?sslmode=disable", host, port.Port())
	backend, err := NewPostgresBackend(dsn)
	require.NoError(t, err)

	require.NoError(t, backend.Initialize(ctx))
	t.Cleanup(func() {
		_ = backend.Close()
	})

	t.Run("config CRUD", func(t *testing.T) {
		require.NoError(t, backend.SetConfig(ctx, "cfg:test", map[string]any{"threshold": 10}))

		val, err := backend.GetConfig(ctx, "cfg:test")
		require.NoError(t, err)
		m, ok := val.(map[string]any)
		require.True(t, ok)
		require.EqualValues(t, 10, m["threshold"])

		configs, err := backend.ListConfigs(ctx)
		require.NoError(t, err)
		require.Contains(t, configs, "cfg:test")
	})

	t.Run("credential CRUD", func(t *testing.T) {
		payload := map[string]any{"access_token": "pg-secret"}
		require.NoError(t, backend.SetCredential(ctx, "cred-1", payload))

		got, err := backend.GetCredential(ctx, "cred-1")
		require.NoError(t, err)
		require.Equal(t, "pg-secret", got["access_token"])

		require.NoError(t, backend.DeleteCredential(ctx, "cred-1"))
		_, err = backend.GetCredential(ctx, "cred-1")
		require.Error(t, err)
	})
}
