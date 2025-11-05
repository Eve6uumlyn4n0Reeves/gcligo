package adapter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StoreCredential 存储凭证
func (f *FileStorageAdapter) StoreCredential(ctx context.Context, cred *Credential) error {
	f.mu.Lock()
	shouldNotify := false
	defer func() {
		f.mu.Unlock()
		if shouldNotify {
			f.notifyWatchers()
		}
	}()

	now := time.Now()
	if cred.State == nil {
		cred.State = &CredentialState{
			ID:          cred.ID,
			HealthScore: 1.0,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}
	cred.State.UpdatedAt = now

	if cred.FilePath == "" {
		cred.FilePath = filepath.Join(f.credDir, cred.ID+".json")
	}

	if err := f.saveCredentialFile(cred); err != nil {
		return fmt.Errorf("failed to save credential file: %w", err)
	}

	if err := f.saveStateFile(cred.State); err != nil {
		return fmt.Errorf("failed to save state file: %w", err)
	}

	f.credentials[cred.ID] = cred
	f.states[cred.ID] = cred.State
	shouldNotify = true

	return nil
}

// LoadCredential 加载凭证
func (f *FileStorageAdapter) LoadCredential(ctx context.Context, id string) (*Credential, error) {
	f.mu.RLock()
	if cred, ok := f.credentials[id]; ok {
		f.mu.RUnlock()
		return cred, nil
	}
	f.mu.RUnlock()

	credFile := filepath.Join(f.credDir, id+".json")
	if _, err := os.Stat(credFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("credential not found: %s", id)
	}

	cred, err := f.loadCredentialFile(credFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load credential file: %w", err)
	}

	stateFile := filepath.Join(f.stateDir, id+".json")
	state, err := f.loadStateFile(stateFile)
	if err != nil {
		state = &CredentialState{
			ID:          cred.ID,
			HealthScore: 1.0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}
	cred.State = state

	f.mu.Lock()
	f.credentials[cred.ID] = cred
	f.states[cred.ID] = state
	f.mu.Unlock()

	return cred, nil
}

// DeleteCredential 删除凭证
func (f *FileStorageAdapter) DeleteCredential(ctx context.Context, id string) error {
	f.mu.Lock()
	defer func() {
		f.mu.Unlock()
		f.notifyWatchers()
	}()

	credFile := filepath.Join(f.credDir, id+".json")
	if err := os.Remove(credFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credential file: %w", err)
	}

	stateFile := filepath.Join(f.stateDir, id+".json")
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete state file: %w", err)
	}

	delete(f.credentials, id)
	delete(f.states, id)

	return nil
}

// UpdateCredential 更新凭证
func (f *FileStorageAdapter) UpdateCredential(ctx context.Context, cred *Credential) error {
	return f.StoreCredential(ctx, cred)
}

// GetAllCredentials 获取所有凭证
func (f *FileStorageAdapter) GetAllCredentials(ctx context.Context) ([]*Credential, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	credentials := make([]*Credential, 0, len(f.credentials))
	for _, cred := range f.credentials {
		credentials = append(credentials, cred)
	}
	return credentials, nil
}
