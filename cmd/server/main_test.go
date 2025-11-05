package main

import (
	"context"
	"testing"

	"gcli2api-go/internal/config"
)

func TestBuildStorageBackendUnsupportedAndAuto(t *testing.T) {
	ctx := context.Background()
	// unknown backend should error
	if _, err := buildStorageBackend(ctx, &config.Config{StorageBackend: "unknown"}); err == nil {
		t.Fatalf("expected error for unsupported backend")
	}
	// auto should fall back to file when nothing configured
	cfg := &config.Config{StorageBackend: "auto", AuthDir: t.TempDir()}
	b, err := buildStorageBackend(ctx, cfg)
	if err != nil {
		t.Fatalf("auto backend failed: %v", err)
	}
	if b == nil {
		t.Fatalf("auto backend returned nil")
	}
}
