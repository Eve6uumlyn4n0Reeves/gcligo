package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
)

func withMongoBackend(t *testing.T, dbName string) (*MongoDBBackend, func()) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "mongo:6.0",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForLog("Waiting for connections").WithStartupTimeout(30 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "27017")
	require.NoError(t, err)
	uri := fmt.Sprintf("mongodb://%s:%s", host, port.Port())

	backend, err := NewMongoDBBackend(uri, dbName)
	require.NoError(t, err)
	require.NoError(t, backend.Initialize(ctx))

	cleanup := func() {
		_ = backend.Close()
		require.NoError(t, container.Terminate(ctx))
	}
	return backend, cleanup
}

func TestMongoApplyConfigBatchSuccess(t *testing.T) {
	requireDocker(t)

	t.Parallel()

	backend, cleanup := withMongoBackend(t, "audit_success")
	defer cleanup()

	ctx := context.Background()
	mutations := []ConfigMutation{
		{Key: "model_variant_config", Value: map[string]any{"auto": true}},
	}

	opts := BatchApplyOptions{
		IdempotencyKey: "plan-success",
		TTL:            time.Minute,
		Stage:          "apply",
	}

	require.NoError(t, backend.ApplyConfigBatch(ctx, mutations, opts))

	cfg, err := backend.GetConfig(ctx, "model_variant_config")
	require.NoError(t, err)
	require.Equal(t, map[string]any{"auto": true}, cfg)

	lockColl := backend.storage.PlanLocksCollection()
	var lockDoc bson.M
	require.NoError(t, lockColl.FindOne(ctx, bson.M{"_id": "plan-success"}).Decode(&lockDoc))
	require.Equal(t, "committed", lockDoc["status"])
	require.Equal(t, "apply", lockDoc["stage"])

	commitColl := backend.storage.PlanCommitCollection()
	cursor, err := commitColl.Find(ctx, bson.M{"key": "plan-success"})
	require.NoError(t, err)
	defer cursor.Close(ctx)
	require.True(t, cursor.Next(ctx))

	var commitDoc bson.M
	require.NoError(t, cursor.Decode(&commitDoc))
	require.Equal(t, "committed", commitDoc["status"])
	require.Equal(t, "apply", commitDoc["stage"])
	require.EqualValues(t, 1, commitDoc["mutation_count"])
}

func TestMongoApplyConfigBatchFailure(t *testing.T) {
	requireDocker(t)

	t.Parallel()

	backend, cleanup := withMongoBackend(t, "audit_failure")
	defer cleanup()

	ctx := context.Background()
	err := backend.ApplyConfigBatch(ctx, []ConfigMutation{
		{Key: "model_variant_config", Value: make(chan int)},
	}, BatchApplyOptions{
		IdempotencyKey: "plan-failure",
		TTL:            time.Minute,
		Stage:          "apply",
	})
	require.Error(t, err)

	lockColl := backend.storage.PlanLocksCollection()
	var lockDoc bson.M
	require.NoError(t, lockColl.FindOne(ctx, bson.M{"_id": "plan-failure"}).Decode(&lockDoc))
	require.Equal(t, "failed", lockDoc["status"])
	require.Equal(t, "apply", lockDoc["stage"])
	require.NotEmpty(t, lockDoc["error"])

	commitColl := backend.storage.PlanCommitCollection()
	var commitDoc bson.M
	require.NoError(t, commitColl.FindOne(ctx, bson.M{"key": "plan-failure"}).Decode(&commitDoc))
	require.Equal(t, "failed", commitDoc["status"])
	require.Equal(t, "apply", commitDoc["stage"])
	require.NotEmpty(t, commitDoc["error"])
}

func requireDocker(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	cli, err := testcontainers.NewDockerClientWithOpts(ctx)
	if err != nil {
		t.Skipf("docker not available: %v", err)
		return
	}
	if _, err := cli.Ping(ctx); err != nil {
		t.Skipf("docker daemon unreachable: %v", err)
		_ = cli.Close()
		return
	}
	_ = cli.Close()
}
