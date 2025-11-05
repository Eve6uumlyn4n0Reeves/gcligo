package adapter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDBStorageAdapter 提供 MongoDB 实现的凭证存储。
type MongoDBStorageAdapter struct {
	client      *mongo.Client
	database    *mongo.Database
	credentials *mongo.Collection
	states      *mongo.Collection
	mu          sync.RWMutex
	config      map[string]interface{}
	dbName      string
	collection  string
	watchers    *watcherHub
}

// MongoDBStorageConfig MongoDB存储配置
type MongoDBStorageConfig struct {
	URI              string                 `yaml:"uri" json:"uri"`
	Database         string                 `yaml:"database" json:"database"`
	Collection       string                 `yaml:"collection" json:"collection"`
	StatesCollection string                 `yaml:"states_collection" json:"states_collection"`
	Timeout          time.Duration          `yaml:"timeout" json:"timeout"`
	Config           map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// CredentialDocument MongoDB中的凭证文档结构
type CredentialDocument struct {
	ID           primitive.ObjectID     `bson:"_id,omitempty"`
	CredID       string                 `bson:"cred_id"`
	Name         string                 `bson:"name"`
	Type         string                 `bson:"type"`
	Token        string                 `bson:"token"`
	RefreshToken string                 `bson:"refresh_token"`
	ClientID     string                 `bson:"client_id"`
	ExpiresAt    primitive.DateTime     `bson:"expires_at,omitempty"`
	Metadata     map[string]interface{} `bson:"metadata,omitempty"`
	FilePath     string                 `bson:"file_path,omitempty"`
	CreatedAt    primitive.DateTime     `bson:"created_at"`
	UpdatedAt    primitive.DateTime     `bson:"updated_at"`
}

// StateDocument MongoDB中的状态文档结构
type StateDocument struct {
	ID              primitive.ObjectID     `bson:"_id,omitempty"`
	CredID          string                 `bson:"cred_id"`
	Disabled        bool                   `bson:"disabled"`
	LastUsed        primitive.DateTime     `bson:"last_used,omitempty"`
	LastSuccess     primitive.DateTime     `bson:"last_success,omitempty"`
	LastFailure     primitive.DateTime     `bson:"last_failure,omitempty"`
	FailureCount    int                    `bson:"failure_count"`
	SuccessCount    int                    `bson:"success_count"`
	FailureReason   string                 `bson:"failure_reason,omitempty"`
	HealthScore     float64                `bson:"health_score"`
	UsageStats      map[string]interface{} `bson:"usage_stats,omitempty"`
	ErrorRate       float64                `bson:"error_rate"`
	AvgResponseTime int64                  `bson:"avg_response_time"`
	CreatedAt       primitive.DateTime     `bson:"created_at"`
	UpdatedAt       primitive.DateTime     `bson:"updated_at"`
}

// NewMongoDBStorageAdapter 创建新的MongoDB存储适配器
func NewMongoDBStorageAdapter(config *MongoDBStorageConfig) (*MongoDBStorageAdapter, error) {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.Database == "" {
		config.Database = "gcli2api"
	}
	if config.Collection == "" {
		config.Collection = "credentials"
	}
	if config.StatesCollection == "" {
		config.StatesCollection = "credential_states"
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(config.Database)

	adapter := &MongoDBStorageAdapter{
		client:      client,
		database:    database,
		credentials: database.Collection(config.Collection),
		states:      database.Collection(config.StatesCollection),
		config:      config.Config,
		dbName:      config.Database,
		collection:  config.Collection,
		watchers:    newWatcherHub(),
	}

	if err := adapter.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return adapter, nil
}

// createIndexes 创建必要的索引
func (m *MongoDBStorageAdapter) createIndexes(ctx context.Context) error {
	credIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "cred_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
	}

	if _, err := m.credentials.Indexes().CreateMany(ctx, credIndexes); err != nil {
		return fmt.Errorf("failed to create credential indexes: %w", err)
	}

	stateIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "cred_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "disabled", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "health_score", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
	}

	if _, err := m.states.Indexes().CreateMany(ctx, stateIndexes); err != nil {
		return fmt.Errorf("failed to create state indexes: %w", err)
	}

	return nil
}

// Ping 检查MongoDB连接
func (m *MongoDBStorageAdapter) Ping(ctx context.Context) error {
	return m.client.Ping(ctx, readpref.Primary())
}

// Close 关闭MongoDB连接
func (m *MongoDBStorageAdapter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.client.Disconnect(context.Background())
}

// GetConfig 获取配置
func (m *MongoDBStorageAdapter) GetConfig() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config := make(map[string]interface{})
	for k, v := range m.config {
		config[k] = v
	}
	return config
}

// SetConfig 设置配置
func (m *MongoDBStorageAdapter) SetConfig(ctx context.Context, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config
	return nil
}
