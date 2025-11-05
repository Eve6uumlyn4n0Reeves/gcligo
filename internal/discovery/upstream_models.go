package discovery

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/models"
	"gcli2api-go/internal/monitoring"
	"gcli2api-go/internal/oauth"
	up "gcli2api-go/internal/upstream/gemini"
	log "github.com/sirupsen/logrus"
)

const (
	discoveryTTL            = 30 * time.Minute
	discoveryRequestTimeout = 20 * time.Second
)

// UpstreamModelDiscovery pulls base models from Gemini upstream and caches them.
type UpstreamModelDiscovery struct {
	cfg     *config.Config
	credMgr *credential.Manager

	mu      sync.RWMutex
	cached  []string
	expires time.Time
}

// NewUpstreamModelDiscovery creates a discovery helper.
func NewUpstreamModelDiscovery(cfg *config.Config, credMgr *credential.Manager) *UpstreamModelDiscovery {
	if cfg == nil || credMgr == nil {
		return nil
	}
	return &UpstreamModelDiscovery{
		cfg:     cfg,
		credMgr: credMgr,
	}
}

// GetBases returns the cached upstream base models or refreshes them when stale.
func (d *UpstreamModelDiscovery) GetBases(ctx context.Context) ([]string, error) {
	if d == nil {
		return nil, errors.New("discovery not configured")
	}

	// Serve cached data if still fresh.
	d.mu.RLock()
	if len(d.cached) > 0 && time.Now().Before(d.expires) {
		out := make([]string, len(d.cached))
		copy(out, d.cached)
		expiresAt := d.expires
		d.mu.RUnlock()
		monitoring.UpstreamDiscoveryCacheHits.Inc()
		monitoring.UpstreamDiscoveryBases.Set(float64(len(out)))
		if !expiresAt.IsZero() {
			monitoring.UpstreamDiscoveryCacheExpiry.Set(float64(expiresAt.Unix()))
		}
		log.WithFields(log.Fields{
			"component":  "upstream_discovery",
			"source":     "cache",
			"bases":      len(out),
			"expires_at": expiresAt.UTC().Format(time.RFC3339),
		}).Debug("serving upstream models from cache")
		return out, nil
	}
	d.mu.RUnlock()

	// Refresh.
	bases, err := d.refresh(ctx)
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	expiresAt := time.Now().Add(discoveryTTL)
	d.cached = make([]string, len(bases))
	copy(d.cached, bases)
	d.expires = expiresAt
	d.mu.Unlock()

	monitoring.UpstreamDiscoveryBases.Set(float64(len(bases)))
	monitoring.UpstreamDiscoveryCacheExpiry.Set(float64(expiresAt.Unix()))

	return bases, nil
}

// Snapshot returns the cached bases without triggering refresh.
func (d *UpstreamModelDiscovery) Snapshot() (bases []string, expires time.Time, ok bool) {
	if d == nil {
		return nil, time.Time{}, false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.cached) == 0 {
		return nil, time.Time{}, false
	}
	out := make([]string, len(d.cached))
	copy(out, d.cached)
	return out, d.expires, true
}

func (d *UpstreamModelDiscovery) refresh(ctx context.Context) ([]string, error) {
	start := time.Now()
	if d.credMgr == nil {
		monitoring.UpstreamDiscoveryFetchTotal.WithLabelValues("no_credential_manager").Inc()
		monitoring.UpstreamDiscoveryFetchDuration.Observe(time.Since(start).Seconds())
		log.WithFields(log.Fields{
			"component": "upstream_discovery",
			"result":    "no_credential_manager",
		}).Warn("upstream discovery skipped: credential manager unavailable")
		return nil, errors.New("credential manager unavailable")
	}

	creds := d.credMgr.GetAllCredentials()
	if len(creds) == 0 {
		monitoring.UpstreamDiscoveryFetchTotal.WithLabelValues("no_credentials").Inc()
		monitoring.UpstreamDiscoveryFetchDuration.Observe(time.Since(start).Seconds())
		log.WithFields(log.Fields{
			"component": "upstream_discovery",
			"result":    "no_credentials",
		}).Warn("upstream discovery skipped: no credentials available")
		return nil, errors.New("no credentials available")
	}

	healthy := make([]*credential.Credential, 0, len(creds))
	unhealthy := make([]*credential.Credential, 0, len(creds))
	for _, c := range creds {
		if strings.TrimSpace(c.AccessToken) == "" {
			continue
		}
		if c.IsHealthy() {
			healthy = append(healthy, c)
		} else {
			unhealthy = append(unhealthy, c)
		}
	}
	ordered := append(healthy, unhealthy...)
	if len(ordered) == 0 {
		monitoring.UpstreamDiscoveryFetchTotal.WithLabelValues("no_credentials").Inc()
		monitoring.UpstreamDiscoveryFetchDuration.Observe(time.Since(start).Seconds())
		log.WithFields(log.Fields{
			"component": "upstream_discovery",
			"result":    "no_credentials",
		}).Warn("upstream discovery skipped: no usable credentials")
		return nil, errors.New("no usable credentials")
	}

	ctx, cancel := context.WithTimeout(ctx, discoveryRequestTimeout)
	defer cancel()

	var errs []error
	for _, cred := range ordered {
		project := strings.TrimSpace(cred.ProjectID)
		if project == "" {
			project = strings.TrimSpace(d.cfg.GoogleProjID)
		}

		client := up.NewWithCredential(d.cfg, &oauth.Credentials{
			AccessToken: cred.AccessToken,
			ProjectID:   project,
		}).WithCaller("discovery")

		bases, err := client.ListModels(ctx, project)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", cred.ID, err))
			continue
		}
		if len(bases) == 0 {
			continue
		}
		normalized := normalizeBases(bases)
		duration := time.Since(start)
		monitoring.UpstreamDiscoveryFetchTotal.WithLabelValues("success").Inc()
		monitoring.UpstreamDiscoveryFetchDuration.Observe(duration.Seconds())
		monitoring.UpstreamDiscoveryBases.Set(float64(len(normalized)))
		monitoring.UpstreamDiscoveryLastSuccess.Set(float64(time.Now().Unix()))
		log.WithFields(log.Fields{
			"component":   "upstream_discovery",
			"result":      "success",
			"credential":  cred.ID,
			"bases":       len(normalized),
			"duration_ms": duration.Milliseconds(),
		}).Info("upstream discovery succeeded")
		return normalized, nil
	}

	if len(errs) == 0 {
		monitoring.UpstreamDiscoveryFetchTotal.WithLabelValues("empty").Inc()
		monitoring.UpstreamDiscoveryFetchDuration.Observe(time.Since(start).Seconds())
		log.WithFields(log.Fields{
			"component": "upstream_discovery",
			"result":    "empty",
			"attempts":  len(ordered),
		}).Warn("upstream discovery returned no models")
		return nil, errors.New("no upstream models returned")
	}

	monitoring.UpstreamDiscoveryFetchTotal.WithLabelValues("error").Inc()
	monitoring.UpstreamDiscoveryFetchDuration.Observe(time.Since(start).Seconds())
	joined := errors.Join(errs...)
	log.WithError(joined).WithFields(log.Fields{
		"component": "upstream_discovery",
		"result":    "error",
		"attempts":  len(ordered),
	}).Error("upstream discovery failed")

	return nil, joined
}

func normalizeBases(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(in))
	for _, entry := range in {
		base := strings.TrimSpace(entry)
		if base == "" {
			continue
		}
		base = strings.ToLower(models.ParseModelName(base).BaseName)
		if base == "" {
			continue
		}
		set[base] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for base := range set {
		out = append(out, base)
	}
	sort.Strings(out)
	return out
}
