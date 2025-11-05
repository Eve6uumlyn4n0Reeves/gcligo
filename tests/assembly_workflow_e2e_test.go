package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/models"
	"gcli2api-go/internal/monitoring"
	srv "gcli2api-go/internal/server"
	"gcli2api-go/internal/stats"
	store "gcli2api-go/internal/storage"
)

func TestAssemblyWorkflowApplyRollbackIdempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	tmp := t.TempDir()

	backend := store.NewFileBackend(tmp)
	require.NoError(t, backend.Initialize(ctx))

	originalOA := []models.RegistryEntry{
		{ID: "gpt-4o", Base: "gpt-4o", Enabled: true},
		{ID: "gpt-4o-mini", Base: "gpt-4o-mini", Enabled: true},
	}
	originalGM := []models.RegistryEntry{
		{ID: "gemini-2.5-pro", Base: "gemini-2.5-pro", Enabled: true},
	}
	variantCfg := map[string]any{
		"default": map[string]any{"routing": "balanced", "cooldown": 30},
	}

	require.NoError(t, backend.SetConfig(ctx, "model_registry_openai", originalOA))
	require.NoError(t, backend.SetConfig(ctx, "model_registry_gemini", originalGM))
	require.NoError(t, backend.SetConfig(ctx, "model_variant_config", variantCfg))

	mgr := credential.NewManager(credential.Options{AuthDir: tmp})
	require.NoError(t, mgr.LoadCredentials())

	cfg := &config.Config{}
	cfg.Server.BasePath = ""
	cfg.Server.WebAdminEnabled = true
	cfg.Security.ManagementKey = "secret-key"
	cfg.Security.ManagementAllowRemote = true
	cfg.SyncFromDomains()

	metrics := monitoring.NewEnhancedMetrics()
	usage := stats.NewUsageStats(backend, time.Hour, "UTC", 0)

	engine, _, _ := srv.BuildEngines(cfg, srv.Dependencies{
		CredentialManager: mgr,
		Storage:           backend,
		EnhancedMetrics:   metrics,
		UsageStats:        usage,
	})

	client := newMgmtClient(engine, cfg.ManagementKey)

	// Ensure no plans exist initially
	resp := client.do(t, http.MethodGet, "/assembly/plans", nil)
	require.Equal(t, http.StatusOK, resp.Code)
	var listBody struct {
		Plans []map[string]any `json:"plans"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &listBody))
	require.Len(t, listBody.Plans, 0)

	planName := "workflow-plan"
	savePayload := map[string]any{
		"name":    planName,
		"include": map[string]bool{"models": true, "variants": true},
	}
	resp = client.doJSON(t, http.MethodPost, "/assembly/plans", savePayload)
	require.Equal(t, http.StatusOK, resp.Code)

	var saveBody struct {
		Message string         `json:"message"`
		Plan    map[string]any `json:"plan"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &saveBody))
	require.Equal(t, "plan saved", saveBody.Message)
	resp = client.do(t, http.MethodGet, fmt.Sprintf("/assembly/plans/%s", planName), nil)
	require.Equal(t, http.StatusOK, resp.Code)

	// Plan should now be retrievable
	resp = client.do(t, http.MethodGet, fmt.Sprintf("/assembly/plans/%s", planName), nil)
	require.Equal(t, http.StatusOK, resp.Code)

	var fetched struct {
		Plan map[string]any `json:"plan"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &fetched))
	require.NotEmpty(t, fetched.Plan["models"])

	originalOAIDs := models.ExposedModelIDsByChannel(cfg, backend, "openai")
	originalGMIDs := models.ExposedModelIDsByChannel(cfg, backend, "gemini")
	require.Equal(t, []string{"gpt-4o", "gpt-4o-mini"}, originalOAIDs)
	require.Equal(t, []string{"gemini-2.5-pro"}, originalGMIDs)

	// Workflow: mutate -> preview -> apply -> rollback (repeat 10x to assert idempotence)
	for i := 0; i < 10; i++ {
		driftOA := []models.RegistryEntry{
			{ID: fmt.Sprintf("gpt-4o-mini-%d", i), Base: "gpt-4o-mini", Enabled: true},
		}
		driftGM := []models.RegistryEntry{
			{ID: fmt.Sprintf("gemini-1.5-flash-%d", i), Base: "gemini-1.5-flash", Enabled: true},
		}
		require.NoError(t, backend.SetConfig(ctx, "model_registry_openai", driftOA))
		require.NoError(t, backend.SetConfig(ctx, "model_registry_gemini", driftGM))

		// Preview differences before apply
		resp = client.do(t, http.MethodGet, fmt.Sprintf("/assembly/plans/%s/dry-run/apply", planName), nil)
		require.Equal(t, http.StatusOK, resp.Code)

		var diffResp struct {
			Diff struct {
				OpenAI struct {
					Add    []string `json:"add"`
					Remove []string `json:"remove"`
				} `json:"openai"`
				Gemini struct {
					Add    []string `json:"add"`
					Remove []string `json:"remove"`
				} `json:"gemini"`
			} `json:"diff"`
		}
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &diffResp))
		require.ElementsMatch(t, originalOAIDs, diffResp.Diff.OpenAI.Add)
		require.ElementsMatch(t, []string{driftOA[0].ID}, diffResp.Diff.OpenAI.Remove)
		require.ElementsMatch(t, originalGMIDs, diffResp.Diff.Gemini.Add)
		require.ElementsMatch(t, []string{driftGM[0].ID}, diffResp.Diff.Gemini.Remove)

		// Apply plan to restore canonical snapshot
		resp = client.do(t, http.MethodPut, fmt.Sprintf("/assembly/plans/%s/apply", planName), nil)
		require.Equal(t, http.StatusOK, resp.Code)

		oaIDs := models.ExposedModelIDsByChannel(cfg, backend, "openai")
		gmIDs := models.ExposedModelIDsByChannel(cfg, backend, "gemini")
		require.ElementsMatch(t, originalOAIDs, oaIDs)
		require.ElementsMatch(t, originalGMIDs, gmIDs)

		// Rolling back should restore the drifted state captured before apply
		resp = client.do(t, http.MethodPut, fmt.Sprintf("/assembly/plans/%s/rollback", planName), nil)
		require.Equal(t, http.StatusOK, resp.Code)

		oaIDs = models.ExposedModelIDsByChannel(cfg, backend, "openai")
		gmIDs = models.ExposedModelIDsByChannel(cfg, backend, "gemini")
		require.ElementsMatch(t, []string{driftOA[0].ID}, oaIDs)
		require.ElementsMatch(t, []string{driftGM[0].ID}, gmIDs)
	}

	// Final apply leaves system in canonical plan state
	resp = client.do(t, http.MethodPut, fmt.Sprintf("/assembly/plans/%s/apply", planName), nil)
	require.Equal(t, http.StatusOK, resp.Code)
	oaIDs := models.ExposedModelIDsByChannel(cfg, backend, "openai")
	gmIDs := models.ExposedModelIDsByChannel(cfg, backend, "gemini")
	require.ElementsMatch(t, originalOAIDs, oaIDs)
	require.ElementsMatch(t, originalGMIDs, gmIDs)
}

type mgmtClient struct {
	engine     *gin.Engine
	authBearer string
}

func newMgmtClient(engine *gin.Engine, key string) *mgmtClient {
	return &mgmtClient{
		engine:     engine,
		authBearer: "Bearer " + key,
	}
}

func (c *mgmtClient) do(t *testing.T, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/routes/api/management"+path, body)
	req.Header.Set("Authorization", c.authBearer)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	c.engine.ServeHTTP(rec, req)
	return rec
}

func (c *mgmtClient) doJSON(t *testing.T, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	buf := &bytes.Buffer{}
	if payload != nil {
		require.NoError(t, json.NewEncoder(buf).Encode(payload))
	}
	return c.do(t, method, path, buf)
}
