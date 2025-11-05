//go:build !stats_isolation

package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const planCommitRetention = 14 * 24 * time.Hour

type configSnapshot struct {
	key    string
	value  interface{}
	exists bool
}

// ApplyConfigBatch implements ConfigBatchApplier for MongoDB backends using a coordinator collection.
func (m *MongoDBBackend) ApplyConfigBatch(ctx context.Context, mutations []ConfigMutation, opts BatchApplyOptions) error {
	if m == nil || m.storage == nil {
		return errors.New("mongodb backend not initialized")
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

	payloadHash := ""
	if payloadBytes, err := json.Marshal(mutations); err == nil {
		sum := sha256.Sum256(payloadBytes)
		payloadHash = hex.EncodeToString(sum[:])
	}

	lockColl := m.storage.PlanLocksCollection()
	if lockColl == nil {
		return errors.New("mongodb plan lock collection unavailable")
	}
	commitColl := m.storage.PlanCommitCollection()

	now := time.Now().UTC()
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = defaultPlanTTL
	}
	expiresAt := now.Add(ttl)

	var existing struct {
		Status    string    `bson:"status"`
		ExpiresAt time.Time `bson:"expires_at"`
	}
	err := lockColl.FindOne(ctx, bson.M{"_id": opts.IdempotencyKey}).Decode(&existing)
	switch err {
	case nil:
		if existing.Status == "committed" {
			return nil
		}
		if existing.Status == "in_progress" && existing.ExpiresAt.After(now) {
			return fmt.Errorf("plan apply already in progress for key %s", opts.IdempotencyKey)
		}
	case mongo.ErrNoDocuments:
		// proceed to acquire lock
	default:
		return err
	}

	filter := bson.M{
		"_id": opts.IdempotencyKey,
		"$or": []bson.M{
			{"status": bson.M{"$ne": "in_progress"}},
			{"expires_at": bson.M{"$lte": now}},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"status":          "in_progress",
			"stage":           stage,
			"expires_at":      expiresAt,
			"last_started_at": now,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	res, err := lockColl.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 && res.UpsertedCount == 0 {
		return fmt.Errorf("plan apply already in progress for key %s", opts.IdempotencyKey)
	}

	history := make([]configSnapshot, 0, len(mutations))
	for _, mut := range mutations {
		val, err := m.GetConfig(ctx, mut.Key)
		if err != nil {
			var nf *ErrNotFound
			if errors.As(err, &nf) {
				history = append(history, configSnapshot{key: mut.Key, exists: false})
				continue
			}
			_ = markPlanLockFailed(ctx, lockColl, opts.IdempotencyKey, stage, err)
			logPlanCommit(ctx, commitColl, opts.IdempotencyKey, stage, "failed", time.Since(start), payloadHash, mutations, err)
			return err
		}
		history = append(history, configSnapshot{key: mut.Key, value: val, exists: true})
	}

	if err := m.applyMongoMutations(ctx, mutations); err != nil {
		restoreErr := m.restoreMongoSnapshots(ctx, history)
		combined := wrapApplyError(err, restoreErr)
		_ = markPlanLockFailed(ctx, lockColl, opts.IdempotencyKey, stage, combined)
		logPlanCommit(ctx, commitColl, opts.IdempotencyKey, stage, "failed", time.Since(start), payloadHash, mutations, combined)
		return combined
	}

	duration := time.Since(start)
	_, err = lockColl.UpdateOne(ctx,
		bson.M{"_id": opts.IdempotencyKey},
		bson.M{"$set": bson.M{"status": "committed", "stage": stage, "committed_at": time.Now().UTC(), "expires_at": expiresAt}},
	)
	if err != nil {
		_ = markPlanLockFailed(ctx, lockColl, opts.IdempotencyKey, stage, err)
		logPlanCommit(ctx, commitColl, opts.IdempotencyKey, stage, "failed", duration, payloadHash, mutations, err)
		return err
	}

	logPlanCommit(ctx, commitColl, opts.IdempotencyKey, stage, "committed", duration, payloadHash, mutations, nil)
	return nil
}

func (m *MongoDBBackend) applyMongoMutations(ctx context.Context, mutations []ConfigMutation) error {
	for _, mut := range mutations {
		if mut.Delete {
			if err := m.DeleteConfig(ctx, mut.Key); err != nil {
				var nf *ErrNotFound
				if errors.As(err, &nf) {
					continue
				}
				return err
			}
			continue
		}
		if err := m.SetConfig(ctx, mut.Key, mut.Value); err != nil {
			return err
		}
	}
	return nil
}

func (m *MongoDBBackend) restoreMongoSnapshots(ctx context.Context, snapshots []configSnapshot) error {
	for i := len(snapshots) - 1; i >= 0; i-- {
		snap := snapshots[i]
		if !snap.exists {
			if err := m.DeleteConfig(ctx, snap.key); err != nil {
				var nf *ErrNotFound
				if errors.As(err, &nf) {
					continue
				}
				return err
			}
			continue
		}
		if err := m.SetConfig(ctx, snap.key, snap.value); err != nil {
			return err
		}
	}
	return nil
}

func markPlanLockFailed(ctx context.Context, coll *mongo.Collection, id, stage string, cause error) error {
	if coll == nil || id == "" {
		return nil
	}
	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"status":         "failed",
			"failed_at":      now,
			"error":          cause.Error(),
			"stage":          stage,
			"expires_at":     now.Add(30 * time.Second),
			"last_attempt":   now,
			"last_failed_at": now,
		},
	}
	_, err := coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func logPlanCommit(ctx context.Context, coll *mongo.Collection, key, stage, status string, duration time.Duration, payloadHash string, mutations []ConfigMutation, cause error) {
	if coll == nil || key == "" {
		return
	}
	doc := bson.M{
		"backend":        "mongodb",
		"key":            key,
		"stage":          stage,
		"status":         status,
		"duration_ms":    duration.Milliseconds(),
		"recorded_at":    time.Now().UTC(),
		"expires_at":     time.Now().UTC().Add(planCommitRetention),
		"mutation_count": len(mutations),
		"mutations":      summarizeMutations(mutations),
		"payload_hash":   payloadHash,
	}
	if cause != nil {
		doc["error"] = cause.Error()
		doc["failed_at"] = time.Now().UTC()
	} else if status == "committed" {
		doc["committed_at"] = time.Now().UTC()
	}
	_, _ = coll.InsertOne(ctx, doc)
}

func summarizeMutations(mutations []ConfigMutation) []bson.M {
	if len(mutations) == 0 {
		return nil
	}
	summary := make([]bson.M, 0, len(mutations))
	for _, mut := range mutations {
		entry := bson.M{
			"key":    mut.Key,
			"delete": mut.Delete,
		}
		if !mut.Delete && mut.Value != nil {
			entry["value_type"] = fmt.Sprintf("%T", mut.Value)
		}
		summary = append(summary, entry)
	}
	return summary
}

func wrapApplyError(applyErr, restoreErr error) error {
	if restoreErr == nil {
		return applyErr
	}
	return fmt.Errorf("apply failed: %w (restore error: %v)", applyErr, restoreErr)
}

// ExportPlanAudit returns commit entries recorded in MongoDB.
func (m *MongoDBBackend) ExportPlanAudit(ctx context.Context) ([]PlanAuditEntry, error) {
	if m == nil || m.storage == nil {
		return nil, errors.New("mongodb backend not initialized")
	}
	commitColl := m.storage.PlanCommitCollection()
	if commitColl == nil {
		return nil, errors.New("plan commit collection unavailable")
	}

	lockInfo := make(map[string]struct {
		Status        string
		Stage         string
		LastStartedAt *time.Time
		LastCommitted *time.Time
		LastFailed    *time.Time
		Error         string
	})

	if lockColl := m.storage.PlanLocksCollection(); lockColl != nil {
		lockCursor, err := lockColl.Find(ctx, bson.M{})
		if err != nil {
			return nil, err
		}
		defer lockCursor.Close(ctx)
		for lockCursor.Next(ctx) {
			var doc struct {
				ID            string     `bson:"_id"`
				Status        string     `bson:"status"`
				Stage         string     `bson:"stage"`
				LastStartedAt *time.Time `bson:"last_started_at,omitempty"`
				LastCommitted *time.Time `bson:"last_committed_at,omitempty"`
				LastFailed    *time.Time `bson:"last_failed_at,omitempty"`
				Error         string     `bson:"error,omitempty"`
			}
			if err := lockCursor.Decode(&doc); err != nil {
				return nil, err
			}
			lockInfo[doc.ID] = struct {
				Status        string
				Stage         string
				LastStartedAt *time.Time
				LastCommitted *time.Time
				LastFailed    *time.Time
				Error         string
			}{
				Status:        doc.Status,
				Stage:         doc.Stage,
				LastStartedAt: doc.LastStartedAt,
				LastCommitted: doc.LastCommitted,
				LastFailed:    doc.LastFailed,
				Error:         doc.Error,
			}
		}
		if err := lockCursor.Err(); err != nil {
			return nil, err
		}
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "recorded_at", Value: -1}})
	cursor, err := commitColl.Find(ctx, bson.M{}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	entries := make([]PlanAuditEntry, 0)
	for cursor.Next(ctx) {
		var doc struct {
			Key           string     `bson:"key"`
			Stage         string     `bson:"stage"`
			Status        string     `bson:"status"`
			DurationMS    int64      `bson:"duration_ms"`
			MutationCount int        `bson:"mutation_count"`
			PayloadHash   string     `bson:"payload_hash,omitempty"`
			Error         string     `bson:"error,omitempty"`
			RecordedAt    time.Time  `bson:"recorded_at"`
			CommittedAt   *time.Time `bson:"committed_at,omitempty"`
			FailedAt      *time.Time `bson:"failed_at,omitempty"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}

		entry := PlanAuditEntry{
			Backend:       "mongodb",
			Key:           doc.Key,
			Stage:         doc.Stage,
			Status:        doc.Status,
			DurationMS:    doc.DurationMS,
			MutationCount: doc.MutationCount,
			PayloadHash:   doc.PayloadHash,
			Error:         doc.Error,
			Source:        "mongodb:config_plan_commits",
			RecordedAt:    doc.RecordedAt,
		}
		if doc.CommittedAt != nil && !doc.CommittedAt.IsZero() {
			entry.CommittedAt = doc.CommittedAt
		}
		if doc.FailedAt != nil && !doc.FailedAt.IsZero() {
			entry.FailedAt = doc.FailedAt
		}

		if lock, ok := lockInfo[doc.Key]; ok {
			if entry.Stage == "" {
				entry.Stage = lock.Stage
			}
			if entry.Status == "" {
				entry.Status = lock.Status
			}
			if lock.LastStartedAt != nil {
				entry.StartedAt = lock.LastStartedAt
			}
			if entry.CommittedAt == nil && lock.LastCommitted != nil {
				entry.CommittedAt = lock.LastCommitted
			}
			if entry.FailedAt == nil && lock.LastFailed != nil {
				entry.FailedAt = lock.LastFailed
			}
			if entry.Error == "" {
				entry.Error = lock.Error
			}
		}

		entries = append(entries, entry)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].RecordedAt.Equal(entries[j].RecordedAt) {
			return entries[i].Key < entries[j].Key
		}
		return entries[i].RecordedAt.After(entries[j].RecordedAt)
	})

	return entries, nil
}
