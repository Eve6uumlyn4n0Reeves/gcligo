package adapter

import (
	"context"
	"testing"
)

func TestNewRedisStorageAdapter(t *testing.T) {
	tests := []struct {
		name    string
		config  *RedisStorageConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &RedisStorageConfig{
				Addr:   "localhost:6379",
				DB:     0,
				Prefix: "test:",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty address",
			config: &RedisStorageConfig{
				Addr:   "",
				DB:     0,
				Prefix: "test:",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewRedisStorageAdapter(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Skipf("Redis not available, skipping test (%v)", err)
			}
			t.Cleanup(func() {
				adapter.Close()
			})
		})
	}
}

func TestRedisStorageAdapter_Ping(t *testing.T) {
	cfg := &RedisStorageConfig{
		Addr:   "localhost:6379",
		DB:     0,
		Prefix: "test:",
	}

	adapter, err := NewRedisStorageAdapter(cfg)
	if err != nil {
		t.Skipf("Redis not available, skipping test (%v)", err)
	}
	defer adapter.Close()

	ctx := context.Background()
	if err := adapter.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestRedisStorageAdapter_GetSetConfig(t *testing.T) {
	cfg := &RedisStorageConfig{
		Addr:   "localhost:6379",
		DB:     0,
		Prefix: "test:",
		Config: map[string]interface{}{
			"mode": "rw",
		},
	}

	adapter, err := NewRedisStorageAdapter(cfg)
	if err != nil {
		t.Skipf("Redis not available, skipping test (%v)", err)
	}
	defer adapter.Close()

	// Verify initial config snapshot
	gotConfig := adapter.GetConfig()
	if gotConfig["mode"] != "rw" {
		t.Errorf("GetConfig() mode = %v, want %v", gotConfig["mode"], "rw")
	}

	// Update config via SetConfig
	ctx := context.Background()
	newConfig := map[string]interface{}{
		"mode":  "ro",
		"stage": "canary",
	}
	if err := adapter.SetConfig(ctx, newConfig); err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}

	gotConfig = adapter.GetConfig()
	if gotConfig["mode"] != "ro" || gotConfig["stage"] != "canary" {
		t.Errorf("GetConfig() after SetConfig = %v, want %v", gotConfig, newConfig)
	}
}
