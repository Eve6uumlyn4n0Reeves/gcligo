package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/monitoring"
	srv "gcli2api-go/internal/server"
	store "gcli2api-go/internal/storage"
	"github.com/gin-gonic/gin"
)

func TestAssemblySplitEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// minimal config with management key
	cfg := &config.Config{}
	cfg.Server.BasePath = ""
	cfg.Server.WebAdminEnabled = true
	cfg.Security.ManagementKey = "secret"
	cfg.Security.ManagementAllowRemote = true
	cfg.SyncFromDomains()

	// deps: file storage + empty cred manager
	tmp := t.TempDir()
	fb := store.NewFileBackend(tmp)
	if err := fb.Initialize(nil); err != nil {
		t.Fatalf("file backend init: %v", err)
	}
	mgr := credential.NewManager(credential.Options{AuthDir: tmp})
	_ = mgr.LoadCredentials()

	deps := srv.Dependencies{CredentialManager: mgr, Storage: fb, EnhancedMetrics: monitoring.NewEnhancedMetrics()}
	engine, _, _ := srv.BuildEngines(cfg, deps)

	// helper
	do := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/routes/api/management"+path, nil)
		req.Header.Set("Authorization", "Bearer "+cfg.ManagementKey)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w
	}

	tests := []string{"/assembly/overview", "/assembly/routes-meta", "/assembly/models", "/assembly/credentials", "/assembly/routing", "/assembly/usage"}
	for _, p := range tests {
		rec := do(p)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s => status %d, body=%s", p, rec.Code, rec.Body.String())
		}
	}
}
