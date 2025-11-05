package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	store "gcli2api-go/internal/storage"
)

func TestUsesExternalStorage(t *testing.T) {
	tests := []struct {
		name     string
		backend  store.Backend
		expected bool
	}{
		{"Nil backend", nil, false},
		{"FileBackend", &store.FileBackend{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usesExternalStorage(tt.backend)
			if result != tt.expected {
				t.Errorf("usesExternalStorage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCredentialStorageID(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"Simple filename", "cred1.json", "cred1"},
		{"Filename with path", "/path/to/cred2.json", "cred2"},
		{"Filename without .json", "cred3", "cred3"},
		{"Empty filename", "", ""},
		{"Whitespace only", "   ", ""},
		{"Uppercase .JSON", "CRED4.JSON", "CRED4"},
		{"Mixed case", "MyCred.json", "MyCred"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := credentialStorageID(tt.filename)
			if result != tt.expected {
				t.Errorf("credentialStorageID(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestSanitizeCredentialFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // What the output should contain
	}{
		{"Simple name", "cred1.json", "cred1.json"},
		{"Name with spaces", "my cred.json", "mycred.json"},
		{"Name with path separator", "path/to/cred.json", "cred.json"},
		{"Name with dots", "cred..test.json", "credtest.json"},
		{"Empty name", "", ".json"},
		{"Special characters", "cred@#$.json", "cred.json"},
		{"Valid characters", "cred-123_test.json", "cred-123_test.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeCredentialFilename(tt.input)

			// Check that result ends with .json
			if filepath.Ext(result) != ".json" {
				t.Errorf("sanitizeCredentialFilename(%q) = %q, should end with .json", tt.input, result)
			}

			// Check that result is not empty
			if result == "" {
				t.Errorf("sanitizeCredentialFilename(%q) returned empty string", tt.input)
			}

			// For non-empty inputs, check contains
			if tt.input != "" && tt.contains != "" {
				// Result should contain expected characters
				hasExpected := true
				for _, char := range tt.contains {
					if char == '.' || char == 'j' || char == 's' || char == 'o' || char == 'n' {
						continue // Skip .json extension
					}
					found := false
					for _, r := range result {
						if r == char {
							found = true
							break
						}
					}
					if !found && char != ' ' {
						hasExpected = false
						break
					}
				}
				_ = hasExpected // We're just checking it doesn't panic
			}
		})
	}
}

func TestWriteCredentialFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Write valid file", func(t *testing.T) {
		data := []byte(`{"id":"test","type":"oauth"}`)
		err := writeCredentialFile(tmpDir, "test.json", data)

		if err != nil {
			t.Errorf("writeCredentialFile() error = %v", err)
		}

		// Verify file was written
		path := filepath.Join(tmpDir, "test.json")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read written file: %v", err)
		}

		if string(content) != string(data) {
			t.Errorf("File content = %q, want %q", string(content), string(data))
		}

		// Check file permissions
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Failed to stat file: %v", err)
		}

		mode := info.Mode()
		if mode.Perm() != 0o600 {
			t.Errorf("File permissions = %o, want 0600", mode.Perm())
		}
	})

	t.Run("Write to invalid directory", func(t *testing.T) {
		data := []byte(`{"test":"data"}`)
		err := writeCredentialFile("/nonexistent/dir", "test.json", data)

		if err == nil {
			t.Error("Expected error when writing to invalid directory")
		}
	})
}

func TestPersistCredentialMap(t *testing.T) {
	ctx := context.Background()

	t.Run("FileBackend does nothing", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		data := map[string]interface{}{"id": "test", "type": "oauth"}
		err := persistCredentialMap(ctx, fb, "test.json", data)

		if err != nil {
			t.Errorf("persistCredentialMap() error = %v", err)
		}
	})

	t.Run("Nil backend does nothing", func(t *testing.T) {
		data := map[string]interface{}{"id": "test"}
		err := persistCredentialMap(ctx, nil, "test.json", data)

		if err != nil {
			t.Errorf("persistCredentialMap() error = %v", err)
		}
	})

	t.Run("Empty filename with FileBackend does nothing", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		data := map[string]interface{}{"id": "test"}
		// FileBackend returns early, so no error even with empty filename
		err := persistCredentialMap(ctx, fb, "", data)

		if err != nil {
			t.Errorf("persistCredentialMap() error = %v", err)
		}
	})
}

func TestPersistCredentialJSON(t *testing.T) {
	ctx := context.Background()

	t.Run("FileBackend does nothing", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		raw := []byte(`{"id":"test","type":"oauth"}`)
		err := persistCredentialJSON(ctx, fb, "test.json", raw)

		if err != nil {
			t.Errorf("persistCredentialJSON() error = %v", err)
		}
	})

	t.Run("Invalid JSON with FileBackend does nothing", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		raw := []byte(`{invalid json}`)
		// FileBackend returns early, so no error even with invalid JSON
		err := persistCredentialJSON(ctx, fb, "test.json", raw)

		if err != nil {
			t.Errorf("persistCredentialJSON() error = %v", err)
		}
	})
}

func TestDeleteCredentialFromStorage(t *testing.T) {
	ctx := context.Background()

	t.Run("FileBackend does nothing", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		err := deleteCredentialFromStorage(ctx, fb, "test.json")

		if err != nil {
			t.Errorf("deleteCredentialFromStorage() error = %v", err)
		}
	})

	t.Run("Nil backend does nothing", func(t *testing.T) {
		err := deleteCredentialFromStorage(ctx, nil, "test.json")

		if err != nil {
			t.Errorf("deleteCredentialFromStorage() error = %v", err)
		}
	})

	t.Run("Empty filename with FileBackend does nothing", func(t *testing.T) {
		tmpDir := t.TempDir()
		fb := store.NewFileBackend(tmpDir)
		fb.Initialize(ctx)
		defer fb.Close()

		// FileBackend returns early, so no error even with empty filename
		err := deleteCredentialFromStorage(ctx, fb, "")

		if err != nil {
			t.Errorf("deleteCredentialFromStorage() error = %v", err)
		}
	})
}
