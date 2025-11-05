package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"
	enhmgmt "gcli2api-go/internal/handlers/management"
	monenh "gcli2api-go/internal/monitoring"
)

func TestBuildEnginesEnforcesAuth(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.OpenAIPort = "0"
	cfg.Upstream.OpenAIKey = "sk-test"
	cfg.Security.ManagementKey = "mgmt"
	cfg.APICompat.PreferredBaseModels = []string{"gemini-2.5-pro"}
	cfg.SyncFromDomains()

	metrics := monenh.NewEnhancedMetrics()
	enhanced := enhmgmt.NewAdminAPIHandler(cfg, nil, metrics, nil, nil)
	openaiEngine := buildOpenAIEngine(cfg, Dependencies{EnhancedMetrics: metrics}, enhanced)

	t.Run("missing key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		rec := httptest.NewRecorder()
		openaiEngine.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("valid key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		req.Header.Set("Authorization", "Bearer sk-test")
		rec := httptest.NewRecorder()
		openaiEngine.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}
