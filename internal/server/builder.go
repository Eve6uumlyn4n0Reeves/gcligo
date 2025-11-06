package server

import (
	"context"
	"net/http"
	"strings"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	gh "gcli2api-go/internal/handlers/gemini"
	enhmgmt "gcli2api-go/internal/handlers/management"
	oh "gcli2api-go/internal/handlers/openai"
	"gcli2api-go/internal/logging"
	mw "gcli2api-go/internal/middleware"
	monenh "gcli2api-go/internal/monitoring"
	usagestats "gcli2api-go/internal/stats"
	store "gcli2api-go/internal/storage"
	upstream "gcli2api-go/internal/upstream"
	upgem "gcli2api-go/internal/upstream/gemini"
	route "gcli2api-go/internal/upstream/strategy"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const adminAssetVersion = "20251026"

// Dependencies encapsulates runtime services required to build the HTTP engines.
type Dependencies struct {
	CredentialManager *credential.Manager
	UsageStats        *usagestats.UsageStats
	Storage           store.Backend
	EnhancedMetrics   *monenh.EnhancedMetrics
	RoutingStrategy   *route.Strategy
}

// BuildEngines constructs OpenAI 和 Gemini 的 Gin 引擎，并返回共享的路由策略实例。
func BuildEngines(cfg *config.Config, deps Dependencies) (*gin.Engine, *gin.Engine, *route.Strategy) {
	// Safety: when remote management is enabled, never allow upstream header passthrough
	if cfg.Security.ManagementAllowRemote && cfg.Security.HeaderPassthroughConfig.Enabled {
		log.Warn("ManagementAllowRemote=true -> forcing HeaderPassthrough=false for safety")
		cfg.Security.HeaderPassthroughConfig.Enabled = false
		cfg.Security.HeaderPassThrough = false
		cfg.HeaderPassThrough = false
	}
	// Disable routing debug headers in production-like mode for safety
	if !cfg.Security.Debug && cfg.Routing.DebugHeaders {
		log.Warn("Debug=false -> disabling RoutingDebugHeaders for safety")
		cfg.Routing.DebugHeaders = false
	}
	metricsEnhanced := deps.EnhancedMetrics
	if metricsEnhanced == nil {
		metricsEnhanced = monenh.NewEnhancedMetrics()
	}
	deps.EnhancedMetrics = metricsEnhanced
	enhancedHandler := enhmgmt.NewAdminAPIHandler(cfg, deps.CredentialManager, metricsEnhanced, deps.UsageStats, deps.Storage)
	// Shared routing strategy across both engines; default onRefresh no-op for now
	sharedRouter := route.NewStrategy(cfg, deps.CredentialManager, nil)

	openaiEngine, openaiHandler := buildOpenAIEngineWithRouter(cfg, deps, enhancedHandler, sharedRouter)
	geminiEngine, geminiHandler := buildGeminiEngineWithRouter(cfg, deps, enhancedHandler, sharedRouter)
	// Bind refresh callback after handlers are created so it can invalidate caches in both
	sharedRouter.SetOnRefresh(func(credID string) {
		if openaiHandler != nil {
			openaiHandler.InvalidateCachesFor(credID)
		}
		if geminiHandler != nil {
			geminiHandler.InvalidateCacheFor(credID)
		}
	})
	// Start auto-probe if enabled
	if cfg.AutoProbe.Enabled {
		enhancedHandler.StartAutoProbe(context.Background())
	}
	return openaiEngine, geminiEngine, sharedRouter
}

func buildOpenAIEngineWithRouter(cfg *config.Config, deps Dependencies, enhancedHandler *enhmgmt.EnhancedHandler, sharedRouter *route.Strategy) (*gin.Engine, *oh.Handler) {
	depsWithStrategy := deps
	depsWithStrategy.RoutingStrategy = sharedRouter

	openaiEngine := gin.New()
	applyStandardEngineSettings(openaiEngine, cfg, "openai")

	logging.InstallWebSocketLogging()

	if cfg.ResponseShaping.PprofEnabled {
		registerPprof(openaiEngine)
	}

	basePath := cfg.Server.BasePath
	root := openaiEngine.Group(basePath)

	// Mount OpenAI-compatible API routes via extracted module
	oa := RegisterOpenAIRoutes(root, cfg, depsWithStrategy, sharedRouter)

	root.GET("/meta/routes", func(c *gin.Context) {
		c.JSON(http.StatusOK, buildRoutesJSON(cfg, deps.Storage))
	})
	if basePath != "" {
		openaiEngine.GET(basePath, func(c *gin.Context) {
			c.Redirect(http.StatusTemporaryRedirect, joinBasePath(cfg.Server.BasePath, "/admin"))
		})
	}
	// Unify entry: redirect landing and /routes to /admin
	root.GET("/", func(c *gin.Context) {
		target := joinBasePath(cfg.Server.BasePath, "/admin")
		c.Redirect(http.StatusTemporaryRedirect, target)
	})
	root.GET("/routes", func(c *gin.Context) {
		target := joinBasePath(cfg.Server.BasePath, "/admin")
		c.Redirect(http.StatusTemporaryRedirect, target)
	})
	// Keep /login for compatibility, but redirect to /admin (admin will inline-login if unauthenticated)
	root.GET("/login", func(c *gin.Context) {
		target := joinBasePath(cfg.Server.BasePath, "/admin")
		c.Redirect(http.StatusTemporaryRedirect, target)
	})
	// /login.js 已内联进 login.html，不再单独暴露
	if cfg.Server.WebAdminEnabled {
		root.GET("/admin", func(c *gin.Context) {
			if enhancedHandler == nil {
				serveLoginHTML(c)
				return
			}
			if token, err := c.Cookie("mgmt_session"); err != nil || !enhancedHandler.ValidateToken(strings.TrimSpace(token)) {
				// Serve login inline under /admin for single-entry UX
				serveLoginHTML(c)
				return
			}
			serveAdminHTML(c)
		})
		// HEAD /admin：避免探活误判
		root.HEAD("/admin", func(c *gin.Context) { c.Status(http.StatusOK) })
		root.GET("/admin/", func(c *gin.Context) {
			if enhancedHandler == nil {
				serveLoginHTML(c)
				return
			}
			if token, err := c.Cookie("mgmt_session"); err != nil || !enhancedHandler.ValidateToken(strings.TrimSpace(token)) {
				serveLoginHTML(c)
				return
			}
			serveAdminHTML(c)
		})
		root.HEAD("/admin/", func(c *gin.Context) { c.Status(http.StatusOK) })
		// Serve admin assets and module imports with proper MIME types + cache control
		registerAdminStatic(root)

		// External model assembly standalone view no longer exposed; redirect to admin#assembly
		root.GET("/assembly", func(c *gin.Context) {
			target := joinBasePath(cfg.Server.BasePath, "/admin#assembly")
			c.Redirect(http.StatusTemporaryRedirect, target)
		})

		// Add base path information endpoint for frontend
		registerMetaBasePath(root, cfg)
	}
	root.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	root.GET("/metrics", mw.MetricsHandler)

	// Aliases outside basePath for static assets when basePath is non-empty
	if basePath != "" {
		registerAdminStatic(openaiEngine)
	}

	registerManagementRoutes2(root, cfg, depsWithStrategy, enhancedHandler)
	return openaiEngine, oa
}

func buildGeminiEngineWithRouter(cfg *config.Config, deps Dependencies, enhancedHandler *enhmgmt.EnhancedHandler, sharedRouter *route.Strategy) (*gin.Engine, *gh.Handler) {
	depsWithStrategy := deps
	depsWithStrategy.RoutingStrategy = sharedRouter

	geminiEngine := gin.New()
	applyStandardEngineSettings(geminiEngine, cfg, "gemini")

	basePath := cfg.Server.BasePath
	root := geminiEngine.Group(basePath)

	// Mount Gemini-native API routes via extracted module
	geminiHandler := RegisterGeminiRoutes(root, cfg, depsWithStrategy, sharedRouter)

	root.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	root.GET("/metrics", mw.MetricsHandler)

	return geminiEngine, geminiHandler
}

// Backward-compat helper for tests referencing the old signature.
func buildOpenAIEngine(cfg *config.Config, deps Dependencies, enhancedHandler *enhmgmt.EnhancedHandler) *gin.Engine {
	strategy := route.NewStrategy(cfg, deps.CredentialManager, nil)
	engine, _ := buildOpenAIEngineWithRouter(cfg, deps, enhancedHandler, strategy)
	return engine
}

// buildProvidersFromConfig registers upstream providers according to configuration.
// By design this project targets a single upstream; this switch keeps the door open
// without introducing multi-upstream complexity.
func buildProvidersFromConfig(cfg *config.Config) *upstream.Manager {
	prov := strings.ToLower(strings.TrimSpace(cfg.Upstream.UpstreamProvider))
	if prov != "" && prov != "gemini" && prov != "code_assist" {
		log.WithField("upstream_provider", prov).Warn("unsupported upstream provider; forcing gemini")
	}
	return upstream.NewManager(upgem.NewProvider(cfg))
}

// keep a small stub for legacy references; the real routes are in routes_management.go
func registerManagementRoutes(router *gin.RouterGroup, cfg *config.Config, deps Dependencies, enhancedHandler *enhmgmt.EnhancedHandler) {
	registerManagementRoutes2(router, cfg, deps, enhancedHandler)
}
