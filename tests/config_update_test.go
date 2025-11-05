package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"gcli2api-go/internal/config"
	enh "gcli2api-go/internal/handlers/management"
)

func TestUpdateConfig_TypeCoercionAndRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// initialize global config
	_ = config.LoadWithFile("")
	cfg := config.Load()

	h := enh.NewAdminAPIHandler(cfg, nil, nil, nil, nil)
	r := gin.New()
	grp := r.Group("/routes/api/management")
	h.RegisterRoutes(grp)

	// provide values as strings; server should coerce and apply to runtime config
	body := map[string]any{
		"rate_limit_rps":             "7",    // string -> int
		"retry_enabled":              "true", // string -> bool
		"openai_images_include_mime": true,   // bool direct
	}
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/routes/api/management/config", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// verify config manager and in-memory cfg were updated
	cm := config.GetConfigManager()
	fc := cm.GetConfig()
	require.Equal(t, 7, fc.RateLimitRPS)
	require.True(t, fc.RetryEnabled)
	require.True(t, fc.OpenAIImagesIncludeMime)
}
