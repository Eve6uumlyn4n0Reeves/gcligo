package server

import (
	"gcli2api-go/internal/config"
	oh "gcli2api-go/internal/handlers/openai"
	mw "gcli2api-go/internal/middleware"
	route "gcli2api-go/internal/upstream/strategy"
	"github.com/gin-gonic/gin"
)

// RegisterOpenAIRoutes mounts OpenAI-compatible endpoints under the given router group.
// It mirrors the original routes previously defined inline in builder.go, without
// changing any external paths or auth behavior.
func RegisterOpenAIRoutes(root *gin.RouterGroup, cfg *config.Config, deps Dependencies, sharedRouter *route.Strategy) *oh.Handler {
	// Prefer multi-key auth when file config provides api_keys; fallback to single RequiredKey
	var openaiAuth gin.HandlerFunc
	if cm := config.GetConfigManager(); cm != nil {
		if fc := cm.GetConfig(); fc != nil && len(fc.APIKeys) > 0 {
			openaiAuth = mw.MultiKeyAuth(fc.APIKeys)
		}
	}
	if openaiAuth == nil {
		openaiAuth = mw.UnifiedAuth(mw.AuthConfig{RequiredKey: cfg.Upstream.OpenAIKey})
	}

	providers := buildProvidersFromConfig(cfg)
	oa := oh.NewWithStrategy(cfg, deps.CredentialManager, deps.UsageStats, deps.Storage, providers, sharedRouter)

	v1 := root.Group("/v1")
	v1.Use(openaiAuth)

	// Health/metrics are registered in builder.go

	// OpenAI-compatible endpoints
	v1.GET("/models", oa.ListModels)
	v1.GET("/models/:id", oa.GetModel)
	v1.POST("/chat/completions", oa.ChatCompletions)
	v1.POST("/completions", oa.Completions)
	v1.POST("/responses", oa.Responses)
	v1.POST("/images/generations", oa.ImagesGenerations)

	return oa
}
