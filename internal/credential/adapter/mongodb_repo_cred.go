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

// StoreCredential 存储凭证
func (m *MongoDBStorageAdapter) StoreCredential(ctx context.Context, cred *Credential) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if cred.State == nil {
		cred.State = &CredentialState{
			ID:          cred.ID,
			HealthScore: 1.0,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	cred.State.UpdatedAt = now

	credDoc := &CredentialDocument{
		CredID:       cred.ID,
		Name:         cred.Name,
		Type:         cred.Type,
		Token:        cred.Token,
		RefreshToken: cred.RefreshToken,
		ClientID:     cred.ClientID,
		Metadata:     cred.Metadata,
		FilePath:     cred.FilePath,
		CreatedAt:    primitive.NewDateTimeFromTime(now),
		UpdatedAt:    primitive.NewDateTimeFromTime(now),
	}

	if cred.ExpiresAt != nil {
		credDoc.ExpiresAt = primitive.NewDateTimeFromTime(*cred.ExpiresAt)
	}

	stateDoc := &StateDocument{
		CredID:          cred.State.ID,
		Disabled:        cred.State.Disabled,
		FailureCount:    cred.State.FailureCount,
		SuccessCount:    cred.State.SuccessCount,
		FailureReason:   cred.State.FailureReason,
		HealthScore:     cred.State.HealthScore,
		UsageStats:      cred.State.UsageStats,
		ErrorRate:       cred.State.ErrorRate,
		AvgResponseTime: cred.State.AvgResponseTime.Milliseconds(),
		CreatedAt:       primitive.NewDateTimeFromTime(cred.State.CreatedAt),
		UpdatedAt:       primitive.NewDateTimeFromTime(cred.State.UpdatedAt),
	}

	if cred.State.LastUsed != nil {
		stateDoc.LastUsed = primitive.NewDateTimeFromTime(*cred.State.LastUsed)
	}
	if cred.State.LastSuccess != nil {
		stateDoc.LastSuccess = primitive.NewDateTimeFromTime(*cred.State.LastSuccess)
	}
	if cred.State.LastFailure != nil {
		stateDoc.LastFailure = primitive.NewDateTimeFromTime(*cred.State.LastFailure)
	}

	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, err := m.credentials.UpdateOne(
			sessCtx,
			bson.M{"cred_id": cred.ID},
			bson.M{"$set": credDoc},
			options.Update().SetUpsert(true),
		); err != nil {
			return nil, fmt.Errorf("failed to store credential: %w", err)
		}

		if _, err := m.states.UpdateOne(
			sessCtx,
			bson.M{"cred_id": cred.ID},
			bson.M{"$set": stateDoc},
			options.Update().SetUpsert(true),
		); err != nil {
			return nil, fmt.Errorf("failed to store state: %w", err)
		}

		return nil, nil
	})
	if err != nil {
		return err
	}

	go m.notifyWatchers()

	return nil
}

// LoadCredential 加载凭证
func (m *MongoDBStorageAdapter) LoadCredential(ctx context.Context, id string) (*Credential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var credDoc CredentialDocument
	if err := m.credentials.FindOne(ctx, bson.M{"cred_id": id}).Decode(&credDoc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("credential not found: %s", id)
		}
		return nil, fmt.Errorf("failed to load credential: %w", err)
	}

	cred := &Credential{
		ID:           credDoc.CredID,
		Name:         credDoc.Name,
		Type:         credDoc.Type,
		Token:        credDoc.Token,
		RefreshToken: credDoc.RefreshToken,
		ClientID:     credDoc.ClientID,
		Metadata:     credDoc.Metadata,
		FilePath:     credDoc.FilePath,
	}

	if credDoc.ExpiresAt != 0 {
		expiresAt := credDoc.ExpiresAt.Time()
		cred.ExpiresAt = &expiresAt
	}

	var stateDoc StateDocument
	err := m.states.FindOne(ctx, bson.M{"cred_id": id}).Decode(&stateDoc)
	switch {
	case err == mongo.ErrNoDocuments:
		cred.State = &CredentialState{
			ID:          cred.ID,
			HealthScore: 1.0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	case err != nil:
		return nil, fmt.Errorf("failed to load state: %w", err)
	default:
		cred.State = &CredentialState{
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
			cred.State.LastUsed = &lastUsed
		}
		if stateDoc.LastSuccess != 0 {
			lastSuccess := stateDoc.LastSuccess.Time()
			cred.State.LastSuccess = &lastSuccess
		}
		if stateDoc.LastFailure != 0 {
			lastFailure := stateDoc.LastFailure.Time()
			cred.State.LastFailure = &lastFailure
		}
	}

	return cred, nil
}

// DeleteCredential 删除凭证
func (m *MongoDBStorageAdapter) DeleteCredential(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, err := m.credentials.DeleteOne(sessCtx, bson.M{"cred_id": id}); err != nil {
			return nil, fmt.Errorf("failed to delete credential: %w", err)
		}

		if _, err := m.states.DeleteOne(sessCtx, bson.M{"cred_id": id}); err != nil {
			return nil, fmt.Errorf("failed to delete state: %w", err)
		}

		return nil, nil
	})
	if err != nil {
		return err
	}

	go m.notifyWatchers()

	return nil
}

// UpdateCredential 更新凭证
func (m *MongoDBStorageAdapter) UpdateCredential(ctx context.Context, cred *Credential) error {
	return m.StoreCredential(ctx, cred)
}

// GetAllCredentials 获取所有凭证
func (m *MongoDBStorageAdapter) GetAllCredentials(ctx context.Context) ([]*Credential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cursor, err := m.credentials.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find credentials: %w", err)
	}
	defer cursor.Close(ctx)

	var credDocs []CredentialDocument
	if err := cursor.All(ctx, &credDocs); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	credentials := make([]*Credential, 0, len(credDocs))
	for _, credDoc := range credDocs {
		cred := &Credential{
			ID:           credDoc.CredID,
			Name:         credDoc.Name,
			Type:         credDoc.Type,
			Token:        credDoc.Token,
			RefreshToken: credDoc.RefreshToken,
			ClientID:     credDoc.ClientID,
			Metadata:     credDoc.Metadata,
			FilePath:     credDoc.FilePath,
		}

		if credDoc.ExpiresAt != 0 {
			expiresAt := credDoc.ExpiresAt.Time()
			cred.ExpiresAt = &expiresAt
		}

		var stateDoc StateDocument
		err := m.states.FindOne(ctx, bson.M{"cred_id": cred.ID}).Decode(&stateDoc)
		switch err {
		case nil:
			cred.State = &CredentialState{
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
				cred.State.LastUsed = &lastUsed
			}
			if stateDoc.LastSuccess != 0 {
				lastSuccess := stateDoc.LastSuccess.Time()
				cred.State.LastSuccess = &lastSuccess
			}
			if stateDoc.LastFailure != 0 {
				lastFailure := stateDoc.LastFailure.Time()
				cred.State.LastFailure = &lastFailure
			}
		case mongo.ErrNoDocuments:
			cred.State = &CredentialState{
				ID:          cred.ID,
				HealthScore: 1.0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
		default:
			continue
		}

		credentials = append(credentials, cred)
	}

	return credentials, nil
}

// GetCredentialsByFilter 根据过滤器获取凭证
func (m *MongoDBStorageAdapter) GetCredentialsByFilter(ctx context.Context, filter *CredentialFilter) ([]*Credential, error) {
	credentials, err := m.GetAllCredentials(ctx)
	if err != nil {
		return nil, err
	}

	return ApplyFilter(credentials, filter), nil
}

// buildFilter 构建MongoDB查询过滤器
func (m *MongoDBStorageAdapter) buildFilter(filter *CredentialFilter) bson.M {
	query := bson.M{}

	if filter == nil {
		return query
	}

	if filter.Disabled != nil {
		query["disabled"] = *filter.Disabled
	}

	if filter.Type != "" {
		query["type"] = filter.Type
	}

	if filter.MinHealth != nil {
		query["health_score"] = bson.M{"$gte": *filter.MinHealth}
	}

	if filter.MaxError != nil {
		query["error_rate"] = bson.M{"$lte": *filter.MaxError}
	}

	if filter.LastUsed != nil {
		query["last_used"] = bson.M{"$gte": primitive.NewDateTimeFromTime(*filter.LastUsed)}
	}

	return query
}
