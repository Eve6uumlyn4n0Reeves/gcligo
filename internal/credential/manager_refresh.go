package credential

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gcli2api-go/internal/oauth"
	log "github.com/sirupsen/logrus"
)

// RefreshCredential refreshes an OAuth token for the given credential ID.
func (m *Manager) RefreshCredential(ctx context.Context, credID string) error {
	if m.refreshCoord != nil {
		return m.refreshCoord.Do(ctx, credID, func(ctx context.Context) error { return m.refreshCredentialCore(ctx, credID) })
	}
	return m.refreshCredentialCore(ctx, credID)
}

func (m *Manager) refreshCredentialCore(ctx context.Context, credID string) error {
	m.mu.RLock()
	var target *Credential
	for _, cred := range m.credentials {
		if cred != nil && cred.ID == credID && cred.Type == "oauth" {
			target = cred
			break
		}
	}
	m.mu.RUnlock()
	if target == nil {
		return fmt.Errorf("credential %s not found or not OAuth type", credID)
	}

	target.mu.RLock()
	refreshToken := target.RefreshToken
	clientID := target.ClientID
	clientSecret := target.ClientSecret
	if refreshToken == "" || clientID == "" || clientSecret == "" {
		target.mu.RUnlock()
		return fmt.Errorf("credential %s missing refresh prerequisites", credID)
	}
	target.mu.RUnlock()

	om := oauth.NewManager(clientID, clientSecret, "")
	oc := &oauth.Credentials{RefreshToken: refreshToken}
	if err := om.RefreshToken(ctx, oc); err != nil {
		return fmt.Errorf("refresh failed: %w", err)
	}

	target.mu.Lock()
	target.AccessToken = oc.AccessToken
	if oc.RefreshToken != "" {
		target.RefreshToken = oc.RefreshToken
	}
	if !oc.ExpiresAt.IsZero() {
		target.ExpiresAt = oc.ExpiresAt
	}
	target.mu.Unlock()

	if err := m.saveCredential(target.Clone()); err != nil {
		log.Warnf("Failed to persist refreshed token for %s: %v", credID, err)
	}
	log.Infof("Refreshed OAuth token for %s", credID)
	return nil
}

// saveCredential persists a credential via its source (fallback to legacy file write).
func (m *Manager) saveCredential(cred *Credential) error {
	if cred == nil {
		return fmt.Errorf("credential is nil")
	}
	ctx := context.Background()
	if src := m.getCredentialSource(cred.ID); src != nil {
		if writable, ok := src.(WritableCredentialSource); ok {
			return writable.Save(ctx, cred)
		}
	}
	for _, src := range m.sources {
		writable, ok := src.(WritableCredentialSource)
		if !ok {
			continue
		}
		if err := writable.Save(ctx, cred); err != nil {
			return err
		}
		if cred.Source == "" {
			cred.Source = src.Name()
		}
		m.mu.Lock()
		if m.credSource == nil {
			m.credSource = make(map[string]CredentialSource)
		}
		m.credSource[cred.ID] = src
		m.mu.Unlock()
		return nil
	}
	return m.saveCredentialLegacy(cred)
}

func (m *Manager) saveCredentialLegacy(cred *Credential) error {
	if m.authDir == "" {
		return fmt.Errorf("auth directory not configured")
	}
	data, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(m.authDir, cred.ID)
	if filepath.Ext(path) == "" {
		path += ".json"
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	cred.Source = "file:" + filepath.Clean(m.authDir)
	m.mu.Lock()
	if m.credSource == nil {
		m.credSource = make(map[string]CredentialSource)
	}
	if fileSrc := m.findFileSource(); fileSrc != nil {
		m.credSource[cred.ID] = fileSrc
	}
	m.mu.Unlock()
	return nil
}

// StartPeriodicRefresh starts a goroutine that periodically refreshes OAuth tokens.
func (m *Manager) StartPeriodicRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.refreshExpiredTokens(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) refreshExpiredTokens(ctx context.Context) {
	creds := m.GetAllCredentials()

	for _, cred := range creds {
		if cred.Type == "oauth" && m.shouldRefresh(cred) && cred.RefreshToken != "" {
			if err := m.RefreshCredential(ctx, cred.ID); err != nil {
				log.Errorf("Failed to refresh credential %s: %v", cred.ID, err)
			}
		}
	}
}

// shouldRefresh determines if a credential should be proactively refreshed based on ExpiresAt and policy window.
func (m *Manager) shouldRefresh(cred *Credential) bool {
	if cred == nil || cred.Type != "oauth" {
		return false
	}
	if cred.RefreshToken == "" {
		return false
	}
	if cred.AccessToken == "" {
		return true
	}
	if cred.ExpiresAt.IsZero() {
		return true
	}
	ahead := time.Duration(m.refreshAheadSec) * time.Second
	// If expiry already passed or within ahead window, refresh.
	return time.Until(cred.ExpiresAt) <= ahead
}
