package usage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStorage_SaveAndLoad(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "usage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	ctx := context.Background()

	// Create test stats
	stats := NewStats()
	stats.TotalRequests = 100
	stats.SuccessCount = 90
	stats.FailureCount = 10
	stats.TotalTokens = 5000

	// Add credential usage
	cred := NewCredentialUsage("test-cred-1")
	cred.TotalCalls = 50
	cred.TotalTokens = 2500
	cred.ModelBreakdown["gemini-2.0-flash-exp"] = &ModelStats{
		ModelName: "gemini-2.0-flash-exp",
		Calls:     50,
		Tokens:    2500,
	}
	stats.Credentials["test-cred-1"] = cred

	// Save stats
	if err := storage.SaveStats(ctx, stats); err != nil {
		t.Fatalf("Failed to save stats: %v", err)
	}

	// Load stats
	loaded, err := storage.LoadStats(ctx)
	if err != nil {
		t.Fatalf("Failed to load stats: %v", err)
	}

	// Verify
	if loaded.TotalRequests != stats.TotalRequests {
		t.Errorf("Expected TotalRequests=%d, got %d", stats.TotalRequests, loaded.TotalRequests)
	}
	if loaded.SuccessCount != stats.SuccessCount {
		t.Errorf("Expected SuccessCount=%d, got %d", stats.SuccessCount, loaded.SuccessCount)
	}
	if loaded.TotalTokens != stats.TotalTokens {
		t.Errorf("Expected TotalTokens=%d, got %d", stats.TotalTokens, loaded.TotalTokens)
	}

	// Verify credential
	loadedCred, ok := loaded.Credentials["test-cred-1"]
	if !ok {
		t.Fatal("Credential not found")
	}
	if loadedCred.TotalCalls != cred.TotalCalls {
		t.Errorf("Expected credential TotalCalls=%d, got %d", cred.TotalCalls, loadedCred.TotalCalls)
	}
}

func TestFileStorage_LoadNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "usage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	ctx := context.Background()

	// Load non-existent stats should return empty stats
	stats, err := storage.LoadStats(ctx)
	if err != nil {
		t.Fatalf("Failed to load stats: %v", err)
	}
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}
	if stats.TotalRequests != 0 {
		t.Errorf("Expected TotalRequests=0, got %d", stats.TotalRequests)
	}
}

func TestFileStorage_CredentialUsage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "usage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	ctx := context.Background()

	// Create credential usage
	usage := NewCredentialUsage("test-cred-2")
	usage.TotalCalls = 100
	usage.TotalTokens = 5000
	usage.LastUsed = time.Now()
	usage.ModelBreakdown["gemini-2.0-flash-exp"] = &ModelStats{
		ModelName: "gemini-2.0-flash-exp",
		Calls:     100,
		Tokens:    5000,
	}

	// Save
	if err := storage.SaveCredentialUsage(ctx, usage); err != nil {
		t.Fatalf("Failed to save credential usage: %v", err)
	}

	// Load
	loaded, err := storage.LoadCredentialUsage(ctx, "test-cred-2")
	if err != nil {
		t.Fatalf("Failed to load credential usage: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected non-nil credential usage")
	}
	if loaded.TotalCalls != usage.TotalCalls {
		t.Errorf("Expected TotalCalls=%d, got %d", usage.TotalCalls, loaded.TotalCalls)
	}
	if loaded.TotalTokens != usage.TotalTokens {
		t.Errorf("Expected TotalTokens=%d, got %d", usage.TotalTokens, loaded.TotalTokens)
	}

	// Load non-existent
	nonExistent, err := storage.LoadCredentialUsage(ctx, "non-existent")
	if err != nil {
		t.Fatalf("Failed to load non-existent credential: %v", err)
	}
	if nonExistent != nil {
		t.Error("Expected nil for non-existent credential")
	}
}

func TestFileStorage_AtomicWrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "usage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	ctx := context.Background()

	// Save stats multiple times
	for i := 0; i < 10; i++ {
		stats := NewStats()
		stats.TotalRequests = int64(i * 10)
		if err := storage.SaveStats(ctx, stats); err != nil {
			t.Fatalf("Failed to save stats iteration %d: %v", i, err)
		}
	}

	// Verify final state
	loaded, err := storage.LoadStats(ctx)
	if err != nil {
		t.Fatalf("Failed to load stats: %v", err)
	}
	if loaded.TotalRequests != 90 {
		t.Errorf("Expected TotalRequests=90, got %d", loaded.TotalRequests)
	}

	// Verify no temp files left
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read dir: %v", err)
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".tmp" {
			t.Errorf("Found temp file: %s", file.Name())
		}
	}
}

