package openai

import (
	"context"
	"net/http"
	"sync"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	statstracker "gcli2api-go/internal/stats"
	store "gcli2api-go/internal/storage"
	upstream "gcli2api-go/internal/upstream"
	upgem "gcli2api-go/internal/upstream/gemini"
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
	cfg         *config.Config
	credMgr     *credential.Manager
	usageStats  *statstracker.UsageStats
	providers   *upstream.Manager
	store       store.Backend
	baseClient  geminiClient
	clientCache map[string]geminiClient
	cacheMu     sync.RWMutex
	router      *route.Strategy
}

// New constructs a new OpenAI-compatible handler set.
func New(cfg *config.Config, credMgr *credential.Manager, usage *statstracker.UsageStats, st store.Backend, providers *upstream.Manager) *Handler {
	if providers == nil {
		providers = upstream.NewManager(upgem.NewProvider(cfg))
	}
	h := &Handler{
		cfg:         cfg,
		credMgr:     credMgr,
		usageStats:  usage,
		providers:   providers,
		store:       st,
		baseClient:  upgem.New(cfg).WithCaller("openai"),
		clientCache: make(map[string]geminiClient),
	}
	// Invalidate caches when router rotates credentials
	h.router = route.NewStrategy(cfg, credMgr, func(credID string) {
		h.invalidateClientCache(credID)
		h.invalidateProviderCache(credID)
	})
	return h
}

// NewWithStrategy constructs handler with a shared routing strategy.
func NewWithStrategy(cfg *config.Config, credMgr *credential.Manager, usage *statstracker.UsageStats, st store.Backend, providers *upstream.Manager, router *route.Strategy) *Handler {
	if providers == nil {
		providers = upstream.NewManager(upgem.NewProvider(cfg))
	}
	h := &Handler{
		cfg:         cfg,
		credMgr:     credMgr,
		usageStats:  usage,
		providers:   providers,
		store:       st,
		baseClient:  upgem.New(cfg).WithCaller("openai"),
		clientCache: make(map[string]geminiClient),
	}
	if router == nil {
		router = route.NewStrategy(cfg, credMgr, func(credID string) {
			h.invalidateClientCache(credID)
			h.invalidateProviderCache(credID)
		})
	}
	h.router = router
	return h
}

// InvalidateCachesFor allows external components to clear per-credential caches.
func (h *Handler) InvalidateCachesFor(credID string) {
	h.invalidateClientCache(credID)
	h.invalidateProviderCache(credID)
}

// Other route handlers and helpers live in split files:
// - openai_chat.go: ChatCompletions
// - openai_completions.go: Completions
// - openai_models.go: ListModels/GetModel
// - openai_client.go: upstream client cache and acquisition
// - openai_usage.go: usage helpers
// - openai_utils.go/openai_fallback.go: streaming/fallback utilities
