package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gcli2api-go/internal/config"
	store "gcli2api-go/internal/storage"
)

func main() {
	mode := flag.String("mode", "", "operation mode: export | import | verify | plan-audit")
	filePath := flag.String("file", "", "file path for export/import/verify (default: stdout/stdin)")
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	timeout := flag.Duration("timeout", 30*time.Second, "operation timeout")
	flag.Parse()

	if *mode == "" {
		fail(fmt.Errorf("missing -mode (export|import|verify)"))
	}

	cfg := config.LoadWithFile(*configPath)
	if cfg == nil {
		fail(errors.New("failed to load configuration"))
	}
	if err := cfg.ValidateAndExpandPaths(); err != nil {
		fail(fmt.Errorf("invalid configuration paths: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	backend, err := buildStorageBackend(ctx, cfg)
	if err != nil {
		fail(fmt.Errorf("build storage backend: %w", err))
	}
	defer backend.Close()

	switch strings.ToLower(*mode) {
	case "export":
		if err := runExport(ctx, backend, *filePath); err != nil {
			fail(err)
		}
	case "import":
		if err := runImport(ctx, backend, *filePath); err != nil {
			fail(err)
		}
	case "verify":
		matches, err := runVerify(ctx, backend, *filePath)
		if err != nil {
			fail(err)
		}
		if !matches {
			os.Exit(1)
		}
	case "plan-audit":
		if err := runPlanAudit(ctx, backend, *filePath); err != nil {
			fail(err)
		}
	default:
		fail(fmt.Errorf("unknown mode %q (expected export|import|verify)", *mode))
	}
}

func runExport(ctx context.Context, backend store.Backend, path string) error {
	data, err := backend.ExportData(ctx)
	if err != nil {
		return fmt.Errorf("export data: %w", err)
	}
	var w io.Writer = os.Stdout
	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create export directory: %w", err)
		}
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("open export file: %w", err)
		}
		defer f.Close()
		w = f
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("write export json: %w", err)
	}
	return nil
}

func runImport(ctx context.Context, backend store.Backend, path string) error {
	payload, err := readJSON(path)
	if err != nil {
		return fmt.Errorf("read import json: %w", err)
	}
	if err := backend.ImportData(ctx, payload); err != nil {
		return fmt.Errorf("import data: %w", err)
	}
	return nil
}

func runVerify(ctx context.Context, backend store.Backend, path string) (bool, error) {
	expected, err := readJSON(path)
	if err != nil {
		return false, fmt.Errorf("read reference json: %w", err)
	}
	current, err := backend.ExportData(ctx)
	if err != nil {
		return false, fmt.Errorf("export current data: %w", err)
	}
	if deepEqualJSON(expected, current) {
		fmt.Println("storage matches reference snapshot")
		return true, nil
	}
	fmt.Println("storage diverges from reference snapshot")
	return false, nil
}

func runPlanAudit(ctx context.Context, backend store.Backend, path string) error {
	exporter, ok := backend.(store.PlanAuditExporter)
	if !ok {
		return fmt.Errorf("plan audit not supported for backend %T", backend)
	}
	entries, err := exporter.ExportPlanAudit(ctx)
	if err != nil {
		return fmt.Errorf("export plan audit: %w", err)
	}

	var w io.Writer = os.Stdout
	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create audit directory: %w", err)
		}
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("open audit file: %w", err)
		}
		defer f.Close()
		w = f
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

func readJSON(path string) (map[string]any, error) {
	var r io.Reader = os.Stdin
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}
	var payload map[string]any
	dec := json.NewDecoder(r)
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func deepEqualJSON(a, b map[string]any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "storageutil:", err)
	os.Exit(1)
}

func buildStorageBackend(ctx context.Context, cfg *config.Config) (store.Backend, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.StorageBackend))
	if backend == "" {
		backend = "file"
	}

	switch backend {
	case "file":
		baseDir := cfg.StorageBaseDir
		if baseDir == "" {
			baseDir = defaultStorageDir(cfg.AuthDir)
		}
		fb := store.NewFileBackend(expandPath(baseDir))
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
	case "mongodb", "mongo":
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
					return rb, nil
				}
			}
		}
		if cfg.PostgresDSN != "" {
			if pb, err := store.NewPostgresBackend(cfg.PostgresDSN); err == nil {
				if err := pb.Initialize(ctx); err == nil {
					return pb, nil
				}
			}
		}
		if cfg.MongoURI != "" {
			if mb, err := store.NewMongoDBBackend(cfg.MongoURI, cfg.MongoDatabase); err == nil {
				if err := mb.Initialize(ctx); err == nil {
					return mb, nil
				}
			}
		}
		fallthrough
	default:
		baseDir := cfg.StorageBaseDir
		if baseDir == "" {
			baseDir = defaultStorageDir(cfg.AuthDir)
		}
		fb := store.NewFileBackend(expandPath(baseDir))
		if err := fb.Initialize(ctx); err != nil {
			return nil, err
		}
		return fb, nil
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
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}
