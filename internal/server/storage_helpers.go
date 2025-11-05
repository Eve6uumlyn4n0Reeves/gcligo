package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	store "gcli2api-go/internal/storage"
)

func usesExternalStorage(backend store.Backend) bool {
	if backend == nil {
		return false
	}
	if _, ok := backend.(*store.FileBackend); ok {
		return false
	}
	return true
}

func credentialStorageID(filename string) string {
	name := strings.TrimSpace(filename)
	if name == "" {
		return ""
	}
	name = filepath.Base(name)
	if strings.HasSuffix(strings.ToLower(name), ".json") {
		name = name[:len(name)-5]
	}
	return strings.TrimSpace(name)
}

func persistCredentialMap(ctx context.Context, backend store.Backend, filename string, data map[string]any) error {
	if !usesExternalStorage(backend) {
		return nil
	}
	id := credentialStorageID(filename)
	if id == "" {
		return fmt.Errorf("invalid credential filename: %s", filename)
	}
	payload := make(map[string]any, len(data))
	for k, v := range data {
		payload[k] = v
	}
	return backend.SetCredential(ctx, id, payload)
}

func persistCredentialJSON(ctx context.Context, backend store.Backend, filename string, raw []byte) error {
	if !usesExternalStorage(backend) {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid credential payload: %w", err)
	}
	return persistCredentialMap(ctx, backend, filename, payload)
}

func deleteCredentialFromStorage(ctx context.Context, backend store.Backend, filename string) error {
	if !usesExternalStorage(backend) {
		return nil
	}
	id := credentialStorageID(filename)
	if id == "" {
		return fmt.Errorf("invalid credential filename: %s", filename)
	}
	if err := backend.DeleteCredential(ctx, id); err != nil {
		var nf *store.ErrNotFound
		if errors.As(err, &nf) {
			return nil
		}
		return err
	}
	return nil
}

func sanitizeCredentialFilename(name string) string {
	base := filepath.Base(name)
	base = strings.ReplaceAll(base, "..", "")
	base = strings.ReplaceAll(base, string(os.PathSeparator), "")
	base = strings.TrimSpace(base)
	b := make([]rune, 0, len(base))
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b = append(b, r)
		}
	}
	base = string(b)
	if base == "" {
		base = "credential-" + time.Now().Format("20060102-150405") + ".json"
	}
	if !strings.HasSuffix(strings.ToLower(base), ".json") {
		base += ".json"
	}
	return base
}

func writeCredentialFile(dir, name string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, name), data, 0o600)
}
