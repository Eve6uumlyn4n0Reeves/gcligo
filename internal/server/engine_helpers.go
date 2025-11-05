package server

import (
	"net/http"

	"gcli2api-go/internal/config"
	mw "gcli2api-go/internal/middleware"
	"github.com/gin-gonic/gin"
)

// applyStandardEngineSettings applies common Gin settings and middlewares
// used by both OpenAI-compatible and Gemini-native servers. It also tags
// requests with a server_label for downstream logging/metrics.
func applyStandardEngineSettings(engine *gin.Engine, cfg *config.Config, serverLabel string) {
	if !cfg.Security.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	_ = engine.SetTrustedProxies([]string{})

	engine.Use(gin.Recovery(), mw.RequestID(), mw.Metrics())
	// Apply CORS for public APIs; middleware itself skips management endpoints.
	engine.Use(mw.CORS())
	if cfg.ResponseShaping.RequestLogEnabled {
		engine.Use(mw.RequestLogger())
	}
	if cfg.RateLimit.Enabled {
		engine.Use(mw.RateLimiterAutoKey(cfg.RateLimit.RPS, cfg.RateLimit.Burst))
	}
	engine.Use(func(c *gin.Context) {
		c.Set("server_label", serverLabel)
		c.Next()
	})
}

// registerMetaBasePath registers a small endpoint that lets the frontend
// discover the effective base path and a few bootstrap flags. It is safe to
// register both under basePath and at root alias.
func registerMetaBasePath(r gin.IRoutes, cfg *config.Config) {
	r.GET("/meta/base-path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"base_path":         cfg.Server.BasePath,
			"web_admin_enabled": cfg.Server.WebAdminEnabled,
			// adminAssetVersion lives in the same package.
			"asset_version": adminAssetVersion,
		})
	})
}
