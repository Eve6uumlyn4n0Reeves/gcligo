//go:build legacy_tests
// +build legacy_tests

package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func setupTestFileBackend(t *testing.T) (*FileBackend, string, func()) {
	tmpDir, err := os.MkdirTemp("", "file-backend-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	backend := NewFileBackend(tmpDir)
	ctx := context.Background()

	if err := backend.Initialize(ctx); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to initialize backend: %v", err)
	}

	cleanup := func() {
		backend.Close()
		os.RemoveAll(tmpDir)
	}

	return backend, tmpDir, cleanup
}

func TestFileBackend_Initialize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file-backend-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backend := NewFileBackend(tmpDir)
	ctx := context.Background()

	err = backend.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize() error = %v", err)
	}

	// Check directories were created
	dirs := []string{"credentials", "config", "usage"}
	for _, dir := range dirs {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}
}

func TestFileBackend_CredentialOperations(t *testing.T) {
	backend, _, cleanup := setupTestFileBackend(t)
	defer cleanup()

	ctx := context.Background()
	testID := "test-cred-1"
	testData := map[string]interface{}{
		"id":    testID,
		"token": "test-token",
		"type":  "oauth",
	}

	// Test SetCredential
	err := backend.SetCredential(ctx, testID, testData)
	if err != nil {
		t.Errorf("SetCredential() error = %v", err)
	}

	// Test GetCredential
	retrieved, err := backend.GetCredential(ctx, testID)
	if err != nil {
		t.Errorf("GetCredential() error = %v", err)
	}

	if retrieved["id"] != testID {
		t.Errorf("GetCredential() id = %v, want %v", retrieved["id"], testID)
	}

	// Test ListCredentials
	ids, err := backend.ListCredentials(ctx)
	if err != nil {
		t.Errorf("ListCredentials() error = %v", err)
	}

	if len(ids) != 1 || ids[0] != testID {
		t.Errorf("ListCredentials() = %v, want [%v]", ids, testID)
	}

	// Test DeleteCredential
	err = backend.DeleteCredential(ctx, testID)
	if err != nil {
		t.Errorf("DeleteCredential() error = %v", err)
	}

	// Verify deletion
	_, err = backend.GetCredential(ctx, testID)
	if err == nil {
		t.Error("GetCredential() should return error after deletion")
	}
}

func TestFileBackend_ConfigOperations(t *testing.T) {
	backend, _, cleanup := setupTestFileBackend(t)
	defer cleanup()

	ctx := context.Background()
	testKey := "test-config-key"
	testValue := map[string]interface{}{
		"setting1": "value1",
		"setting2": 123,
	}

	// Test SetConfig
	err := backend.SetConfig(ctx, testKey, testValue)
	if err != nil {
		t.Errorf("SetConfig() error = %v", err)
	}

	// Test GetConfig
	retrieved, err := backend.GetConfig(ctx, testKey)
	if err != nil {
		t.Errorf("GetConfig() error = %v", err)
	}

	if retrieved["setting1"] != "value1" {
		t.Errorf("GetConfig() setting1 = %v, want value1", retrieved["setting1"])
	}

	// Test DeleteConfig
	err = backend.DeleteConfig(ctx, testKey)
	if err != nil {
		t.Errorf("DeleteConfig() error = %v", err)
	}

	// Verify deletion
	_, err = backend.GetConfig(ctx, testKey)
	if err == nil {
		t.Error("GetConfig() should return error after deletion")
	}
}

func TestFileBackend_UsageOperations(t *testing.T) {
	backend, _, cleanup := setupTestFileBackend(t)
	defer cleanup()

	ctx := context.Background()
	testID := "test-usage-1"
	testStats := map[string]interface{}{
		"count": 10,
		"total": 100,
	}

	// Test SetUsageStats
	err := backend.SetUsageStats(ctx, testID, testStats)
	if err != nil {
		t.Errorf("SetUsageStats() error = %v", err)
	}

	// Test GetUsageStats
	retrieved, err := backend.GetUsageStats(ctx, testID)
	if err != nil {
		t.Errorf("GetUsageStats() error = %v", err)
	}

	if retrieved["count"] != float64(10) { // JSON unmarshals numbers as float64
		t.Errorf("GetUsageStats() count = %v, want 10", retrieved["count"])
	}

	// Test DeleteUsageStats
	err = backend.DeleteUsageStats(ctx, testID)
	if err != nil {
		t.Errorf("DeleteUsageStats() error = %v", err)
	}
}

func TestFileBackend_Health(t *testing.T) {
	backend, tmpDir, cleanup := setupTestFileBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Test healthy backend
	err := backend.Health(ctx)
	if err != nil {
		t.Errorf("Health() error = %v", err)
	}

	// Test unhealthy backend (remove directory)
	backend.Close()
	os.RemoveAll(tmpDir)

	err = backend.Health(ctx)
	if err == nil {
		t.Error("Health() should return error when directory is removed")
	}
}

func TestFileBackend_GetNotFound(t *testing.T) {
	backend, _, cleanup := setupTestFileBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Test GetCredential with non-existent ID
	_, err := backend.GetCredential(ctx, "non-existent")
	if err == nil {
		t.Error("GetCredential() should return error for non-existent ID")
	}

	// Test GetConfig with non-existent key
	_, err = backend.GetConfig(ctx, "non-existent")
	if err == nil {
		t.Error("GetConfig() should return error for non-existent key")
	}

	// Test GetUsageStats with non-existent ID
	_, err = backend.GetUsageStats(ctx, "non-existent")
	if err == nil {
		t.Error("GetUsageStats() should return error for non-existent ID")
	}
}

func TestFileBackend_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file-backend-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	testID := "test-persist"
	testData := map[string]interface{}{
		"id":    testID,
		"value": "test",
	}

	// Create backend and save data
	backend1 := NewFileBackend(tmpDir)
	if err := backend1.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if err := backend1.SetCredential(ctx, testID, testData); err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	if err := backend1.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Create new backend and verify data persisted
	backend2 := NewFileBackend(tmpDir)
	if err := backend2.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	defer backend2.Close()

	retrieved, err := backend2.GetCredential(ctx, testID)
	if err != nil {
		t.Errorf("GetCredential() error = %v", err)
	}

	if retrieved["id"] != testID {
		t.Errorf("GetCredential() id = %v, want %v", retrieved["id"], testID)
	}
}
