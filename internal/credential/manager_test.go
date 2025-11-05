package credential

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubStateStore struct {
	mu        sync.Mutex
	persisted map[string]*CredentialState
	deleted   []string
}

func newStubStateStore() *stubStateStore {
	return &stubStateStore{
		persisted: make(map[string]*CredentialState),
	}
}

func (s *stubStateStore) Persist(_ context.Context, cred *Credential, state *CredentialState) error {
	if cred == nil || state == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.persisted[cred.ID] = state
	return nil
}

func (s *stubStateStore) Restore(context.Context, *Credential) (*CredentialState, error) {
	return nil, nil
}

func (s *stubStateStore) Delete(_ context.Context, credID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, credID)
	return nil
}

func newTestManager(creds ...*Credential) *Manager {
	return &Manager{
		credentials:       creds,
		rotationThreshold: 100,
		credSource:        make(map[string]CredentialSource),
		lastPersist:       make(map[string]time.Time),
		sems:              make(map[string]chan struct{}),
		reloadCh:          make(chan struct{}, 1),
		stopRecovery:      make(chan struct{}),
		refreshAheadSec:   60,
		autoBan:           DefaultAutoBanConfig,
	}
}

func TestManagerDisableEnableCredential(t *testing.T) {
	store := newStubStateStore()
	cred := &Credential{ID: "cred-alpha"}
	mgr := newTestManager(cred)
	mgr.stateStore = store

	require.NoError(t, mgr.DisableCredential("cred-alpha"))
	require.True(t, cred.Disabled)
	require.Contains(t, store.persisted, "cred-alpha")

	require.NoError(t, mgr.EnableCredential("cred-alpha"))
	require.False(t, cred.Disabled)
	require.Zero(t, cred.FailureCount)
}

func TestManagerDeleteCredentialRemovesState(t *testing.T) {
	store := newStubStateStore()
	cred := &Credential{ID: "cred-delete"}
	mgr := newTestManager(cred)
	mgr.stateStore = store

	require.NoError(t, mgr.DeleteCredential("cred-delete"))
	require.Len(t, mgr.credentials, 0)

	store.mu.Lock()
	defer store.mu.Unlock()
	require.Contains(t, store.deleted, "cred-delete")
}

func TestManagerCleanupExpired(t *testing.T) {
	expired := &Credential{ID: "expired", Type: "oauth", ExpiresAt: time.Now().Add(-time.Hour)}
	noRefresh := &Credential{ID: "valid", Type: "oauth", ExpiresAt: time.Now().Add(time.Hour), RefreshToken: "token"}

	mgr := newTestManager(expired, noRefresh)
	mgr.CleanupExpired()

	require.Len(t, mgr.credentials, 1)
	require.Equal(t, "valid", mgr.credentials[0].ID)
}

func TestManagerResetAllStats(t *testing.T) {
	cred := &Credential{
		ID:               "stats",
		TotalRequests:    10,
		SuccessCount:     5,
		FailureCount:     5,
		ConsecutiveFails: 3,
		ErrorCodeCounts:  map[int]int{429: 2},
	}
	mgr := newTestManager(cred)

	mgr.ResetAllStats()

	require.Zero(t, cred.TotalRequests)
	require.Zero(t, cred.SuccessCount)
	require.Zero(t, cred.FailureCount)
	require.Zero(t, cred.ConsecutiveFails)
	require.Len(t, cred.ErrorCodeCounts, 0)
}
