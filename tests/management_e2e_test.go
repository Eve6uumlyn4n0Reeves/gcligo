package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	mw "gcli2api-go/internal/middleware"
	"gcli2api-go/internal/models"
	"gcli2api-go/internal/monitoring"
	srv "gcli2api-go/internal/server"
	store "gcli2api-go/internal/storage"
	route "gcli2api-go/internal/upstream/strategy"
)

func buildEngineWithConfig(t *testing.T, cfg *config.Config) (*gin.Engine, store.Backend, *route.Strategy) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	fb := store.NewFileBackend(storageDir)
	require.NoError(t, fb.Initialize(context.Background()))

	if cfg.Security.AuthDir == "" {
		cfg.Security.AuthDir = filepath.Join(root, "auth")
	}
	require.NoError(t, os.MkdirAll(cfg.Security.AuthDir, 0o755))
	cfg.SyncFromDomains()

	mgr := credential.NewManager(credential.Options{AuthDir: cfg.Security.AuthDir})
	require.NoError(t, mgr.LoadCredentials())

	deps := srv.Dependencies{
		CredentialManager: mgr,
		Storage:           fb,
		EnhancedMetrics:   monitoring.NewEnhancedMetrics(),
	}

	openaiEngine, _, strategy := srv.BuildEngines(cfg, deps)
	return openaiEngine, fb, strategy
}

func buildManagementEngine(t *testing.T) (*gin.Engine, *config.Config, store.Backend, *route.Strategy) {
	cfg := &config.Config{}
	cfg.Server.BasePath = ""
	cfg.Server.WebAdminEnabled = true
	cfg.Security.ManagementKey = "secret"
	cfg.Security.ManagementAllowRemote = true
	openaiEngine, fb, strategy := buildEngineWithConfig(t, cfg)
	return openaiEngine, cfg, fb, strategy
}

func doJSON(t *testing.T, engine *gin.Engine, cfg *config.Config, method, path string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+cfg.ManagementKey)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func doRequest(t *testing.T, engine *gin.Engine, cfg *config.Config, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+cfg.ManagementKey)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

// 覆盖装配计划完整生命周期：保存->查询->应用->回滚->删除（通过管理端点）
func TestAssemblyPlanLifecycleEndpoints(t *testing.T) {
	engine, cfg, st, _ := buildManagementEngine(t)

	ctx := context.Background()
	original := []models.RegistryEntry{{ID: "orig", Base: "gemini-2.5-pro", Enabled: true}}
	require.NoError(t, st.SetConfig(ctx, "model_registry_openai", original))

	planBody := map[string]any{
		"name": "blueprint",
		"include": map[string]bool{
			"models":   true,
			"variants": true,
		},
	}
	rec := doJSON(t, engine, cfg, http.MethodPost, "/routes/api/management/assembly/plans", planBody)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 修改当前 registry 以便应用计划时能恢复
	modified := []models.RegistryEntry{{ID: "draft", Base: "gemini-2.5-flash", Enabled: false}}
	require.NoError(t, st.SetConfig(ctx, "model_registry_openai", modified))

	// dry-run apply
	rec = doRequest(t, engine, cfg, http.MethodGet, "/routes/api/management/assembly/plans/blueprint/dry-run/apply")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// inline dry-run
	planResp := doRequest(t, engine, cfg, http.MethodGet, "/routes/api/management/assembly/plans/blueprint")
	require.Equal(t, http.StatusOK, planResp.Code, planResp.Body.String())
	var planPayload map[string]any
	require.NoError(t, json.Unmarshal(planResp.Body.Bytes(), &planPayload))
	planData := planPayload["plan"].(map[string]any)
	rec = doJSON(t, engine, cfg, http.MethodPost, "/routes/api/management/assembly/dry-run", map[string]any{"plan": planData})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// apply plan -> 恢复为 original
	assertApplyIsIdempotent := func(iterations int) {
		for i := 0; i < iterations; i++ {
			rec := doRequest(t, engine, cfg, http.MethodPut, "/routes/api/management/assembly/plans/blueprint/apply")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

			diffRec := doRequest(t, engine, cfg, http.MethodGet, "/routes/api/management/assembly/plans/blueprint/dry-run/apply")
			require.Equal(t, http.StatusOK, diffRec.Code, diffRec.Body.String())
			assertNoDiff(t, diffRec.Body.Bytes(), "openai", "gemini")
		}
	}
	assertApplyIsIdempotent(10)

	// dry-run rollback
	rec = doRequest(t, engine, cfg, http.MethodGet, "/routes/api/management/assembly/plans/blueprint/dry-run/rollback")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assertNoDiff(t, rec.Body.Bytes(), "openai", "gemini")

	// rollback plan -> 恢复此前备份（modified）
	assertRollbackIsIdempotent := func(iterations int) {
		for i := 0; i < iterations; i++ {
			rec := doRequest(t, engine, cfg, http.MethodPut, "/routes/api/management/assembly/plans/blueprint/rollback")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

			diffRec := doRequest(t, engine, cfg, http.MethodGet, "/routes/api/management/assembly/plans/blueprint/dry-run/rollback")
			require.Equal(t, http.StatusOK, diffRec.Code, diffRec.Body.String())
			assertNoDiff(t, diffRec.Body.Bytes(), "openai", "gemini")
		}
	}
	assertRollbackIsIdempotent(10)

	// Delete plan
	rec = doRequest(t, engine, cfg, http.MethodDelete, "/routes/api/management/assembly/plans/blueprint")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// 验证路由状态持久化与恢复端点，确保 cooldown 信息可 round-trip
func TestRoutingPersistRestoreEndpoints(t *testing.T) {
	engine, cfg, st, strategy := buildManagementEngine(t)

	require.NotNil(t, strategy)
	const credID = "cred-routing-e2e"
	strategy.SetCooldown(credID, 2, time.Now().Add(time.Minute))
	_, existing := strategy.Snapshot()
	require.NotEmpty(t, existing, "expected cooldown snapshot populated")

	// persist state
	rec := doRequest(t, engine, cfg, http.MethodPost, "/routes/api/management/routing/persist")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// state saved in storage
	raw, err := st.GetConfig(context.Background(), "routing_state")
	require.NoError(t, err)
	_, ok := raw.(map[string]any)
	require.True(t, ok)

	// clear current cooldowns to ensure restore repopulates
	strategy.ClearCooldown(credID)

	// restore state
	rec = doRequest(t, engine, cfg, http.MethodPost, "/routes/api/management/routing/restore")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	// 再次设置冷却用于测试新清理接口
	strategy.SetCooldown(credID, 1, time.Now().Add(10*time.Second))

	// clear via new endpoint
	rec = doJSON(t, engine, cfg, http.MethodPost, "/routes/api/management/assembly/cooldowns/clear", map[string]any{"credentials": []string{credID}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var clearPayload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &clearPayload))
	clearedList := make([]string, 0)
	if arr, ok := clearPayload["cleared"].([]any); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				clearedList = append(clearedList, s)
			}
		}
	}
	require.Contains(t, clearedList, credID)
	require.Equal(t, 0, toInt(clearPayload["total_after"]))

	// cleanup for other tests
	strategy.ClearCooldown(credID)
}

// TestStreamingMetricsRecorded verifies that SSE metrics are recorded via Prometheus
func TestStreamingMetricsRecorded(t *testing.T) {
	engine, _, _, _ := buildManagementEngine(t)

	const path = "/v1/chat/test-stream"
	mw.RecordSSELines("openai", path, 3)
	mw.RecordToolCalls("openai", path, 1)
	mw.RecordSSEClose("openai", path, "client_disconnect")

	// Verify metrics are exposed via Prometheus /metrics endpoint (now uses promhttp)
	// The /metrics endpoint returns Prometheus text format, not JSON
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "gcli2api_sse_lines_total")
	require.Contains(t, rec.Body.String(), "gcli2api_tool_calls_total")
	require.Contains(t, rec.Body.String(), "gcli2api_sse_disconnects_total")
}

func metricEntryExists(v any, path string) bool {
	found := false
	forEachMetricEntry(v, func(m map[string]any) {
		if m["path"] == path {
			found = true
		}
	})
	return found
}

func extractMetricCount(v any, path string) int {
	result := 0
	forEachMetricEntry(v, func(m map[string]any) {
		if m["path"] == path {
			result = toInt(m["count"])
		}
	})
	return result
}

func forEachMetricEntry(v any, fn func(map[string]any)) {
	switch entries := v.(type) {
	case []map[string]any:
		for _, m := range entries {
			if m != nil {
				fn(m)
			}
		}
	case []any:
		for _, entry := range entries {
			if m, ok := entry.(map[string]any); ok && m != nil {
				fn(m)
			}
		}
	}
}

func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

func getDiffList(payload map[string]any, channel, kind string) []string {
	diff, _ := payload["diff"].(map[string]any)
	if diff == nil {
		return nil
	}
	channelMap, _ := diff[channel].(map[string]any)
	if channelMap == nil {
		return nil
	}
	raw, _ := channelMap[kind].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func assertNoDiff(t *testing.T, body []byte, channels ...string) {
	t.Helper()
	var diffPayload map[string]any
	require.NoError(t, json.Unmarshal(body, &diffPayload))
	if len(channels) == 0 {
		channels = []string{"openai", "gemini"}
	}
	for _, channel := range channels {
		require.Zero(t, len(getDiffList(diffPayload, channel, "added")), "channel=%s should have no additions", channel)
		require.Zero(t, len(getDiffList(diffPayload, channel, "removed")), "channel=%s should have no removals", channel)
	}
}
