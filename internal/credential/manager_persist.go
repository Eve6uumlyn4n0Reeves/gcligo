package credential

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

func (m *Manager) restoreCredentialStateLegacy(cred *Credential) {
	if cred == nil {
		return
	}
	path := m.stateFilePath(cred.ID)
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var state CredentialState
	if err := json.Unmarshal(data, &state); err != nil {
		log.WithError(err).Warnf("credential manager: failed to parse state for %s", cred.ID)
		return
	}
	cred.RestoreState(&state)
}

// restoreCredentialState chooses store or legacy file
func (m *Manager) restoreCredentialState(cred *Credential) {
	if m.stateStore == nil {
		m.restoreCredentialStateLegacy(cred)
		return
	}
	st, err := m.stateStore.Restore(context.Background(), cred)
	if err != nil || st == nil {
		return
	}
	cred.RestoreState(st)
}

func (m *Manager) persistCredentialState(cred *Credential, force bool) {
	if cred == nil {
		return
	}
	state := cred.SnapshotState()
	if state == nil {
		return
	}
	if !m.markPersistAttempt(cred.ID, force) {
		return
	}
	if m.stateStore != nil {
		_ = m.stateStore.Persist(context.Background(), cred, state)
		return
	}
	if src := m.getCredentialSource(cred.ID); src != nil {
		if stateful, ok := src.(StatefulCredentialSource); ok {
			if err := stateful.PersistState(context.Background(), cred, state); err != nil {
				log.WithError(err).Warnf("credential manager: state persist via %s failed for %s", src.Name(), cred.ID)
			}
			return
		}
	}
	m.persistCredentialStateLegacy(cred, state)
}

func (m *Manager) markPersistAttempt(id string, force bool) bool {
	now := time.Now()
	m.persistMu.Lock()
	defer m.persistMu.Unlock()
	if !force {
		if last := m.lastPersist[id]; !last.IsZero() && now.Sub(last) < statePersistInterval {
			return false
		}
	}
	m.lastPersist[id] = now
	return true
}

func (m *Manager) persistCredentialStateLegacy(cred *Credential, state *CredentialState) {
	if cred == nil || state == nil {
		return
	}
	path := m.stateFilePath(cred.ID)
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		log.WithError(err).Warn("credential manager: failed to prepare state directory")
		return
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.WithError(err).Warnf("credential manager: failed to marshal state for %s", cred.ID)
		return
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		log.WithError(err).Warnf("credential manager: failed to write state tmp for %s", cred.ID)
		return
	}
	if err := os.Rename(tmpPath, path); err != nil {
		log.WithError(err).Warnf("credential manager: failed to persist state for %s", cred.ID)
	}
}

func (m *Manager) deleteCredentialState(credID string) {
	if credID == "" {
		return
	}
	if m.stateStore != nil {
		_ = m.stateStore.Delete(context.Background(), credID)
		goto done
	}
	if src := m.getCredentialSource(credID); src != nil {
		if stateful, ok := src.(StatefulCredentialSource); ok {
			if err := stateful.DeleteState(context.Background(), credID); err != nil {
				log.WithError(err).Warnf("credential manager: state delete via %s failed for %s", src.Name(), credID)
			}
		} else {
			m.deleteCredentialStateLegacy(credID)
		}
	} else {
		m.deleteCredentialStateLegacy(credID)
	}
done:
	m.persistMu.Lock()
	delete(m.lastPersist, credID)
	m.persistMu.Unlock()
}

func (m *Manager) deleteCredentialStateLegacy(credID string) {
	path := m.stateFilePath(credID)
	if path == "" {
		return
	}
	_ = os.Remove(path)
}

func (m *Manager) deleteCredentialLegacy(credID string) error {
	if m.authDir == "" {
		return nil
	}
	path := filepath.Join(m.authDir, ensureJSONExtension(credID))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credential file: %w", err)
	}
	return nil
}
