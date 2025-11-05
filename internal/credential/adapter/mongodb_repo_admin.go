package adapter

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// EnableCredentials 启用凭证
func (m *MongoDBStorageAdapter) EnableCredentials(ctx context.Context, credIDs []string) error {
	return m.updateCredentialStatus(ctx, credIDs, false)
}

// DisableCredentials 禁用凭证
func (m *MongoDBStorageAdapter) DisableCredentials(ctx context.Context, credIDs []string) error {
	return m.updateCredentialStatus(ctx, credIDs, true)
}

// DeleteCredentials 批量删除凭证
func (m *MongoDBStorageAdapter) DeleteCredentials(ctx context.Context, credIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, err := m.credentials.DeleteMany(sessCtx, bson.M{"cred_id": bson.M{"$in": credIDs}}); err != nil {
			return nil, fmt.Errorf("failed to delete credentials: %w", err)
		}

		if _, err := m.states.DeleteMany(sessCtx, bson.M{"cred_id": bson.M{"$in": credIDs}}); err != nil {
			return nil, fmt.Errorf("failed to delete states: %w", err)
		}

		return nil, nil
	})
	if err != nil {
		return err
	}

	go m.notifyWatchers()

	return nil
}

// GetHealthyCredentials 获取健康凭证
func (m *MongoDBStorageAdapter) GetHealthyCredentials(ctx context.Context) ([]*Credential, error) {
	return m.GetCredentialsByFilter(ctx, &CredentialFilter{
		Disabled:  boolPtr(false),
		MinHealth: float64Ptr(0.5),
	})
}

// GetUnhealthyCredentials 获取不健康凭证
func (m *MongoDBStorageAdapter) GetUnhealthyCredentials(ctx context.Context) ([]*Credential, error) {
	return m.GetCredentialsByFilter(ctx, &CredentialFilter{
		Disabled: boolPtr(true),
	})
}

// ValidateCredential 验证凭证
func (m *MongoDBStorageAdapter) ValidateCredential(ctx context.Context, cred *Credential) error {
	if cred.ID == "" {
		return fmt.Errorf("credential ID cannot be empty")
	}
	if cred.Type == "" {
		return fmt.Errorf("credential type cannot be empty")
	}

	switch cred.Type {
	case "oauth":
		if cred.Token == "" && cred.RefreshToken == "" {
			return fmt.Errorf("OAuth credential must have token or refresh token")
		}
	case "api_key":
		if cred.Token == "" {
			return fmt.Errorf("API key credential must have token")
		}
	case "service_account":
		if cred.ClientID == "" {
			return fmt.Errorf("Service account must have client ID")
		}
	}

	return nil
}

func (m *MongoDBStorageAdapter) updateCredentialStatus(ctx context.Context, credIDs []string, disabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	update := bson.M{
		"$set": bson.M{
			"disabled":   disabled,
			"updated_at": primitive.NewDateTimeFromTime(time.Now()),
		},
	}

	if _, err := m.states.UpdateMany(
		ctx,
		bson.M{"cred_id": bson.M{"$in": credIDs}},
		update,
	); err != nil {
		return fmt.Errorf("failed to update credential status: %w", err)
	}

	return nil
}
