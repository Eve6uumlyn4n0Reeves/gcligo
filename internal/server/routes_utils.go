package server

import (
	"net/http"
	pp "net/http/pprof"
	"strconv"
	"strings"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/models"
	store "gcli2api-go/internal/storage"
	"github.com/gin-gonic/gin"
)

func respondError(c *gin.Context, status int, message string, details any) {
	payload := gin.H{"error": message}
	if details != nil {
		payload["details"] = details
	}
	c.JSON(status, payload)
}

func respondValidationError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	respondError(c, http.StatusBadRequest, "invalid json", err.Error())
}

func bindJSON(c *gin.Context, dest any) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		respondValidationError(c, err)
		return false
	}
	return true
}

func setNoCacheHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

func registerPprof(r *gin.Engine) {
	ppGroup := r.Group("/debug/pprof")
	ppGroup.GET("/", gin.WrapF(pp.Index))
	ppGroup.GET("/cmdline", gin.WrapF(pp.Cmdline))
	ppGroup.GET("/profile", gin.WrapF(pp.Profile))
	ppGroup.POST("/symbol", gin.WrapF(pp.Symbol))
	ppGroup.GET("/symbol", gin.WrapF(pp.Symbol))
	ppGroup.GET("/trace", gin.WrapF(pp.Trace))
	ppGroup.GET("/allocs", gin.WrapF(pp.Handler("allocs").ServeHTTP))
	ppGroup.GET("/block", gin.WrapF(pp.Handler("block").ServeHTTP))
	ppGroup.GET("/goroutine", gin.WrapF(pp.Handler("goroutine").ServeHTTP))
	ppGroup.GET("/heap", gin.WrapF(pp.Handler("heap").ServeHTTP))
	ppGroup.GET("/mutex", gin.WrapF(pp.Handler("mutex").ServeHTTP))
	ppGroup.GET("/threadcreate", gin.WrapF(pp.Handler("threadcreate").ServeHTTP))
}

func buildRoutesJSON(cfg *config.Config, st store.Backend) map[string]interface{} {
	ids := models.ExposedModelIDsByChannel(cfg, st, "openai")
	if len(ids) == 0 {
		ids = cfg.APICompat.PreferredBaseModels
		if len(ids) == 0 {
			ids = models.DefaultBaseModels()
		}
	}
	modelObjs := make([]map[string]any, 0, len(ids))
	imagesEnabled := false
	for _, id := range ids {
		base := strings.ToLower(models.BaseFromFeature(id))
		modalities := []string{"text"}
		if strings.Contains(base, "flash-image") {
			modalities = []string{"image", "text"}
			imagesEnabled = true
		}
		modelObjs = append(modelObjs, map[string]any{"id": id, "modalities": modalities})
	}
	servers := make([]any, 0, 2)
	openaiServer := map[string]any{
		"name":     "openai",
		"port":     mustAtoi(cfg.Server.OpenAIPort),
		"base_url": joinBasePath(cfg.Server.BasePath, "/v1"),
		"endpoints": []string{
			joinBasePath(cfg.Server.BasePath, "/v1/models"),
			joinBasePath(cfg.Server.BasePath, "/v1/models/:id"),
			joinBasePath(cfg.Server.BasePath, "/v1/chat/completions"),
			joinBasePath(cfg.Server.BasePath, "/v1/images/generations"),
			joinBasePath(cfg.Server.BasePath, "/v1/responses"),
			joinBasePath(cfg.Server.BasePath, "/v1/completions"),
		},
		"features": map[string]any{
			"images_enabled":        imagesEnabled,
			"tool_args_delta_chunk": cfg.APICompat.ToolArgsDeltaChunk,
			"anti_truncation_max":   cfg.ResponseShaping.AntiTruncationMax,
			"header_passthrough":    cfg.Security.HeaderPassThrough,
		},
		"models": modelObjs,
		"auth":   map[string]any{"type": "bearer", "key": cfg.Upstream.OpenAIKey},
	}
	servers = append(servers, openaiServer)
	if cfg.Server.GeminiPort != "" && cfg.Server.GeminiPort != "0" {
		geminiServer := map[string]any{
			"name":     "gemini",
			"port":     mustAtoi(cfg.Server.GeminiPort),
			"base_url": joinBasePath(cfg.Server.BasePath, "/v1"),
			"endpoints": []string{
				joinBasePath(cfg.Server.BasePath, "/v1/models"),
				joinBasePath(cfg.Server.BasePath, "/v1/models/:id"),
				joinBasePath(cfg.Server.BasePath, "/v1/models/:model:generateContent"),
				joinBasePath(cfg.Server.BasePath, "/v1/models/:model:streamGenerateContent"),
				joinBasePath(cfg.Server.BasePath, "/v1beta/models/:model:generateContent"),
				joinBasePath(cfg.Server.BasePath, "/v1beta/models/:model:streamGenerateContent"),
			},
			"features": map[string]any{
				"native_gemini_format":        true,
				"supports_contents":           true,
				"supports_system_instruction": true,
				"supports_generation_config":  true,
				"multi_auth_methods":          true,
			},
			"models": modelObjs,
			"auth":   map[string]any{"types": []string{"bearer", "x-goog-api-key", "url_param"}, "key": cfg.Upstream.GeminiKey},
		}
		servers = append(servers, geminiServer)
	}
	return map[string]any{
		"name":     "GCLI2API-Go",
		"version":  "3.1.0",
		"servers":  servers,
		"note":     "Models and endpoints reflect current dynamic registry. Dual-endpoint system: OpenAI-compatible + Gemini native.",
		"features": map[string]any{"dual_endpoints": len(servers) > 1, "env_credentials": cfg.Execution.AutoLoadEnvCreds, "variant_config_api": true, "batch_operations": true, "enhanced_oauth": true},
	}
}

// sanitizePlanName cleans plan name for safe storage keys.
func sanitizePlanName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// allow [a-zA-Z0-9_-]
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		out = "default"
	}
	return out
}

func joinBasePath(basePath, suffix string) string {
	if basePath == "" {
		if suffix == "" {
			return ""
		}
		return suffix
	}
	if suffix == "" {
		return basePath
	}
	if suffix == "/" {
		return basePath + "/"
	}
	if strings.HasPrefix(suffix, "/") {
		return basePath + suffix
	}
	return basePath + "/" + suffix
}

func mustAtoi(v string) int {
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		return n
	}
	return 0
}
