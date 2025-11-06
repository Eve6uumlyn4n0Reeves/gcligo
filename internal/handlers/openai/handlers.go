package openai

import (
	"context"
	"net/http"
	"sync"
	"time"

	"gcli2api-go/internal/antitrunc"
	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/monitoring"
	statstracker "gcli2api-go/internal/stats"
	store "gcli2api-go/internal/storage"
	upstream "gcli2api-go/internal/upstream"
	upgem "gcli2api-go/internal/upstream/gemini"
	"gcli2api-go/internal/usage"
	route "gcli2api-go/internal/upstream/strategy"
)

// geminiClient captures the subset of the upstream Gemini client used by OpenAI compatibility handlers.
type geminiClient interface {
	Generate(context.Context, []byte) (*http.Response, error)
	Stream(context.Context, []byte) (*http.Response, error)
	CountTokens(context.Context, []byte) (*http.Response, error)
	Action(context.Context, string, []byte) (*http.Response, error)
}

var _ geminiClient = (*upgem.Client)(nil)

// Handler aggregates shared dependencies for OpenAI-compatible endpoints.
type Handler struct {
	cfg           *config.Config
	credMgr       *credential.Manager
	usageStats    *statstracker.UsageStats
	usageTracker  *usage.Tracker
	providers     *upstream.Manager
	store         store.Backend
	baseClient    geminiClient
	clientCache   map[string]geminiClient
	cacheMu       sync.RWMutex
	router        *route.Strategy
	regexReplacer *antitrunc.RegexReplacer
}

// New constructs a new OpenAI-compatible handler set.
func New(cfg *config.Config, credMgr *credential.Manager, usage *statstracker.UsageStats, st store.Backend, providers *upstream.Manager) *Handler {
	if providers == nil {
		providers = upstream.NewManager(upgem.NewProvider(cfg))
	}
	h := &Handler{
		cfg:          cfg,
		credMgr:      credMgr,
		usageStats:   usage,
		usageTracker: nil, // Will be set later via SetUsageTracker
		providers:    providers,
		store:        st,
		baseClient:   upgem.New(cfg).WithCaller("openai"),
		clientCache:  make(map[string]geminiClient),
	}
	// Invalidate caches when router rotates credentials
	h.router = route.NewStrategy(cfg, credMgr, func(credID string) {
		h.invalidateClientCache(credID)
		h.invalidateProviderCache(credID)
	})
	h.initRegexReplacer(cfg)

	// Register cache invalidation hook with credential manager
	if credMgr != nil {
		credMgr.RegisterInvalidationHook(func(credID string, reason string) {
			h.invalidateClientCache(credID)
			h.invalidateProviderCache(credID)

			// Record cache invalidation metrics
			if metrics := monitoring.DefaultMetrics(); metrics != nil {
				metrics.RecordCacheInvalidation(credID, reason)
			}
		})
	}

	return h
}

// NewWithStrategy constructs handler with a shared routing strategy.
func NewWithStrategy(cfg *config.Config, credMgr *credential.Manager, usage *statstracker.UsageStats, st store.Backend, providers *upstream.Manager, router *route.Strategy) *Handler {
	if providers == nil {
		providers = upstream.NewManager(upgem.NewProvider(cfg))
	}
	h := &Handler{
		cfg:          cfg,
		credMgr:      credMgr,
		usageStats:   usage,
		usageTracker: nil, // Will be set later via SetUsageTracker
		providers:    providers,
		store:        st,
		baseClient:   upgem.New(cfg).WithCaller("openai"),
		clientCache:  make(map[string]geminiClient),
	}
	if router == nil {
		router = route.NewStrategy(cfg, credMgr, func(credID string) {
			h.invalidateClientCache(credID)
			h.invalidateProviderCache(credID)
		})
	}
	h.router = router
	h.initRegexReplacer(cfg)

	// Register cache invalidation hook with credential manager
	if credMgr != nil {
		credMgr.RegisterInvalidationHook(func(credID string, reason string) {
			h.invalidateClientCache(credID)
			h.invalidateProviderCache(credID)

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

// InvalidateCachesFor allows external components to clear per-credential caches.
func (h *Handler) InvalidateCachesFor(credID string) {
	h.invalidateClientCache(credID)
	h.invalidateProviderCache(credID)
}

// recordCredentialUsage records credential-level usage statistics for a request
func (h *Handler) recordCredentialUsage(credentialID, model string, tokens *usage.TokenUsage, success bool) {
	if h.usageTracker == nil {
		return
	}

	record := &usage.RequestRecord{
		Timestamp:    time.Now(),
		CredentialID: credentialID,
		API:          "openai",
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

// Other route handlers and helpers live in split files:
// - openai_chat.go: ChatCompletions
// - openai_completions.go: Completions
// - openai_models.go: ListModels/GetModel
// - openai_client.go: upstream client cache and acquisition
// - openai_usage.go: usage helpers
// - openai_utils.go/openai_fallback.go: streaming/fallback utilities
