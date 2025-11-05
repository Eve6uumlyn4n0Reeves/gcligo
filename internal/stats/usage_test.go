package stats

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	store "gcli2api-go/internal/storage"
	"gcli2api-go/internal/utils"
)

func TestUsageStatsRecordAndGet(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	backend := store.NewFileBackend(tmp)
	require.NoError(t, backend.Initialize(ctx))

	us := NewUsageStats(backend, time.Minute, "UTC", 0)

	err := us.RecordRequest(ctx, "key-a", "gemini-2.5-pro", true, 10, 5)
	require.NoError(t, err)

	err = us.RecordRequest(ctx, "key-a", "gemini-2.5-pro", false, 3, 2)
	require.NoError(t, err)

	record, err := us.GetUsage(ctx, "key-a")
	require.NoError(t, err)
	require.Equal(t, int64(2), record.TotalRequests)
	require.Equal(t, int64(1), record.SuccessRequests)
	require.Equal(t, int64(1), record.FailedRequests)
	require.Equal(t, int64(20), record.TotalTokens)
	require.Equal(t, int64(13), record.PromptTokens)
	require.Equal(t, int64(7), record.CompletionTokens)
	require.InDelta(t, 50.0, record.GetSuccessRate(), 0.01)
	require.InDelta(t, 50.0, record.GetFailureRate(), 0.01)

	total, err := us.GetUsage(ctx, aggregateTotalKey)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total.TotalRequests)

	modelUsage, err := us.GetUsage(ctx, aggregateModelPrefix+"gemini-2.5-pro")
	require.NoError(t, err)
	assert.Equal(t, int64(2), modelUsage.TotalRequests)
}

func TestUsageStatsResetUsage(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	backend := store.NewFileBackend(tmp)
	require.NoError(t, backend.Initialize(ctx))

	us := NewUsageStats(backend, time.Minute, "UTC", 0)
	require.NoError(t, us.RecordRequest(ctx, "key-reset", "gemini-2.5-pro", true, 1, 1))

	require.NoError(t, us.ResetUsage(ctx, "key-reset"))

	_, err := us.GetUsage(ctx, "key-reset")
	require.Error(t, err)
}

func TestUsageStatsGetAllAndResetAll(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	backend := store.NewFileBackend(tmp)
	require.NoError(t, backend.Initialize(ctx))

	us := NewUsageStats(backend, time.Hour, "UTC", 0)
	require.NoError(t, us.RecordRequest(ctx, "key-1", "gemini-2.5-pro", true, 2, 2))
	require.NoError(t, us.RecordRequest(ctx, "key-2", "gemini-2.5-flash", false, 0, 1))

	all, err := us.GetAllUsage(ctx)
	require.NoError(t, err)
	require.Contains(t, all, "key-1")
	require.Contains(t, all, "key-2")
	require.Contains(t, all, aggregateTotalKey)
	require.Contains(t, all, aggregateModelPrefix+"gemini-2.5-pro")

	require.NoError(t, us.ResetAll(ctx))
	require.True(t, us.resetSchedule.After(time.Now().UTC()))

	_, err = us.GetUsage(ctx, "key-1")
	require.Error(t, err)
	_, err = us.GetUsage(ctx, "key-2")
	require.Error(t, err)
	_, err = us.GetUsage(ctx, aggregateTotalKey)
	require.Error(t, err)
}

func TestUsageStatsNoBackend(t *testing.T) {
	ctx := context.Background()
	us := NewUsageStats(nil, time.Minute, "UTC", 0)

	err := us.RecordRequest(ctx, "noop", "", true, 0, 0)
	require.Error(t, err)

	_, err = us.GetUsage(ctx, "noop")
	require.Error(t, err)

	err = us.ResetUsage(ctx, "noop")
	require.Error(t, err)

	_, err = us.GetAllUsage(ctx)
	require.Error(t, err)
}

func TestUsageStatsDailyResetTimezone(t *testing.T) {
	loc, err := utils.ParseLocation("UTC+7")
	require.NoError(t, err)

	us := NewUsageStats(nil, 24*time.Hour, "UTC+7", 0)
	require.NotNil(t, us)
	require.NotNil(t, us.resetLocation)
	require.Equal(t, loc.String(), us.resetLocation.String())

	nextLocal := us.resetSchedule.In(us.resetLocation)
	require.Equal(t, 0, nextLocal.Hour())
	require.Equal(t, 0, nextLocal.Minute())
}

func TestUsageStatsModelAggregateUsesBase(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	backend := store.NewFileBackend(tmp)
	require.NoError(t, backend.Initialize(ctx))

	us := NewUsageStats(backend, time.Minute, "UTC", 0)
	require.NoError(t, us.RecordRequest(ctx, "k", "流式抗截断/gemini-2.5-pro-maxthinking", true, 0, 0))

	rec, err := us.GetUsage(ctx, aggregateModelPrefix+"gemini-2.5-pro")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rec.TotalRequests)
}
