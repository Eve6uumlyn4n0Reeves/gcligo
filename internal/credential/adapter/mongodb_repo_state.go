package adapter

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetAllCredentialStates 获取所有凭证状态
func (m *MongoDBStorageAdapter) GetAllCredentialStates(ctx context.Context) (map[string]*CredentialState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cursor, err := m.states.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find states: %w", err)
	}
	defer cursor.Close(ctx)

	var stateDocs []StateDocument
	if err := cursor.All(ctx, &stateDocs); err != nil {
		return nil, fmt.Errorf("failed to decode states: %w", err)
	}

	states := make(map[string]*CredentialState)
	for _, stateDoc := range stateDocs {
		state := &CredentialState{
			ID:              stateDoc.CredID,
			Disabled:        stateDoc.Disabled,
			FailureCount:    stateDoc.FailureCount,
			SuccessCount:    stateDoc.SuccessCount,
			FailureReason:   stateDoc.FailureReason,
			HealthScore:     stateDoc.HealthScore,
			UsageStats:      stateDoc.UsageStats,
			ErrorRate:       stateDoc.ErrorRate,
			AvgResponseTime: time.Duration(stateDoc.AvgResponseTime) * time.Millisecond,
			CreatedAt:       stateDoc.CreatedAt.Time(),
			UpdatedAt:       stateDoc.UpdatedAt.Time(),
		}

		if stateDoc.LastUsed != 0 {
			lastUsed := stateDoc.LastUsed.Time()
			state.LastUsed = &lastUsed
		}
		if stateDoc.LastSuccess != 0 {
			lastSuccess := stateDoc.LastSuccess.Time()
			state.LastSuccess = &lastSuccess
		}
		if stateDoc.LastFailure != 0 {
			lastFailure := stateDoc.LastFailure.Time()
			state.LastFailure = &lastFailure
		}

		states[stateDoc.CredID] = state
	}

	return states, nil
}

// UpdateCredentialStates 批量更新凭证状态
func (m *MongoDBStorageAdapter) UpdateCredentialStates(ctx context.Context, states map[string]*CredentialState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		for id, state := range states {
			state.UpdatedAt = time.Now()

			stateDoc := &StateDocument{
				CredID:          state.ID,
				Disabled:        state.Disabled,
				FailureCount:    state.FailureCount,
				SuccessCount:    state.SuccessCount,
				FailureReason:   state.FailureReason,
				HealthScore:     state.HealthScore,
				UsageStats:      state.UsageStats,
				ErrorRate:       state.ErrorRate,
				AvgResponseTime: state.AvgResponseTime.Milliseconds(),
				CreatedAt:       primitive.NewDateTimeFromTime(state.CreatedAt),
				UpdatedAt:       primitive.NewDateTimeFromTime(state.UpdatedAt),
			}

			if state.LastUsed != nil {
				stateDoc.LastUsed = primitive.NewDateTimeFromTime(*state.LastUsed)
			}
			if state.LastSuccess != nil {
				stateDoc.LastSuccess = primitive.NewDateTimeFromTime(*state.LastSuccess)
			}
			if state.LastFailure != nil {
				stateDoc.LastFailure = primitive.NewDateTimeFromTime(*state.LastFailure)
			}

			if _, err := m.states.UpdateOne(
				sessCtx,
				bson.M{"cred_id": id},
				bson.M{"$set": stateDoc},
				options.Update().SetUpsert(true),
			); err != nil {
				return nil, fmt.Errorf("failed to update state for %s: %w", id, err)
			}
		}

		return nil, nil
	})

	return err
}

// DiscoverCredentials 发现凭证
func (m *MongoDBStorageAdapter) DiscoverCredentials(ctx context.Context) ([]*Credential, error) {
	return m.GetAllCredentials(ctx)
}

// RefreshCredential 刷新凭证
func (m *MongoDBStorageAdapter) RefreshCredential(ctx context.Context, credID string) (*Credential, error) {
	return m.LoadCredential(ctx, credID)
}

// UpdateUsageStats 更新使用统计
func (m *MongoDBStorageAdapter) UpdateUsageStats(ctx context.Context, credID string, stats map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	update := bson.M{
		"$set": bson.M{
			"updated_at": primitive.NewDateTimeFromTime(time.Now()),
		},
	}

	if len(stats) > 0 {
		update["$set"].(bson.M)["usage_stats"] = stats
	}

	if _, err := m.states.UpdateOne(
		ctx,
		bson.M{"cred_id": credID},
		update,
		options.Update().SetUpsert(true),
	); err != nil {
		return fmt.Errorf("failed to update usage stats: %w", err)
	}

	return nil
}

// GetUsageStats 获取使用统计
func (m *MongoDBStorageAdapter) GetUsageStats(ctx context.Context, credID string) (map[string]interface{}, error) {
	var stateDoc StateDocument
	if err := m.states.FindOne(ctx, bson.M{"cred_id": credID}).Decode(&stateDoc); err != nil {
		if err == mongo.ErrNoDocuments {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	if stateDoc.UsageStats == nil {
		return make(map[string]interface{}), nil
	}

	stats := make(map[string]interface{})
	for k, v := range stateDoc.UsageStats {
		stats[k] = v
	}

	return stats, nil
}

// GetUsageStatsSummary 获取使用统计汇总
func (m *MongoDBStorageAdapter) GetUsageStatsSummary(ctx context.Context) (map[string]interface{}, error) {
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":               nil,
				"total_credentials": bson.M{"$sum": 1},
				"disabled_count": bson.M{
					"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$disabled", true}}, 1, 0}},
				},
				"total_success":    bson.M{"$sum": "$success_count"},
				"total_failure":    bson.M{"$sum": "$failure_count"},
				"avg_health_score": bson.M{"$avg": "$health_score"},
				"avg_error_rate":   bson.M{"$avg": "$error_rate"},
			},
		},
	}

	cursor, err := m.states.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalCredentials int     `bson:"total_credentials"`
		DisabledCount    int     `bson:"disabled_count"`
		TotalSuccess     int64   `bson:"total_success"`
		TotalFailure     int64   `bson:"total_failure"`
		AvgHealthScore   float64 `bson:"avg_health_score"`
		AvgErrorRate     float64 `bson:"avg_error_rate"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode aggregation result: %w", err)
		}
	}

	summary := map[string]interface{}{
		"total_credentials":    result.TotalCredentials,
		"active_credentials":   result.TotalCredentials - result.DisabledCount,
		"disabled_credentials": result.DisabledCount,
		"total_success":        result.TotalSuccess,
		"total_failure":        result.TotalFailure,
		"avg_health_score":     result.AvgHealthScore,
		"avg_error_rate":       result.AvgErrorRate,
	}

	return summary, nil
}
