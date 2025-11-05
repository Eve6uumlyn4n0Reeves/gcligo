package storage

import (
	"context"
	"errors"
	"testing"
)

type stubExportBackend struct {
	ids      []string
	creds    map[string]map[string]interface{}
	configs  map[string]interface{}
	usage    map[string]map[string]interface{}
	listErr  error
	batchErr error
}

func (s *stubExportBackend) ListCredentials(context.Context) ([]string, error) {
	return s.ids, s.listErr
}

func (s *stubExportBackend) BatchGetCredentials(context.Context, []string) (map[string]map[string]interface{}, error) {
	return s.creds, s.batchErr
}

func (s *stubExportBackend) ListConfigs(context.Context) (map[string]interface{}, error) {
	return s.configs, nil
}

func (s *stubExportBackend) ListUsage(context.Context) (map[string]map[string]interface{}, error) {
	return s.usage, nil
}

type stubImportBackend struct {
	setCreds map[string]map[string]interface{}
	configs  map[string]interface{}
	setErr   error
}

func (s *stubImportBackend) BatchSetCredentials(_ context.Context, data map[string]map[string]interface{}) error {
	s.setCreds = data
	return s.setErr
}

func (s *stubImportBackend) SetConfig(_ context.Context, key string, value interface{}) error {
	if s.configs == nil {
		s.configs = make(map[string]interface{})
	}
	s.configs[key] = value
	return nil
}

type stubStatsBackend struct {
	ids    []string
	config map[string]interface{}
	usage  map[string]map[string]interface{}
	err    error
}

func (s *stubStatsBackend) ListCredentials(context.Context) ([]string, error) {
	return s.ids, s.err
}

func (s *stubStatsBackend) ListConfigs(context.Context) (map[string]interface{}, error) {
	return s.config, nil
}

func (s *stubStatsBackend) ListUsage(context.Context) (map[string]map[string]interface{}, error) {
	return s.usage, nil
}

func TestExportDataCommon(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	backend := &stubExportBackend{
		ids: []string{"a", "b"},
		creds: map[string]map[string]interface{}{
			"a": {"token": "secret"},
			"b": {"token": "secret2"},
		},
		configs: map[string]interface{}{"mode": "test"},
		usage:   map[string]map[string]interface{}{"a": {"count": 3}},
	}

	exported, err := exportDataCommon(ctx, "stub", backend)
	if err != nil {
		t.Fatalf("exportDataCommon error: %v", err)
	}
	if exported["backend"] != "stub" {
		t.Fatalf("expected backend stub, got %v", exported["backend"])
	}
	creds := exported["credentials"].(map[string]map[string]interface{})
	if len(creds) != 2 {
		t.Fatalf("expected 2 creds, got %d", len(creds))
	}
	if exported["configs"].(map[string]interface{})["mode"] != "test" {
		t.Fatalf("missing configs data")
	}
	usage := exported["usage"].(map[string]map[string]interface{})
	if len(usage) != 1 {
		t.Fatalf("expected usage data")
	}
}

func TestImportDataCommon(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	backend := &stubImportBackend{}

	payload := map[string]interface{}{
		"credentials": map[string]interface{}{
			"a": map[string]interface{}{"token": "secret"},
		},
		"configs": map[string]interface{}{"mode": "test"},
	}

	if err := importDataCommon(ctx, backend, payload); err != nil {
		t.Fatalf("importDataCommon error: %v", err)
	}
	if _, ok := backend.setCreds["a"]; !ok {
		t.Fatalf("expected credential a to be set")
	}
	if backend.configs["mode"] != "test" {
		t.Fatalf("expected config to be stored")
	}
}

func TestImportDataCommonHandlesErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	backend := &stubImportBackend{setErr: errors.New("boom")}

	payload := map[string]interface{}{
		"credentials": map[string]interface{}{
			"a": map[string]interface{}{"token": "secret"},
		},
	}

	err := importDataCommon(ctx, backend, payload)
	if err == nil {
		t.Fatalf("expected error when BatchSetCredentials fails")
	}
}

func TestStorageStatsCommon(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	backend := &stubStatsBackend{
		ids:    []string{"a", "b"},
		config: map[string]interface{}{"mode": "prod"},
		usage:  map[string]map[string]interface{}{"a": {"count": 1}},
	}

	stats, err := storageStatsCommon(ctx, "stub", backend)
	if err != nil {
		t.Fatalf("storageStatsCommon error: %v", err)
	}
	if stats.Backend != "stub" || !stats.Healthy {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if stats.CredentialCount != 2 || stats.ConfigCount != 1 || stats.UsageRecordCount != 1 {
		t.Fatalf("incorrect stats counts: %+v", stats)
	}
}
