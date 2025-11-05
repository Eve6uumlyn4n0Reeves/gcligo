package server

import (
	"gcli2api-go/internal/config"
	gh "gcli2api-go/internal/handlers/gemini"
	mw "gcli2api-go/internal/middleware"
	route "gcli2api-go/internal/upstream/strategy"
	"github.com/gin-gonic/gin"
)

// RegisterGeminiRoutes mounts Gemini-native endpoints under the given router group.
// It mirrors the original routes previously defined inline in builder.go.
func RegisterGeminiRoutes(root *gin.RouterGroup, cfg *config.Config, deps Dependencies, sharedRouter *route.Strategy) *gh.Handler {
	geminiHandler := gh.NewWithStrategy(cfg, deps.CredentialManager, deps.UsageStats, deps.Storage, sharedRouter)

	// Health/metrics are registered in builder.go

	// Gemini native API endpoints
	var geminiAuth gin.HandlerFunc
	if cm := config.GetConfigManager(); cm != nil {
		if fc := cm.GetConfig(); fc != nil && len(fc.APIKeys) > 0 {
			geminiAuth = mw.MultiKeyAuth(fc.APIKeys)
		}
	}
	if geminiAuth == nil {
		geminiAuth = mw.UnifiedAuth(mw.AuthConfig{RequiredKey: cfg.Upstream.GeminiKey})
	}

	v1 := root.Group("/v1")
	v1.Use(geminiAuth)
	{
		v1.GET("/models", geminiHandler.Models)
		v1.GET("/models/:id", geminiHandler.GetModel)
		// Gin 不支持同一段内混合路径参数与字面冒号，使用尾部 *action 分发
		v1.POST("/models/:model/*action", func(c *gin.Context) {
			switch c.Param("action") {
			case ":generateContent":
				geminiHandler.GenerateContent(c)
			case ":streamGenerateContent":
				geminiHandler.StreamGenerateContent(c)
			case ":countTokens":
				geminiHandler.CountTokens(c)
			default:
				c.JSON(404, gin.H{"error": "unknown action"})
			}
		})
	}

	// Also support v1beta paths for compatibility
	v1beta := root.Group("/v1beta")
	{
		if geminiAuth != nil {
			v1beta.Use(geminiAuth)
		}
		v1beta.GET("/models", geminiHandler.ListModels)
		v1beta.GET("/models/:id", geminiHandler.ModelInfo)
	}

	return geminiHandler
}
