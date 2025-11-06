package gemini

import (
	"context"
	"net/http"
	"sync"
	"time"

	"gcli2api-go/internal/antitrunc"
	"gcli2api-go/internal/config"
	credpkg "gcli2api-go/internal/credential"
	hcommon "gcli2api-go/internal/handlers/common"
	"gcli2api-go/internal/monitoring"
	statstracker "gcli2api-go/internal/stats"
	store "gcli2api-go/internal/storage"
	upstream "gcli2api-go/internal/upstream"
	up "gcli2api-go/internal/upstream/gemini"
	"gcli2api-go/internal/usage"
	route "gcli2api-go/internal/upstream/strategy"
)

type upstreamClient interface {
	Generate(ctx context.Context, payload []byte) (*http.Response, error)
	Stream(ctx context.Context, payload []byte) (*http.Response, error)
	CountTokens(ctx context.Context, payload []byte) (*http.Response, error)
	Action(ctx context.Context, action string, payload []byte) (*http.Response, error)
}

var _ upstreamClient = (*up.Client)(nil)

// Handler manages Gemini-native endpoints and upstream coordination.
type Handler struct {
	cfg           *config.Config
	cl            upstreamClient
	credMgr       *credpkg.Manager
	usageStats    *statstracker.UsageStats
	usageTracker  *usage.Tracker
	clientCache   map[string]upstreamClient
	cacheMu       sync.RWMutex
	store         store.Backend
	router        *route.Strategy
	regexReplacer *antitrunc.RegexReplacer
}

func New(cfg *config.Config, credMgr *credpkg.Manager, usage *statstracker.UsageStats, st store.Backend) *Handler {
	h := &Handler{
		cfg:          cfg,
		cl:           up.New(cfg).WithCaller("gemini"),
		credMgr:      credMgr,
		usageStats:   usage,
		usageTracker: nil, // Will be set later via SetUsageTracker
		clientCache:  make(map[string]upstreamClient),
		store:        st,
	}
	h.router = route.NewStrategy(cfg, credMgr, func(credID string) { h.invalidateClientCache(credID) })
	h.initRegexReplacer(cfg)

	// Register cache invalidation hook with credential manager
	if credMgr != nil {
		credMgr.RegisterInvalidationHook(func(credID string, reason string) {
			h.invalidateClientCache(credID)

			// Record cache invalidation metrics
			if metrics := monitoring.DefaultMetrics(); metrics != nil {
				metrics.RecordCacheInvalidation(credID, reason)
			}
		})
	}

	return h
}

// NewWithStrategy constructs handler with a shared routing strategy.
func NewWithStrategy(cfg *config.Config, credMgr *credpkg.Manager, usage *statstracker.UsageStats, st store.Backend, router *route.Strategy) *Handler {
	h := &Handler{
		cfg:          cfg,
		cl:           up.New(cfg).WithCaller("gemini"),
		credMgr:      credMgr,
		usageStats:   usage,
		usageTracker: nil, // Will be set later via SetUsageTracker
		clientCache:  make(map[string]upstreamClient),
		store:        st,
	}
	if router == nil {
		router = route.NewStrategy(cfg, credMgr, func(credID string) { h.invalidateClientCache(credID) })
	}
	h.router = router
	h.initRegexReplacer(cfg)

	// Register cache invalidation hook with credential manager
	if credMgr != nil {
		credMgr.RegisterInvalidationHook(func(credID string, reason string) {
			h.invalidateClientCache(credID)

			// Record cache invalidation metrics
			if metrics := monitoring.DefaultMetrics(); metrics != nil {
				metrics.RecordCacheInvalidation(credID, reason)
			}
		})
	}

	return h
}

// initRegexReplacer initializes the regex replacer from configuration
func (h *Handler) initRegexReplacer(cfg *config.Config) {
	if cfg == nil {
		return
	}

	rules := make([]antitrunc.RegexRule, 0, len(cfg.RegexReplacements))
	for _, r := range cfg.RegexReplacements {
		rules = append(rules, antitrunc.RegexRule{
			Name:        r.Name,
			Pattern:     r.Pattern,
			Replacement: r.Replacement,
			Enabled:     r.Enabled,
		})
	}

	if len(rules) > 0 {
		replacer, err := antitrunc.NewRegexReplacer(rules)
		if err == nil {
			h.regexReplacer = replacer
		}
	}
}

// SetUsageTracker sets the usage tracker for credential-level statistics
func (h *Handler) SetUsageTracker(tracker *usage.Tracker) {
	h.usageTracker = tracker
}

// InvalidateCacheFor clears per-credential client cache.
func (h *Handler) InvalidateCacheFor(credID string) { h.invalidateClientCache(credID) }

func (h *Handler) invalidateClientCache(credID string) {
	if credID == "" {
		return
	}
	h.cacheMu.Lock()
	delete(h.clientCache, credID)
	h.cacheMu.Unlock()
}

// recordCredentialUsage records credential-level usage statistics for a request
func (h *Handler) recordCredentialUsage(credentialID, model string, tokens *usage.TokenUsage, success bool) {
	if h.usageTracker == nil {
		return
	}

	record := &usage.RequestRecord{
		Timestamp:    time.Now(),
		CredentialID: credentialID,
		API:          "gemini",
		Model:        model,
		Success:      success,
		Tokens:       tokens,
	}
	h.usageTracker.Record(record)
}

// extractTokenUsage extracts token usage from response body
func (h *Handler) extractTokenUsage(body []byte) *usage.TokenUsage {
	return usage.ExtractTokenUsageFromGeminiResponse(body)
}

func (h *Handler) shouldRefreshAhead(c *credpkg.Credential) bool {
	return hcommon.ShouldRefreshAhead(h.cfg, c)
}

// getUpstreamClient returns a per-request client bound to a selected credential if available.
func (h *Handler) getUpstreamClient(ctx context.Context) (upstreamClient, *credpkg.Credential) {
	if h.credMgr != nil && h.router != nil {
		hdr := upstream.HeaderOverrides(ctx)
		if cred := h.router.Pick(ctx, hdr); cred != nil {
			return h.getClientFor(cred), cred
		}
	}
	if h.credMgr != nil {
		if cred, err := h.credMgr.GetCredential(); err == nil && cred != nil {
			cred = h.router.PrepareCredential(ctx, cred)
			return h.getClientFor(cred), cred
		}
	}
	return h.cl, nil
}

// getClientFor returns a cached upstream client for the given credential id, creating if necessary.
func (h *Handler) getClientFor(cred *credpkg.Credential) upstreamClient {
	if cred == nil || cred.ID == "" {
		return h.cl
	}
	h.cacheMu.RLock()
	c, ok := h.clientCache[cred.ID]
	h.cacheMu.RUnlock()
	if ok && c != nil {
		return c
	}
	nc := hcommon.UpstreamClientFor(h.cfg, cred, "gemini")
	h.cacheMu.Lock()
	h.clientCache[cred.ID] = nc
	h.cacheMu.Unlock()
	return nc
}
