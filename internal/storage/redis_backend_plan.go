//go:build !stats_isolation

package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultPlanTTL       = 2 * time.Minute
	idempotencyStatusKey = "plan:status"
)

// ApplyConfigBatch implements ConfigBatchApplier for Redis backends using a best-effort two-phase protocol.
func (r *RedisBackend) ApplyConfigBatch(ctx context.Context, mutations []ConfigMutation, opts BatchApplyOptions) error {
	if r.client == nil {
		return errors.New("redis backend not initialized")
	}
	if len(mutations) == 0 {
		return nil
	}
	if opts.IdempotencyKey == "" {
		return fmt.Errorf("missing idempotency key")
	}
	stage := strings.TrimSpace(opts.Stage)
	if stage == "" {
		stage = "apply"
	}
	start := time.Now()
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = defaultPlanTTL
	}

	metaKey := r.prefix + "plan:meta:" + opts.IdempotencyKey
	status, err := r.client.HGet(ctx, metaKey, idempotencyStatusKey).Result()
	if err == nil && status == "committed" {
		return nil
	}
	if err == nil && status == "in_progress" {
		return fmt.Errorf("plan apply already in progress for key %s", opts.IdempotencyKey)
	}
	if err != nil && err != redis.Nil {
		return err
	}

	if ok, err := r.client.HSetNX(ctx, metaKey, idempotencyStatusKey, "in_progress").Result(); err != nil {
		return err
	} else if !ok {
		// Another writer beat us to initialization; re-check status.
		status, err = r.client.HGet(ctx, metaKey, idempotencyStatusKey).Result()
		if err == nil && status == "committed" {
			return nil
		}
		if err != nil && err != redis.Nil {
			return err
		}
		return fmt.Errorf("plan apply already started for key %s", opts.IdempotencyKey)
	}
	if err := r.client.Expire(ctx, metaKey, ttl).Err(); err != nil {
		r.recordPlanMetaFailure(ctx, metaKey, stage, ttl, err)
		return err
	}

	now := time.Now().UTC()
	_ = r.client.HSet(ctx, metaKey, map[string]interface{}{
		"last_stage":      stage,
		"last_started_at": now.Format(time.RFC3339Nano),
	}).Err()

	payloadHash := ""
	if payloadBytes, err := r.adapter.MarshalValue(mutations); err == nil {
		sum := sha256.Sum256(payloadBytes)
		payloadHash = hex.EncodeToString(sum[:])
	}

	var hmArgs []interface{}
	delFields := make([]string, 0)
	for _, mut := range mutations {
		if mut.Delete {
			delFields = append(delFields, mut.Key)
			continue
		}
		payload, err := r.adapter.MarshalValue(mut.Value)
		if err != nil {
			r.recordPlanMetaFailure(ctx, metaKey, stage, ttl, err)
			return err
		}
		hmArgs = append(hmArgs, mut.Key, payload)
	}

	_, err = r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		if len(hmArgs) > 0 {
			pipe.HSet(ctx, r.prefix+"config", hmArgs...)
		}
		if len(delFields) > 0 {
			pipe.HDel(ctx, r.prefix+"config", delFields...)
		}
		return nil
	})
	if err != nil {
		r.recordPlanMetaFailure(ctx, metaKey, stage, ttl, err)
		return err
	}

	r.recordPlanMetaSuccess(ctx, metaKey, stage, ttl, time.Since(start), len(mutations), payloadHash)
	return nil
}

func (r *RedisBackend) recordPlanMetaFailure(ctx context.Context, metaKey, stage string, ttl time.Duration, cause error) {
	if r.client == nil || metaKey == "" {
		return
	}
	fields := map[string]interface{}{
		idempotencyStatusKey: "failed",
		"last_stage":         stage,
		"last_error":         cause.Error(),
		"last_failed_at":     time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := r.client.HSet(ctx, metaKey, fields).Err(); err == nil && ttl > 0 {
		_ = r.client.Expire(ctx, metaKey, ttl).Err()
	}
}

func (r *RedisBackend) recordPlanMetaSuccess(ctx context.Context, metaKey, stage string, ttl time.Duration, duration time.Duration, mutationCount int, payloadHash string) {
	if r.client == nil || metaKey == "" {
		return
	}
	fields := map[string]interface{}{
		idempotencyStatusKey:  "committed",
		"last_stage":          stage,
		"last_error":          "",
		"last_committed_at":   time.Now().UTC().Format(time.RFC3339Nano),
		"last_duration_ms":    duration.Milliseconds(),
		"last_mutation_count": mutationCount,
	}
	if payloadHash != "" {
		fields["last_payload_hash"] = payloadHash
	}
	if err := r.client.HSet(ctx, metaKey, fields).Err(); err == nil && ttl > 0 {
		_ = r.client.Expire(ctx, metaKey, ttl).Err()
	}
}

// ExportPlanAudit collects plan metadata stored in Redis.
func (r *RedisBackend) ExportPlanAudit(ctx context.Context) ([]PlanAuditEntry, error) {
	if r.client == nil {
		return nil, errors.New("redis backend not initialized")
	}
	pattern := r.prefix + "plan:meta:*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}
	entries := make([]PlanAuditEntry, 0, len(keys))
	for _, fullKey := range keys {
		data, err := r.client.HGetAll(ctx, fullKey).Result()
		if err != nil {
			return nil, err
		}
		key := strings.TrimPrefix(fullKey, r.prefix+"plan:meta:")
		entry := PlanAuditEntry{
			Backend: "redis",
			Key:     key,
			Status:  data[idempotencyStatusKey],
			Stage:   data["last_stage"],
			Error:   data["last_error"],
			Source:  "redis:plan:meta",
		}
		if v := data["last_duration_ms"]; v != "" {
			if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
				entry.DurationMS = ms
			}
		}
		if v := data["last_mutation_count"]; v != "" {
			if c, err := strconv.ParseInt(v, 10, 64); err == nil {
				entry.MutationCount = int(c)
			}
		}
		entry.PayloadHash = data["last_payload_hash"]

		if ts := parseRFC3339Ptr(data["last_started_at"]); ts != nil {
			entry.StartedAt = ts
		}
		if ts := parseRFC3339Ptr(data["last_committed_at"]); ts != nil {
			entry.CommittedAt = ts
			entry.RecordedAt = *ts
		}
		if ts := parseRFC3339Ptr(data["last_failed_at"]); ts != nil {
			entry.FailedAt = ts
			if entry.RecordedAt.IsZero() {
				entry.RecordedAt = *ts
			}
		}
		if entry.RecordedAt.IsZero() {
			entry.RecordedAt = time.Now().UTC()
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].RecordedAt.Equal(entries[j].RecordedAt) {
			return entries[i].Key < entries[j].Key
		}
		return entries[i].RecordedAt.After(entries[j].RecordedAt)
	})
	return entries, nil
}

func parseRFC3339Ptr(raw string) *time.Time {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	ts, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil
	}
	return &ts
}
