package storage

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestRedisApplyConfigBatchSuccessAndIdempotency(t *testing.T) {
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

	mutations := []ConfigMutation{
		{Key: "model_variant_config", Value: map[string]any{"foo": "bar"}},
	}
	opts := BatchApplyOptions{
		IdempotencyKey: "plan-success",
		TTL:            45 * time.Second,
		Stage:          "apply",
	}

	require.NoError(t, rb.ApplyConfigBatch(ctx, mutations, opts))

	cfg, err := rb.GetConfig(ctx, "model_variant_config")
	require.NoError(t, err)
	stored, ok := cfg.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "bar", fmt.Sprint(stored["foo"]))

	meta, err := rb.client.HGetAll(ctx, "gcli2api:plan:meta:plan-success").Result()
	require.NoError(t, err)
	require.Equal(t, "committed", meta[idempotencyStatusKey])
	require.Equal(t, "apply", meta["last_stage"])
	require.NotEmpty(t, meta["last_committed_at"])
	require.NotEmpty(t, meta["last_payload_hash"])
	require.Equal(t, "", meta["last_error"])

	count, err := strconv.ParseInt(meta["last_mutation_count"], 10, 64)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	ttl := mr.TTL("gcli2api:plan:meta:plan-success")
	require.GreaterOrEqual(t, int(ttl/time.Second), 30)

	// Re-applying with the same key should stay idempotent.
	require.NoError(t, rb.ApplyConfigBatch(ctx, mutations, opts))
	meta, err = rb.client.HGetAll(ctx, "gcli2api:plan:meta:plan-success").Result()
	require.NoError(t, err)
	require.Equal(t, "committed", meta[idempotencyStatusKey])
}

func TestRedisApplyConfigBatchFailureRecordsMetadata(t *testing.T) {
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

	err = rb.ApplyConfigBatch(ctx, []ConfigMutation{
		{Key: "model_variant_config", Value: map[string]any{"bad": make(chan int)}},
	}, BatchApplyOptions{
		IdempotencyKey: "plan-failure",
		TTL:            30 * time.Second,
		Stage:          "apply",
	})
	require.Error(t, err)
	meta, err := rb.client.HGetAll(ctx, "gcli2api:plan:meta:plan-failure").Result()
	require.NoError(t, err)
	require.Equal(t, "failed", meta[idempotencyStatusKey])
	require.Equal(t, "apply", meta["last_stage"])
	require.NotEmpty(t, meta["last_error"])
	require.NotEmpty(t, meta["last_failed_at"])
}
