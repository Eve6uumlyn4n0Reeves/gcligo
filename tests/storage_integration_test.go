package tests

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	store "gcli2api-go/internal/storage"
)

func TestRedisBackend_ConfigCRUD(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR not set; skipping Redis integration test")
	}
	pwd := os.Getenv("REDIS_PASSWORD")
	db := 0
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			db = n
		}
	}
	prefix := os.Getenv("REDIS_PREFIX")

	backend, err := store.NewRedisBackend(addr, pwd, db, prefix)
	if err != nil {
		t.Fatalf("new redis backend: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := backend.Initialize(ctx); err != nil {
		t.Fatalf("init redis: %v", err)
	}
	defer backend.Close()

	crudPlanScenario(t, ctx, backend)
}

func TestPostgresBackend_ConfigCRUD(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN not set; skipping Postgres integration test")
	}
	backend, err := store.NewPostgresBackend(dsn)
	if err != nil {
		t.Fatalf("new postgres backend: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := backend.Initialize(ctx); err != nil {
		t.Fatalf("init postgres: %v", err)
	}
	defer backend.Close()

	crudPlanScenario(t, ctx, backend)
}

func TestMongoBackend_ConfigCRUD(t *testing.T) {
	uri := os.Getenv("MONGO_URI")
	db := os.Getenv("MONGO_DB")
	if uri == "" || db == "" {
		t.Skip("MONGO_URI/MONGO_DB not set; skipping Mongo integration test")
	}
	backend, err := store.NewMongoDBBackend(uri, db)
	if err != nil {
		t.Fatalf("new mongo backend: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := backend.Initialize(ctx); err != nil {
		t.Fatalf("init mongo: %v", err)
	}
	defer backend.Close()

	crudPlanScenario(t, ctx, backend)
}

// crudPlanScenario performs minimal config CRUD + plan save/load simulation.
func crudPlanScenario(t *testing.T, ctx context.Context, backend store.Backend) {
	name := "itplan_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	key := "assembly_plan:" + name
	plan := map[string]any{
		"name":      name,
		"timestamp": time.Now().Unix(),
		"variant_config": map[string]any{
			"fake_streaming_prefix":  "假流式/",
			"anti_truncation_prefix": "流式抗截断/",
		},
		"models": map[string]any{
			"openai": []map[string]any{{"base": "gemini-2.5-pro", "enabled": true, "upstream": "code_assist"}},
			"gemini": []map[string]any{{"base": "gemini-2.5-flash", "enabled": true, "upstream": "code_assist"}},
		},
	}

	// Set
	if err := backend.SetConfig(ctx, key, plan); err != nil {
		t.Fatalf("set plan: %v", err)
	}

	// Get
	got, err := backend.GetConfig(ctx, key)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	if got == nil {
		t.Fatalf("get plan returned nil")
	}

	// List
	all, err := backend.ListConfigs(ctx)
	if err != nil {
		t.Fatalf("list configs: %v", err)
	}
	if _, ok := all[key]; !ok {
		t.Fatalf("expected key %s in ListConfigs", key)
	}

	// Simulate apply: write variant_config and ensure Get works
	if err := backend.SetConfig(ctx, "model_variant_config", plan["variant_config"]); err != nil {
		t.Fatalf("set variant_config: %v", err)
	}
	if _, err := backend.GetConfig(ctx, "model_variant_config"); err != nil {
		t.Fatalf("get variant_config: %v", err)
	}
}
