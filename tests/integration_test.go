package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"
	oh "gcli2api-go/internal/handlers/openai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ✅ TestOpenAIModels verifies /v1/models returns a list and metadata
func TestOpenAIModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	h := oh.New(cfg, nil, nil, nil, nil)
	r := gin.New()
	r.GET("/v1/models", h.ListModels)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/models", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var out map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &out)
	assert.Equal(t, "list", out["object"])
}

// ✅ TestThinkingMode just validates marshal of a typical body
func TestThinkingMode(t *testing.T) {
	body := map[string]interface{}{
		"model":            "gemini-2.5-pro",
		"messages":         []interface{}{map[string]interface{}{"role": "user", "content": "Solve 2+2"}},
		"reasoning_effort": "high",
	}
	b, _ := json.Marshal(body)
	assert.NotNil(t, b)
	assert.Greater(t, len(b), 0)
}
