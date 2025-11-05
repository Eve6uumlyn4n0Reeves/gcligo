package gemini

import (
	"context"
	"net/http"
	"sync"

	"gcli2api-go/internal/config"
	credpkg "gcli2api-go/internal/credential"
	hcommon "gcli2api-go/internal/handlers/common"
	statstracker "gcli2api-go/internal/stats"
	store "gcli2api-go/internal/storage"
	upstream "gcli2api-go/internal/upstream"
	up "gcli2api-go/internal/upstream/gemini"
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
	cfg         *config.Config
	cl          upstreamClient
	credMgr     *credpkg.Manager
	usageStats  *statstracker.UsageStats
	clientCache map[string]upstreamClient
	cacheMu     sync.RWMutex
	store       store.Backend
	router      *route.Strategy
}

func New(cfg *config.Config, credMgr *credpkg.Manager, usage *statstracker.UsageStats, st store.Backend) *Handler {
	h := &Handler{
		cfg:         cfg,
		cl:          up.New(cfg).WithCaller("gemini"),
		credMgr:     credMgr,
		usageStats:  usage,
		clientCache: make(map[string]upstreamClient),
		store:       st,
	}
	h.router = route.NewStrategy(cfg, credMgr, func(credID string) { h.invalidateClientCache(credID) })
	return h
}

// NewWithStrategy constructs handler with a shared routing strategy.
func NewWithStrategy(cfg *config.Config, credMgr *credpkg.Manager, usage *statstracker.UsageStats, st store.Backend, router *route.Strategy) *Handler {
	h := &Handler{
		cfg:         cfg,
		cl:          up.New(cfg).WithCaller("gemini"),
		credMgr:     credMgr,
		usageStats:  usage,
		clientCache: make(map[string]upstreamClient),
		store:       st,
	}
	if router == nil {
		router = route.NewStrategy(cfg, credMgr, func(credID string) { h.invalidateClientCache(credID) })
	}
	h.router = router
	return h
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
