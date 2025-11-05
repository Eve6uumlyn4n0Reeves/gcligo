package tests

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/models"
	"gcli2api-go/internal/monitoring"
	srv "gcli2api-go/internal/server"
	store "gcli2api-go/internal/storage"
)

// minimal router mounting the management subset we touch
func newMgmtRouterForTest(t *testing.T, st store.Backend) *gin.Engine {
	t.Helper()
	cfg := &config.Config{}
	cfg.Server.WebAdminEnabled = false
	cfg.Server.BasePath = ""
	cfg.Security.ManagementKey = "secret"
	cfg.Security.ManagementAllowRemote = true
	cfg.SyncFromDomains()
	engine, _, _ := srv.BuildEngines(cfg, srv.Dependencies{Storage: st, EnhancedMetrics: monitoring.NewEnhancedMetrics()})
	return engine
}

func TestAssemblyPlanApplyAndRollback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st := newTempFileBackend(t)

	// Seed current registry with one entry
	seed := []models.RegistryEntry{{ID: "gemini-2.5-pro", Base: "gemini-2.5-pro", Enabled: true}}
	if err := st.SetConfig(context.Background(), "model_registry_openai", seed); err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := srv.NewAssemblyService(&config.Config{}, st, monitoring.NewEnhancedMetrics(), nil)

	// Save plan via service
	plan, err := svc.SavePlan(context.Background(), "it", map[string]bool{"models": true, "variants": false})
	if err != nil {
		t.Fatalf("save plan: %v", err)
	}
	_ = plan

	// Apply plan (no-op since it uses current snapshot)
	if err := svc.ApplyPlan(context.Background(), "it"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := st.GetConfig(context.Background(), "model_registry_openai"); err != nil {
		t.Fatalf("get registry: %v", err)
	}

	// Rollback (should succeed, restoring backup)
	if err := svc.RollbackPlan(context.Background(), "it"); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if _, err := st.GetConfig(context.Background(), "model_registry_openai"); err != nil {
		t.Fatalf("get registry 2: %v", err)
	}
}
