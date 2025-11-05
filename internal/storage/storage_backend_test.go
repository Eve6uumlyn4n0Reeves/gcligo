package storage

import (
	"context"
	"testing"
)

// This test exercises BatchSet/BatchGet/BatchDelete path on the file backend (always available in CI)
func TestFileBackend_BatchAndExport(t *testing.T) {
	ctx := context.Background()
	fb := NewFileBackend(t.TempDir())
	if err := fb.Initialize(ctx); err != nil {
		t.Fatalf("init file backend: %v", err)
	}
	defer fb.Close()

	data := map[string]map[string]interface{}{
		"cred-a": {"id": "cred-a", "token": "A"},
		"cred-b": {"id": "cred-b", "token": "B"},
	}
	if err := fb.BatchSetCredentials(ctx, data); err != nil {
		t.Fatalf("batch set: %v", err)
	}

	got, err := fb.BatchGetCredentials(ctx, []string{"cred-a", "cred-b", "missing"})
	if err != nil {
		t.Fatalf("batch get: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 creds, got %d", len(got))
	}

	// Export should include credentials + configs + usage
	exp, err := fb.ExportData(ctx)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if _, ok := exp["credentials"]; !ok {
		t.Fatalf("export missing credentials key")
	}

	if err := fb.BatchDeleteCredentials(ctx, []string{"cred-a", "cred-b"}); err != nil {
		t.Fatalf("batch delete: %v", err)
	}
	if _, err := fb.GetCredential(ctx, "cred-a"); err == nil {
		t.Fatalf("expected cred-a deleted")
	}
}
