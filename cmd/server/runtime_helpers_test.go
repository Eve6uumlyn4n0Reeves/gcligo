package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gcli2api-go/internal/config"
	store "gcli2api-go/internal/storage"
)

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int64
	}{
		{"int", 42, 42},
		{"int32", int32(42), 42},
		{"int64", int64(42), 42},
		{"float64", float64(42.5), 42},
		{"float32", float32(42.5), 42},
		{"string", "42", 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toInt64(tt.input)
			if result != tt.expected {
				t.Errorf("toInt64(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "hello", "hello"},
		{"nil", nil, ""},
		{"int", 42, "42"},
		{"bool", true, "true"},
		{"map", map[string]string{"key": "value"}, `{"key":"value"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.input)
			if result != tt.expected {
				t.Errorf("toString(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultStorageDir(t *testing.T) {
	tests := []struct {
		name     string
		authDir  string
		expected string
	}{
		{
			name:     "Empty authDir",
			authDir:  "",
			expected: "./storage",
		},
		{
			name:     "AuthDir with auths",
			authDir:  "/path/to/auths",
			expected: "/path/to/storage",
		},
		{
			name:     "AuthDir without auths",
			authDir:  "/path/to/credentials",
			expected: "/path/to/credentials/../storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultStorageDir(tt.authDir)
			// Normalize paths for comparison
			expected := filepath.Clean(tt.expected)
			result = filepath.Clean(result)
			if result != expected {
				t.Errorf("defaultStorageDir(%q) = %q, want %q", tt.authDir, result, expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty path", "", ""},
		{"Absolute path", "/absolute/path", "/absolute/path"},
		{"Relative path", "relative/path", "relative/path"},
	}

	// Only test home expansion if we have a home directory
	if home != "" {
		tests = append(tests, struct {
			name     string
			input    string
			expected string
		}{
			"Home expansion",
			"~/test",
			filepath.Join(home, "test"),
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEnsureCredentialFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple ID", "cred1", "cred1.json"},
		{"ID with spaces", "my cred", "my-cred.json"},
		{"ID with dots", "cred..test", "credtest.json"},
		{"Already has .json", "cred.json", "cred.json"},
		{"Empty ID", "", "credential.json"},
		{"Whitespace only", "   ", "credential.json"},
		{"Uppercase", "CRED1", "cred1.json"},
		{"Mixed case", "MyCred", "mycred.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureCredentialFilename(tt.input)
			if result != tt.expected {
				t.Errorf("ensureCredentialFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildStorageBackend(t *testing.T) {
	ctx := context.Background()

	t.Run("File backend", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			StorageBackend: "file",
			StorageBaseDir: tmpDir,
		}

		backend, err := buildStorageBackend(ctx, cfg)
		if err != nil {
			t.Fatalf("buildStorageBackend() error = %v", err)
		}
		defer backend.Close()

		if _, ok := backend.(*store.FileBackend); !ok {
			t.Errorf("Expected FileBackend, got %T", backend)
		}
	})

	t.Run("Empty backend defaults to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			StorageBackend: "",
			StorageBaseDir: tmpDir,
		}

		backend, err := buildStorageBackend(ctx, cfg)
		if err != nil {
			t.Fatalf("buildStorageBackend() error = %v", err)
		}
		defer backend.Close()

		if _, ok := backend.(*store.FileBackend); !ok {
			t.Errorf("Expected FileBackend, got %T", backend)
		}
	})

	t.Run("Unsupported backend", func(t *testing.T) {
		cfg := &config.Config{
			StorageBackend: "unsupported",
		}

		_, err := buildStorageBackend(ctx, cfg)
		if err == nil {
			t.Error("Expected error for unsupported backend")
		}
	})
}

func TestMirrorCredentialsFromStorage(t *testing.T) {
	ctx := context.Background()

	t.Run("Nil backend returns false", func(t *testing.T) {
		changed, err := mirrorCredentialsFromStorage(ctx, nil, "/tmp/auth")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if changed {
			t.Error("Expected changed=false for nil backend")
		}
	})

	t.Run("FileBackend returns false", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		changed, err := mirrorCredentialsFromStorage(ctx, fb, "/tmp/auth")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if changed {
			t.Error("Expected changed=false for FileBackend")
		}
	})

	t.Run("Empty authDir returns false", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		changed, err := mirrorCredentialsFromStorage(ctx, fb, "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if changed {
			t.Error("Expected changed=false for empty authDir")
		}
	})
}

func TestPersistRoutingState(t *testing.T) {
	ctx := context.Background()

	t.Run("Nil strategy does nothing", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		// Should not panic
		persistRoutingState(ctx, fb, nil)
	})

	t.Run("Nil backend does nothing", func(t *testing.T) {
		// Should not panic
		persistRoutingState(ctx, nil, nil)
	})
}

func TestRestoreRoutingState(t *testing.T) {
	ctx := context.Background()

	t.Run("Nil strategy does nothing", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		// Should not panic
		restoreRoutingState(ctx, fb, nil)
	})

	t.Run("Nil backend does nothing", func(t *testing.T) {
		// Should not panic
		restoreRoutingState(ctx, nil, nil)
	})
}

func TestStartRoutingStatePersistence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Nil backend does nothing", func(t *testing.T) {
		done := make(chan bool)
		go func() {
			startRoutingStatePersistence(ctx, nil, nil, time.Second)
			done <- true
		}()

		cancel()
		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Error("Function did not return after context cancel")
		}
	})

	t.Run("Zero interval does nothing", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		done := make(chan bool)
		go func() {
			startRoutingStatePersistence(ctx, fb, nil, 0)
			done <- true
		}()

		select {
		case <-done:
			// Success - function returned immediately
		case <-time.After(100 * time.Millisecond):
			t.Error("Function did not return immediately for zero interval")
		}
	})
}
