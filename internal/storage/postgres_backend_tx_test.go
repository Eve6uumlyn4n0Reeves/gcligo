package storage

import (
	"context"
	"testing"
)

func TestPostgresTransactionEnsureOpenFailsWhenClosed(t *testing.T) {
	t.Parallel()
	tr := &postgresTransaction{closed: true}
	if _, err := tr.GetCredential(context.Background(), "cred"); err == nil {
		t.Fatalf("expected ensureOpen error when transaction closed")
	}
	if err := tr.SetCredential(context.Background(), "cred", map[string]interface{}{}); err == nil {
		t.Fatalf("expected ensureOpen error for SetCredential")
	}
	if err := tr.DeleteCredential(context.Background(), "cred"); err == nil {
		t.Fatalf("expected ensureOpen error for DeleteCredential")
	}
	if _, err := tr.GetConfig(context.Background(), "key"); err == nil {
		t.Fatalf("expected ensureOpen error for GetConfig")
	}
	if err := tr.SetConfig(context.Background(), "key", "value"); err == nil {
		t.Fatalf("expected ensureOpen error for SetConfig")
	}
	if err := tr.DeleteConfig(context.Background(), "key"); err == nil {
		t.Fatalf("expected ensureOpen error for DeleteConfig")
	}
}

func TestPostgresTransactionCommitAndRollbackNoopOnNil(t *testing.T) {
	t.Parallel()
	tr := &postgresTransaction{}
	if err := tr.Commit(context.Background()); err != nil {
		t.Fatalf("expected nil commit error, got %v", err)
	}
	if err := tr.Rollback(context.Background()); err != nil {
		t.Fatalf("expected nil rollback error, got %v", err)
	}

	tr.closed = true
	if err := tr.Commit(context.Background()); err != nil {
		t.Fatalf("expected nil commit error when closed, got %v", err)
	}
	if err := tr.Rollback(context.Background()); err != nil {
		t.Fatalf("expected nil rollback error when closed, got %v", err)
	}
}
