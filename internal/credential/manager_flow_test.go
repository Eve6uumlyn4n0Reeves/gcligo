package credential

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManagerGetCredentialPrefersHealthy(t *testing.T) {
	healthy := &Credential{
		ID:            "healthy",
		TotalRequests: 20,
		SuccessCount:  18,
		ErrorCodeCounts: map[int]int{
			200: 20,
		},
	}
	unhealthy := &Credential{
		ID:         "unhealthy",
		AutoBanned: true,
	}
	mgr := newTestManager(unhealthy, healthy)
	mgr.currentIndex = 0

	cred, err := mgr.GetCredential()
	require.NoError(t, err)
	require.Equal(t, "healthy", cred.ID)
}

func TestManagerGetAlternateCredentialSkipsExcluded(t *testing.T) {
	exclude := &Credential{ID: "primary", TotalRequests: 5, SuccessCount: 5}
	disabled := &Credential{ID: "disabled", Disabled: true}
	alt := &Credential{ID: "backup", TotalRequests: 10, SuccessCount: 9}

	mgr := newTestManager(exclude, disabled, alt)
	mgr.currentIndex = 0

	cred, err := mgr.GetAlternateCredential("primary")
	require.NoError(t, err)
	require.Equal(t, "backup", cred.ID)
}

func TestManagerMarkFailureTriggersAutoBan(t *testing.T) {
	store := newStubStateStore()
	cred := &Credential{ID: "cred-ban", ErrorCodeCounts: make(map[int]int)}
	mgr := newTestManager(cred)
	mgr.stateStore = store
	mgr.autoBan.Threshold429 = 2

	mgr.MarkFailure("cred-ban", "rate limit", 429)
	require.False(t, cred.AutoBanned)

	mgr.MarkFailure("cred-ban", "rate limit", 429)
	require.True(t, cred.AutoBanned)

	store.mu.Lock()
	_, ok := store.persisted["cred-ban"]
	store.mu.Unlock()
	require.True(t, ok, "state should be persisted after failure")
}

func TestManagerMarkSuccessPersistsState(t *testing.T) {
	store := newStubStateStore()
	cred := &Credential{ID: "cred-success", ErrorCodeCounts: make(map[int]int)}
	mgr := newTestManager(cred)
	mgr.stateStore = store

	mgr.MarkSuccess("cred-success")
	require.Equal(t, int64(1), cred.TotalRequests)
	require.Equal(t, int64(1), cred.SuccessCount)

	store.mu.Lock()
	_, ok := store.persisted["cred-success"]
	store.mu.Unlock()
	require.True(t, ok, "state should be persisted after success")
}

func TestManagerConcurrencyControls(t *testing.T) {
	cred := &Credential{ID: "cred-concurrency"}
	mgr := newTestManager(cred)
	mgr.maxConcPerCred = 1

	require.True(t, mgr.HasCapacity("cred-concurrency"))

	release := mgr.Acquire("cred-concurrency")
	require.False(t, mgr.HasCapacity("cred-concurrency"))

	release()
	require.True(t, mgr.HasCapacity("cred-concurrency"))
}

func TestFindBestCredentialUsesHighestScore(t *testing.T) {
	now := time.Now()
	lo := &Credential{
		ID:            "low",
		TotalRequests: 10,
		SuccessCount:  3,
		LastFailure:   now,
		ErrorCodeCounts: map[int]int{
			429: 2,
		},
	}
	hi := &Credential{
		ID:            "high",
		TotalRequests: 20,
		SuccessCount:  18,
		LastSuccess:   now,
		ErrorCodeCounts: map[int]int{
			200: 20,
		},
	}
	mgr := newTestManager(lo, hi)

	best := mgr.findBestCredential()
	require.NotNil(t, best)
	require.Equal(t, "high", best.ID)
}
