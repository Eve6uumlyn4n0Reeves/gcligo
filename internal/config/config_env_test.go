package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadFromEnvPreferredBaseModels(t *testing.T) {
	t.Setenv("DISABLED_MODELS", "model.a , model.b")
	t.Setenv("PREFERRED_BASE_MODELS", "alpha , beta ,gamma")

	cfg := loadFromEnv()

	expectedDisabled := []string{"model.a", "model.b"}
	if !reflect.DeepEqual(expectedDisabled, cfg.DisabledModels) {
		t.Fatalf("expected disabled models %v, got %v", expectedDisabled, cfg.DisabledModels)
	}

	expectedPreferred := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(expectedPreferred, cfg.PreferredBaseModels) {
		t.Fatalf("expected preferred models %v, got %v", expectedPreferred, cfg.PreferredBaseModels)
	}
}

func TestLoadFromEnvLogFileAndRouting(t *testing.T) {
	t.Setenv("LOG_FILE", "/tmp/gcli2api/app.log")
	t.Setenv("PERSIST_ROUTING_STATE", "true")
	t.Setenv("ROUTING_PERSIST_INTERVAL_SEC", "90")
	t.Setenv("AUTO_IMAGE_PLACEHOLDER", "0")

	cfg := loadFromEnv()
	if cfg.LogFile != "/tmp/gcli2api/app.log" {
		t.Fatalf("expected log file /tmp/gcli2api/app.log, got %s", cfg.LogFile)
	}
	if !cfg.PersistRoutingState {
		t.Fatalf("expected persist routing state to be true")
	}
	if cfg.RoutingPersistIntervalSec != 90 {
		t.Fatalf("expected routing interval 90, got %d", cfg.RoutingPersistIntervalSec)
	}
	if cfg.AutoImagePlaceholder {
		t.Fatalf("expected auto image placeholder disabled")
	}
}

func TestValidateAndExpandPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{
		AuthDir:              "~/auths",
		StorageBaseDir:       "~/storage",
		LogFile:              "~/logs/app.log",
		AutoImagePlaceholder: true,
	}

	if err := cfg.ValidateAndExpandPaths(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if !strings.HasPrefix(cfg.AuthDir, home) {
		t.Fatalf("expected auth dir to be expanded under %s, got %s", home, cfg.AuthDir)
	}
	if !strings.HasPrefix(cfg.StorageBaseDir, home) {
		t.Fatalf("expected storage dir to be expanded under %s, got %s", home, cfg.StorageBaseDir)
	}
	if !strings.HasPrefix(cfg.LogFile, home+string(filepath.Separator)) {
		t.Fatalf("expected log file to be expanded under %s, got %s", home, cfg.LogFile)
	}
}
