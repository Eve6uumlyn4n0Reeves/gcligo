package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	store "gcli2api-go/internal/storage"
	route "gcli2api-go/internal/upstream/strategy"
	log "github.com/sirupsen/logrus"
)

func startRoutingStatePersistence(ctx context.Context, backend store.Backend, strategy *route.Strategy, interval time.Duration) {
	if backend == nil || strategy == nil || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			persistRoutingState(ctx, backend, strategy)
		case <-ctx.Done():
			return
		}
	}
}

func persistRoutingState(ctx context.Context, backend store.Backend, strategy *route.Strategy) {
	if strategy == nil || backend == nil {
		return
	}
	_, cds := strategy.Snapshot()
	payload := map[string]any{"cooldowns": cds}
	_ = backend.SetConfig(ctx, "routing_state", payload)
}

func restoreRoutingState(ctx context.Context, backend store.Backend, strategy *route.Strategy) {
	if strategy == nil || backend == nil {
		return
	}
	data, err := backend.GetConfig(ctx, "routing_state")
	if err != nil || data == nil {
		return
	}
	raw, ok := data.(map[string]any)
	if !ok {
		return
	}
	arr, _ := raw["cooldowns"].([]any)
	if len(arr) == 0 {
		return
	}
	for _, it := range arr {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		id, _ := m["credential_id"].(string)
		strikes := int(toInt64(m["strikes"]))
		if id != "" && strikes > 0 {
			strategy.SetCooldown(id, strikes, time.Now().Add(5*time.Second))
		}
	}
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	default:
		return 0
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func buildStorageBackend(ctx context.Context, cfg *config.Config) (store.Backend, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.StorageBackend))
	switch backend {
	case "", "file":
		baseDir := cfg.StorageBaseDir
		if baseDir == "" {
			baseDir = defaultStorageDir(cfg.AuthDir)
		}
		baseDir = expandPath(baseDir)
		fb := store.NewFileBackend(baseDir)
		if err := fb.Initialize(ctx); err != nil {
			return nil, err
		}
		return fb, nil
	case "redis":
		addr := cfg.RedisAddr
		if addr == "" {
			addr = "localhost:6379"
		}
		rb, err := store.NewRedisBackend(addr, cfg.RedisPassword, cfg.RedisDB, cfg.RedisPrefix)
		if err != nil {
			return nil, err
		}
		if err := rb.Initialize(ctx); err != nil {
			return nil, err
		}
		return rb, nil
	case "mongo", "mongodb":
		mb, err := store.NewMongoDBBackend(cfg.MongoURI, cfg.MongoDatabase)
		if err != nil {
			return nil, err
		}
		if err := mb.Initialize(ctx); err != nil {
			return nil, err
		}
		return mb, nil
	case "postgres", "postgresql":
		pb, err := store.NewPostgresBackend(cfg.PostgresDSN)
		if err != nil {
			return nil, err
		}
		if err := pb.Initialize(ctx); err != nil {
			return nil, err
		}
		return pb, nil
	case "git":
		gb := store.NewGitBackendFromConfig(cfg)
		if err := gb.Initialize(ctx); err != nil {
			return nil, err
		}
		return gb, nil
	case "auto":
		if cfg.RedisAddr != "" {
			if rb, err := store.NewRedisBackend(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.RedisPrefix); err == nil {
				if err := rb.Initialize(ctx); err == nil {
					log.Info("storage auto: using redis backend")
					return rb, nil
				}
			}
			log.Warn("storage auto: redis backend initialization failed, falling back")
		}
		if cfg.PostgresDSN != "" {
			if pb, err := store.NewPostgresBackend(cfg.PostgresDSN); err == nil {
				if err := pb.Initialize(ctx); err == nil {
					log.Info("storage auto: using postgres backend")
					return pb, nil
				}
			}
			log.Warn("storage auto: postgres backend initialization failed, falling back")
		}
		if cfg.MongoURI != "" {
			if mb, err := store.NewMongoDBBackend(cfg.MongoURI, cfg.MongoDatabase); err == nil {
				if err := mb.Initialize(ctx); err == nil {
					log.Info("storage auto: using mongodb backend")
					return mb, nil
				}
			}
			log.Warn("storage auto: mongodb backend initialization failed, falling back")
		}
		baseDir := defaultStorageDir(cfg.AuthDir)
		fb := store.NewFileBackend(expandPath(baseDir))
		if err := fb.Initialize(ctx); err != nil {
			return nil, err
		}
		log.Info("storage auto: using local file backend")
		return fb, nil
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", backend)
	}
}

func defaultStorageDir(authDir string) string {
	if authDir == "" {
		return "./storage"
	}
	expanded := expandPath(authDir)
	clean := filepath.Clean(expanded)
	if filepath.Base(clean) == "auths" {
		return filepath.Join(filepath.Dir(clean), "storage")
	}
	return filepath.Join(clean, "..", "storage")
}

func expandPath(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}

func mirrorCredentialsFromStorage(ctx context.Context, backend store.Backend, authDir string) (bool, error) {
	if backend == nil {
		return false, nil
	}
	if _, ok := backend.(*store.FileBackend); ok {
		return false, nil
	}
	dir := strings.TrimSpace(authDir)
	if dir == "" {
		return false, nil
	}
	expanded := expandPath(dir)
	if err := os.MkdirAll(expanded, 0o700); err != nil {
		return false, err
	}
	ids, err := backend.ListCredentials(ctx)
	if err != nil {
		return false, err
	}
	desired := make(map[string]struct{}, len(ids))
	changed := false

	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			continue
		}
		data, err := backend.GetCredential(ctx, id)
		if err != nil {
			return false, err
		}
		payload, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return false, err
		}
		filename := ensureCredentialFilename(id)
		desired[filename] = struct{}{}
		path := filepath.Join(expanded, filename)
		if existing, err := os.ReadFile(path); err == nil {
			if bytes.Equal(bytes.TrimSpace(existing), bytes.TrimSpace(payload)) {
				continue
			}
		} else if !os.IsNotExist(err) {
			return false, err
		}
		if err := os.WriteFile(path, payload, 0o600); err != nil {
			return changed, err
		}
		changed = true
	}
	if entries, err := os.ReadDir(expanded); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if _, keep := desired[name]; keep {
				continue
			}
			low := strings.ToLower(name)
			if !strings.HasSuffix(low, ".json") || strings.HasSuffix(low, ".state.json") {
				continue
			}
			if err := os.Remove(filepath.Join(expanded, name)); err != nil && !os.IsNotExist(err) {
				return changed, err
			}
			changed = true
		}
	} else {
		return changed, err
	}
	return changed, nil
}

func startStorageMirror(ctx context.Context, backend store.Backend, authDir string, mgr *credential.Manager) {
	if backend == nil || mgr == nil {
		return
	}
	if _, ok := backend.(*store.FileBackend); ok {
		return
	}
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, err := mirrorCredentialsFromStorage(ctx, backend, authDir)
			if err != nil {
				log.Warnf("mirror credentials from storage: %v", err)
				continue
			}
			if changed {
				if err := mgr.LoadCredentials(); err != nil {
					log.Warnf("reload credentials after storage mirror: %v", err)
				} else {
					log.Info("mirrored credentials from storage backend")
				}
			}
		}
	}
}

func ensureCredentialFilename(id string) string {
	clean := strings.TrimSpace(strings.ToLower(id))
	if clean == "" {
		clean = "credential"
	}
	clean = strings.ReplaceAll(clean, " ", "-")
	clean = strings.ReplaceAll(clean, "..", "")
	if !strings.HasSuffix(clean, ".json") {
		clean += ".json"
	}
	return clean
}
