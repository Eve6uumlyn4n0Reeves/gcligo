package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (f *FileStorageAdapter) saveCredentialFile(cred *Credential) error {
	if cred.FilePath == "" {
		cred.FilePath = filepath.Join(f.credDir, cred.ID+".json")
	}

	stored := *cred
	stored.State = nil

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cred.FilePath, data, 0o600)
}

func (f *FileStorageAdapter) loadCredentialFile(path string) (*Credential, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cred Credential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, err
	}
	if cred.ID == "" {
		cred.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	cred.FilePath = path

	return &cred, nil
}

func (f *FileStorageAdapter) saveStateFile(state *CredentialState) error {
	if state == nil {
		return nil
	}

	stateFile := filepath.Join(f.stateDir, state.ID+".json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0o600)
}

func (f *FileStorageAdapter) loadStateFile(path string) (*CredentialState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state CredentialState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if state.ID == "" {
		state.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	return &state, nil
}

func (f *FileStorageAdapter) loadAllCredentials() error {
	entries, err := os.ReadDir(f.credDir)
	if err != nil {
		return err
	}

	newCreds := make(map[string]*Credential)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(f.credDir, entry.Name())
		cred, err := f.loadCredentialFile(path)
		if err != nil {
			continue
		}
		newCreds[cred.ID] = cred
	}

	f.mu.Lock()
	for id, cred := range newCreds {
		if state, ok := f.states[id]; ok {
			cred.State = state
		}
		newCreds[id] = cred
	}
	f.credentials = newCreds
	f.mu.Unlock()

	return nil
}

func (f *FileStorageAdapter) loadAllStates() error {
	entries, err := os.ReadDir(f.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	newStates := make(map[string]*CredentialState)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(f.stateDir, entry.Name())
		state, err := f.loadStateFile(path)
		if err != nil {
			continue
		}
		if state.CreatedAt.IsZero() {
			state.CreatedAt = time.Now()
		}
		newStates[state.ID] = state
	}

	f.mu.Lock()
	f.states = newStates
	for id, cred := range f.credentials {
		if state, ok := newStates[id]; ok {
			cred.State = state
			f.credentials[id] = cred
		}
	}
	f.mu.Unlock()

	return nil
}
