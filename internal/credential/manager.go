package credential

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"gcli2api-go/internal/events"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

// AutoBanConfig controls automatic banning thresholds
type AutoBanConfig struct {
	Enabled              bool
	Threshold429         int
	Threshold403         int
	Threshold401         int
	Threshold5xx         int
	ConsecutiveFailLimit int
}

// DefaultAutoBanConfig mirrors the legacy behaviour prior to configuration support.
var DefaultAutoBanConfig = AutoBanConfig{
	Enabled:              true,
	Threshold429:         3,
	Threshold403:         5,
	Threshold401:         3,
	Threshold5xx:         10,
	ConsecutiveFailLimit: 10,
}

// Options configure how the credential manager behaves.
type Options struct {
	AuthDir                    string
	RotationThreshold          int32
	AutoBan                    AutoBanConfig
	AutoRecoveryEnabled        bool
	AutoRecoveryInterval       time.Duration
	Sources                    []CredentialSource
	MaxConcurrentPerCredential int
	// Token refresh
	RefreshAheadSeconds int
	// Optional stores/coordinators
	StateStore         StateStore
	RefreshCoordinator RefreshCoordinator
}

// Manager manages multiple credentials with rotation and circuit breaking
type Manager struct {
	credentials       []*Credential
	currentIndex      int
	rotationThreshold int32
	mu                sync.RWMutex
	authDir           string
	autoBan           AutoBanConfig
	sources           []CredentialSource
	credSource        map[string]CredentialSource

	// ✅ Auto-recovery
	autoRecoveryEnabled  bool
	autoRecoveryInterval time.Duration
	recoveryTicker       *time.Ticker
	stopRecovery         chan struct{}

	// ✅ Hot reload
	reloadCh    chan struct{}
	watchOnce   sync.Once
	watcher     *fsnotify.Watcher
	reloadMu    sync.Mutex
	reloadTimer *time.Timer
	persistMu   sync.Mutex
	lastPersist map[string]time.Time

	// Concurrency control per credential
	maxConcPerCred int
	sems           map[string]chan struct{}
	semMu          sync.Mutex

	// Token refresh policy
	refreshAheadSec int

	// Optional components
	stateStore   StateStore
	refreshCoord RefreshCoordinator

	publisher events.Publisher
}

const (
	credentialStateSuffix = ".state.json"
	statePersistInterval  = 10 * time.Second
	watchDebounceInterval = 300 * time.Millisecond
)

// NewManager creates a new credential manager
func NewManager(opts Options) *Manager {
	rotation := opts.RotationThreshold
	if rotation <= 0 {
		rotation = 100
	}
	autoBan := DefaultAutoBanConfig
	if opts.AutoBan.Enabled {
		autoBan.Enabled = true
	} else if opts.AutoBan.Enabled == false {
		autoBan.Enabled = false
	}
	if opts.AutoBan.Threshold429 > 0 {
		autoBan.Threshold429 = opts.AutoBan.Threshold429
	}
	if opts.AutoBan.Threshold403 > 0 {
		autoBan.Threshold403 = opts.AutoBan.Threshold403
	}
	if opts.AutoBan.Threshold401 > 0 {
		autoBan.Threshold401 = opts.AutoBan.Threshold401
	}
	if opts.AutoBan.Threshold5xx > 0 {
		autoBan.Threshold5xx = opts.AutoBan.Threshold5xx
	}
	if opts.AutoBan.ConsecutiveFailLimit > 0 {
		autoBan.ConsecutiveFailLimit = opts.AutoBan.ConsecutiveFailLimit
	}

	interval := opts.AutoRecoveryInterval
	if interval <= 0 {
		interval = 10 * time.Minute
	}

	ahead := opts.RefreshAheadSeconds
	if ahead <= 0 {
		ahead = 180
	}
	mgr := &Manager{
		credentials:          make([]*Credential, 0),
		rotationThreshold:    rotation,
		authDir:              opts.AuthDir,
		sources:              filterSources(opts.Sources),
		credSource:           make(map[string]CredentialSource),
		autoBan:              autoBan,
		autoRecoveryEnabled:  opts.AutoRecoveryEnabled,
		autoRecoveryInterval: interval,
		stopRecovery:         make(chan struct{}),
		reloadCh:             make(chan struct{}, 1),
		lastPersist:          make(map[string]time.Time),
		maxConcPerCred:       opts.MaxConcurrentPerCredential,
		sems:                 make(map[string]chan struct{}),
		refreshAheadSec:      ahead,
		stateStore:           opts.StateStore,
		refreshCoord:         opts.RefreshCoordinator,
	}

	if len(mgr.sources) == 0 && mgr.authDir != "" {
		mgr.sources = []CredentialSource{NewFileSource(mgr.authDir)}
	}

	return mgr
}

func filterSources(sources []CredentialSource) []CredentialSource {
	if len(sources) == 0 {
		return nil
	}
	out := make([]CredentialSource, 0, len(sources))
	for _, src := range sources {
		if src != nil {
			out = append(out, src)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (m *Manager) getCredentialSource(id string) CredentialSource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if src, ok := m.credSource[id]; ok {
		return src
	}
	return nil
}

func (m *Manager) findFileSource() CredentialSource {
	for _, src := range m.sources {
		if _, ok := src.(*FileSource); ok {
			return src
		}
	}
	return nil
}

// SetEventPublisher wires the event hub used to broadcast credential changes.
func (m *Manager) SetEventPublisher(p events.Publisher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publisher = p
}

// LoadCredentials loads credentials from configured sources (defaults to authDir).
func (m *Manager) LoadCredentials() error {
	ctx := context.Background()
	if len(m.sources) == 0 {
		return fmt.Errorf("no credential sources configured")
	}

	aggregated := make([]*Credential, 0)
	sourceIndex := make(map[string]CredentialSource)
	seen := make(map[string]struct{})

	for _, src := range m.sources {
		creds, err := src.Load(ctx)
		if err != nil {
			log.WithError(err).Warnf("credential source %s load failed", src.Name())
			continue
		}
		if len(creds) == 0 {
			continue
		}
		for _, cred := range creds {
			if cred == nil {
				continue
			}
			if cred.ID == "" {
				log.Warnf("credential source %s returned credential without id", src.Name())
				continue
			}
			if _, exists := seen[cred.ID]; exists {
				log.Warnf("duplicate credential id %s found in source %s, skipping", cred.ID, src.Name())
				continue
			}
			if cred.Source == "" {
				cred.Source = src.Name()
			}
			if stateful, ok := src.(StatefulCredentialSource); ok {
				if err := stateful.RestoreState(ctx, cred); err != nil {
					log.WithError(err).Warnf("restore state failed for %s via source %s", cred.ID, src.Name())
				}
			} else {
				m.restoreCredentialState(cred)
			}
			aggregated = append(aggregated, cred)
			sourceIndex[cred.ID] = src
			seen[cred.ID] = struct{}{}
		}
	}

	sort.Slice(aggregated, func(i, j int) bool {
		if aggregated[i] == nil || aggregated[j] == nil {
			return false
		}
		return aggregated[i].ID < aggregated[j].ID
	})

	m.mu.Lock()
	m.credentials = aggregated
	m.credSource = sourceIndex
	m.mu.Unlock()

	m.persistMu.Lock()
	m.lastPersist = make(map[string]time.Time, len(aggregated))
	m.persistMu.Unlock()

	log.Infof("Loaded %d credentials from %d source(s)", len(aggregated), len(m.sources))
	m.emitCredentialSnapshot(aggregated)
	return nil
}

// GetAllCredentials returns a copy of all credentials (for metrics/monitoring)
func (m *Manager) GetAllCredentials() []*Credential {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clones := make([]*Credential, len(m.credentials))
	for i, cred := range m.credentials {
		clones[i] = cred.Clone()
	}
	return clones
}

// GetCredentialByID returns a cloned credential by id if present.
func (m *Manager) GetCredentialByID(id string) (*Credential, bool) {
	if id == "" {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, cred := range m.credentials {
		if cred != nil && cred.ID == id {
			return cred.Clone(), true
		}
	}
	return nil, false
}

// DisableCredential manually disables a credential
func (m *Manager) DisableCredential(credID string) error {
	target, err := m.mutateCredential(credID, func(c *Credential) error {
		c.Disabled = true
		return nil
	})
	if err != nil {
		return err
	}

	log.Infof("Disabled credential %s", credID)
	m.persistCredentialState(target, true)
	m.emitCredentialEvent("disabled", target.Clone())
	return nil
}

// EnableCredential manually enables a credential
func (m *Manager) EnableCredential(credID string) error {
	target, err := m.mutateCredential(credID, func(c *Credential) error {
		c.Disabled = false
		c.FailureCount = 0
		return nil
	})
	if err != nil {
		return err
	}

	log.Infof("Enabled credential %s", credID)
	m.persistCredentialState(target, true)
	m.emitCredentialEvent("enabled", target.Clone())
	return nil
}

// DeleteCredential removes a credential from manager and deletes backing file
func (m *Manager) DeleteCredential(credID string) error {
	target, src, err := m.removeCredential(credID)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if src != nil {
		if writable, ok := src.(WritableCredentialSource); ok {
			if err := writable.Delete(ctx, credID); err != nil {
				return fmt.Errorf("failed to delete credential via %s: %w", src.Name(), err)
			}
		} else if err := m.deleteCredentialLegacy(credID); err != nil {
			return err
		}
	} else if err := m.deleteCredentialLegacy(credID); err != nil {
		return err
	}
	log.Infof("Deleted credential %s", credID)
	m.deleteCredentialState(credID)
	if target != nil {
		m.emitCredentialEvent("deleted", target.Clone())
	}
	return nil
}

// CleanupExpired removes expired credentials that cannot be refreshed
func (m *Manager) CleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var active []*Credential
	for _, cred := range m.credentials {
		if cred.Type == "oauth" && cred.IsExpired() && cred.RefreshToken == "" {
			log.Infof("Removing expired credential %s", cred.ID)
			continue
		}
		active = append(active, cred)
	}

	m.credentials = active
}

// ResetAllStats clears runtime counters for every credential
func (m *Manager) ResetAllStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cred := range m.credentials {
		cred.ResetStats()
	}
}

// ✅ StartAutoRecovery starts automatic recovery of banned credentials
// See manager_watch.go for WatchAuthDirectory implementation
// See manager_persist.go for credential state persistence implementation
