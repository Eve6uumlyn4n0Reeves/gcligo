package credential

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// StateStore abstracts persistence of per-credential state (ban/cooldown/counters).
type StateStore interface {
	Persist(ctx context.Context, cred *Credential, state *CredentialState) error
	Restore(ctx context.Context, cred *Credential) (*CredentialState, error)
	Delete(ctx context.Context, credID string) error
}

// FileStateStore is a simple file-based state store compatible with legacy layout.
type FileStateStore struct{ Dir string }

func (f *FileStateStore) path(id string) string {
	if f == nil || f.Dir == "" || id == "" {
		return ""
	}
	base := strings.TrimSuffix(id, filepath.Ext(id))
	return filepath.Join(f.Dir, base+credentialStateSuffix)
}

func (f *FileStateStore) Persist(_ context.Context, cred *Credential, state *CredentialState) error {
	if cred == nil || state == nil {
		return nil
	}
	p := f.path(cred.ID)
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (f *FileStateStore) Restore(_ context.Context, cred *Credential) (*CredentialState, error) {
	if cred == nil {
		return nil, nil
	}
	p := f.path(cred.ID)
	if p == "" {
		return nil, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var st CredentialState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (f *FileStateStore) Delete(_ context.Context, credID string) error {
	p := f.path(credID)
	if p == "" {
		return nil
	}
	_ = os.Remove(p)
	return nil
}
