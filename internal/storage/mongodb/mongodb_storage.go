package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	storagecommon "gcli2api-go/internal/storage/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ✅ MongoDBStorage implements storage using MongoDB
type MongoDBStorage struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
	uri        string
	dbName     string
}

const defaultMongoTimeout = 5 * time.Second
const planCommitRetentionSeconds int32 = 14 * 24 * 3600

// ensureMongoTimeout is deprecated, use storagecommon.WithStorageTimeout instead
func ensureMongoTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return storagecommon.WithStorageTimeout(ctx, defaultMongoTimeout)
}

// NewMongoDBStorage creates a new MongoDB storage backend
func NewMongoDBStorage(uri string, dbName string) (*MongoDBStorage, error) {
	if dbName == "" {
		dbName = "gcli2api"
	}

	return &MongoDBStorage{
		uri:    uri,
		dbName: dbName,
	}, nil
}

// Initialize connects to MongoDB
func (m *MongoDBStorage) Initialize(ctx context.Context) error {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	clientOptions := options.Client().ApplyURI(m.uri)
	clientOptions.SetMaxPoolSize(10)
	clientOptions.SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test connection
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.client = client
	m.database = client.Database(m.dbName)
	m.collection = m.database.Collection("credentials")

	// Create indexes for credentials
	_, err = m.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "project_id", Value: 1}},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Create indexes for usage stats and configs
	if _, err := m.database.Collection("usage_stats").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "key", Value: 1}},
	}); err != nil {
		return fmt.Errorf("failed to create usage_stats index: %w", err)
	}
	if _, err := m.database.Collection("configs").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "key", Value: 1}},
	}); err != nil {
		return fmt.Errorf("failed to create configs index: %w", err)
	}

	lockColl := m.database.Collection("config_plan_locks")
	if _, err := lockColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}); err != nil {
		return fmt.Errorf("failed to create plan lock TTL index: %w", err)
	}

	commitColl := m.database.Collection("config_plan_commits")
	if _, err := commitColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "key", Value: 1}, {Key: "committed_at", Value: -1}},
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(planCommitRetentionSeconds),
		},
	}); err != nil {
		return fmt.Errorf("failed to create plan commit indexes: %w", err)
	}

	return nil
}

// PoolStats returns approximate connection pool stats.
// Note: MongoDB Go driver does not expose idle connection count directly.
// We approximate Active using NumberSessionsInProgress and leave Idle=0.
func (m *MongoDBStorage) PoolStats(ctx context.Context) (active int64, idle int64, err error) {
	if m == nil || m.client == nil {
		return 0, 0, fmt.Errorf("mongo client not initialized")
	}
	// Try to use driver session count as an approximation of active usage.
	// This does not require admin privileges unlike serverStatus.
	active = int64(m.client.NumberSessionsInProgress())
	return active, 0, nil
}

// Close closes MongoDB connection
func (m *MongoDBStorage) Close() error {
	if m.client != nil {
		return m.client.Disconnect(context.Background())
	}
	return nil
}

// GetCredential retrieves a credential by ID
func (m *MongoDBStorage) GetCredential(ctx context.Context, id string) ([]byte, error) {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	var result bson.M
	err := m.collection.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("credential not found: %s", id)
		}
		return nil, err
	}

	// Remove MongoDB _id field
	delete(result, "_id")

	return json.Marshal(result)
}

// SetCredential stores a credential
func (m *MongoDBStorage) SetCredential(ctx context.Context, id string, data []byte) error {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	doc["id"] = id
	doc["updated_at"] = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err := m.collection.ReplaceOne(ctx, bson.M{"id": id}, doc, opts)
	return err
}

// DeleteCredential removes a credential
func (m *MongoDBStorage) DeleteCredential(ctx context.Context, id string) error {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	_, err := m.collection.DeleteOne(ctx, bson.M{"id": id})
	return err
}

// ListCredentials lists all credentials
func (m *MongoDBStorage) ListCredentials(ctx context.Context) ([]string, error) {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	cursor, err := m.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		if id, ok := result["id"].(string); ok {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// BulkGetCredentials retrieves multiple credentials in a single query.
func (m *MongoDBStorage) BulkGetCredentials(ctx context.Context, ids []string) (map[string][]byte, error) {
	if len(ids) == 0 {
		return map[string][]byte{}, nil
	}

	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	cursor, err := m.collection.Find(ctx, bson.M{"id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[string][]byte, len(ids))
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		idVal, ok := doc["id"].(string)
		if !ok || idVal == "" {
			continue
		}
		delete(doc, "_id")
		payload, err := json.Marshal(doc)
		if err != nil {
			continue
		}
		result[idVal] = payload
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// BulkUpsertCredentials performs unordered bulk upserts for credentials.
func (m *MongoDBStorage) BulkUpsertCredentials(ctx context.Context, items map[string][]byte) error {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	if len(items) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(items))
	now := time.Now()
	for id, raw := range items {
		var doc map[string]interface{}
		if err := json.Unmarshal(raw, &doc); err != nil {
			return err
		}
		doc["id"] = id
		doc["updated_at"] = now
		rm := mongo.NewReplaceOneModel().SetFilter(bson.M{"id": id}).SetReplacement(doc).SetUpsert(true)
		models = append(models, rm)
	}
	_, err := m.collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	return err
}

// BulkDeleteCredentials performs unordered bulk deletes for credentials.
func (m *MongoDBStorage) BulkDeleteCredentials(ctx context.Context, ids []string) error {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	if len(ids) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(ids))
	for _, id := range ids {
		dm := mongo.NewDeleteOneModel().SetFilter(bson.M{"id": id})
		models = append(models, dm)
	}
	_, err := m.collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	return err
}

// IncrementUsage increments usage counter
func (m *MongoDBStorage) IncrementUsage(ctx context.Context, key string, field string, value int64) error {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	statsCollection := m.database.Collection("usage_stats")

	filter := bson.M{"key": key}
	update := bson.M{
		"$inc": bson.M{field: value},
		"$set": bson.M{"updated_at": time.Now()},
	}

	opts := options.Update().SetUpsert(true)
	_, err := statsCollection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetUsage retrieves usage statistics
func (m *MongoDBStorage) GetUsage(ctx context.Context, key string) (map[string]interface{}, error) {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	statsCollection := m.database.Collection("usage_stats")

	var result bson.M
	err := statsCollection.FindOne(ctx, bson.M{"key": key}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}

	delete(result, "_id")
	delete(result, "key")
	delete(result, "updated_at")

	// Convert to map
	data := make(map[string]interface{})
	for k, v := range result {
		data[k] = v
	}

	return data, nil
}

// ResetUsage resets usage statistics
func (m *MongoDBStorage) ResetUsage(ctx context.Context, key string) error {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	statsCollection := m.database.Collection("usage_stats")
	_, err := statsCollection.DeleteOne(ctx, bson.M{"key": key})
	return err
}

// ListUsage lists all usage counters grouped by key.
func (m *MongoDBStorage) ListUsage(ctx context.Context) (map[string]map[string]interface{}, error) {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	statsCollection := m.database.Collection("usage_stats")
	cursor, err := statsCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[string]map[string]interface{})
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		key, _ := doc["key"].(string)
		if key == "" {
			continue
		}
		delete(doc, "_id")
		delete(doc, "key")
		delete(doc, "updated_at")
		inner := make(map[string]interface{})
		for k, v := range doc {
			inner[k] = v
		}
		result[key] = inner
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (m *MongoDBStorage) SetConfig(ctx context.Context, key string, value interface{}) error {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	configCollection := m.database.Collection("configs")
	doc := bson.M{
		"_id":        key,
		"key":        key,
		"value":      value,
		"updated_at": time.Now(),
	}
	opts := options.Replace().SetUpsert(true)
	_, err := configCollection.ReplaceOne(ctx, bson.M{"_id": key}, doc, opts)
	return err
}

func (m *MongoDBStorage) GetConfig(ctx context.Context, key string) (interface{}, error) {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	configCollection := m.database.Collection("configs")
	var doc bson.M
	err := configCollection.FindOne(ctx, bson.M{"_id": key}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return doc["value"], nil
}

func (m *MongoDBStorage) DeleteConfig(ctx context.Context, key string) error {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	configCollection := m.database.Collection("configs")
	res, err := configCollection.DeleteOne(ctx, bson.M{"_id": key})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (m *MongoDBStorage) ListConfigs(ctx context.Context) (map[string]interface{}, error) {
	ctx, cancel := ensureMongoTimeout(ctx)
	defer cancel()
	configCollection := m.database.Collection("configs")
	cursor, err := configCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[string]interface{})
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		key, _ := doc["key"].(string)
		if key == "" {
			continue
		}
		result[key] = doc["value"]
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// PlanLocksCollection returns the collection used to coordinate idempotent plan writes.
func (m *MongoDBStorage) PlanLocksCollection() *mongo.Collection {
	if m == nil || m.database == nil {
		return nil
	}
	return m.database.Collection("config_plan_locks")
}

// PlanCommitCollection returns the collection used to persist plan apply audit logs.
func (m *MongoDBStorage) PlanCommitCollection() *mongo.Collection {
	if m == nil || m.database == nil {
		return nil
	}
	return m.database.Collection("config_plan_commits")
}
