package management

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/models"
	"gcli2api-go/internal/monitoring"
	store "gcli2api-go/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func canBind() bool {
	if l, err := net.Listen("tcp", "127.0.0.1:0"); err == nil {
		_ = l.Close()
		return true
	}
	if l6, err := net.Listen("tcp6", "[::1]:0"); err == nil {
		_ = l6.Close()
		return true
	}
	return false
}

func TestProbeCredentialsIntegration(t *testing.T) {
	if !canBind() {
		t.Skip("sandbox does not allow binding ports for httptest")
	}
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	writeCredentialFile(t, tmpDir, "success.json", map[string]any{
		"AccessToken": "token-success",
		"ProjectID":   "proj-1",
	})
	writeCredentialFile(t, tmpDir, "failure.json", map[string]any{
		"AccessToken": "token-fail",
		"ProjectID":   "proj-2",
	})

	mgr := credential.NewManager(credential.Options{
		AuthDir: tmpDir,
		AutoBan: credential.AutoBanConfig{Enabled: false},
	})
	require.NoError(t, mgr.LoadCredentials())

	upstreamSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		auth := r.Header.Get("Authorization")
		switch {
		case strings.Contains(auth, "token-success"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"response":{"candidates":[{"content":{"parts":[{"text":"pong"}]}}]}}`))
		case strings.Contains(auth, "token-fail"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"upstream failure"}}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"unknown token"}}`))
		}
	}))
	defer upstreamSrv.Close()

	cfg := &config.Config{
		CodeAssist:   upstreamSrv.URL,
		GoogleProjID: "proj-default",
		AuthDir:      tmpDir,
	}

	handler := NewAdminAPIHandler(cfg, mgr, monitoring.NewEnhancedMetrics(), nil, nil)
	router := gin.New()
	group := router.Group("/routes/api/management")
	handler.RegisterRoutes(group)

	payload := map[string]any{
		"model":       "gemini-2.5-flash",
		"timeout_sec": 5,
	}
	body, _ := json.Marshal(payload)

	before := testutil.ToFloat64(monitoring.AutoProbeRunsTotal.WithLabelValues("manual", "partial", "gemini-2.5-flash"))

	req := httptest.NewRequest(http.MethodPost, "/routes/api/management/credentials/probe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "probe endpoint should succeed")

	var resp struct {
		Model   string `json:"model"`
		Results []struct {
			ID     string `json:"id"`
			OK     bool   `json:"ok"`
			Status int    `json:"status"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "gemini-2.5-flash", resp.Model)
	require.Len(t, resp.Results, 2)

	var successRes, failureRes *struct {
		ID     string `json:"id"`
		OK     bool   `json:"ok"`
		Status int    `json:"status"`
	}
	for i := range resp.Results {
		res := &resp.Results[i]
		switch res.ID {
		case "success.json":
			successRes = res
		case "failure.json":
			failureRes = res
		}
	}
	require.NotNil(t, successRes, "should contain success credential result")
	require.NotNil(t, failureRes, "should contain failure credential result")
	assert.True(t, successRes.OK)
	assert.Equal(t, http.StatusOK, successRes.Status)
	assert.False(t, failureRes.OK)
	assert.Equal(t, http.StatusInternalServerError, failureRes.Status)

	after := testutil.ToFloat64(monitoring.AutoProbeRunsTotal.WithLabelValues("manual", "partial", "gemini-2.5-flash"))
	assert.InDelta(t, before+1, after, 0.0001, "probe run counter should increment")

	handler.probeHistoryMu.Lock()
	require.NotEmpty(t, handler.probeHistory)
	latest := handler.probeHistory[0]
	handler.probeHistoryMu.Unlock()
	assert.Equal(t, "manual", latest.Source)
	assert.Equal(t, 1, latest.Success)
	assert.Equal(t, 2, latest.Total)
	assert.Equal(t, "gemini-2.5-flash", latest.Model)

	creds := mgr.GetAllCredentials()
	credMap := make(map[string]*credential.Credential, len(creds))
	for _, c := range creds {
		credMap[c.ID] = c
	}
	require.Contains(t, credMap, "failure.json")
	require.Contains(t, credMap, "success.json")

	assert.Greater(t, credMap["failure.json"].FailureWeight, 0.0, "failure credential should accumulate failure weight")
	assert.Equal(t, 0.0, credMap["success.json"].FailureWeight, "successful credential should not accumulate failure weight")
	assert.Positive(t, credMap["success.json"].SuccessCount, "success credential should mark success")
	assert.Positive(t, credMap["failure.json"].FailureCount, "failure credential should track failures")
}

func TestUpstreamSuggestWithStoredRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	ctx := context.Background()
	fileBackend := store.NewFileBackend(tmpDir)
	require.NoError(t, fileBackend.Initialize(ctx))

	entries := []models.RegistryEntry{
		{
			ID:       "gemini-2.5-pro",
			Base:     "gemini-2.5-pro",
			Enabled:  true,
			Upstream: "code_assist",
		},
	}
	require.NoError(t, fileBackend.SetConfig(ctx, "model_registry_openai", entries))

	cfg := &config.Config{
		PreferredBaseModels: []string{"gemini-2.5-pro", "gemini-2.5-ultra"},
	}

	handler := NewAdminAPIHandler(cfg, nil, monitoring.NewEnhancedMetrics(), nil, fileBackend)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/routes/api/management/models/upstream-suggest", nil)
	c.Request = req

	handler.UpstreamSuggest(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Bases        []string `json:"bases"`
		Missing      []string `json:"missing"`
		ExistingBase []string `json:"existing_bases"`
		Preferred    []string `json:"preferred"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.Contains(t, resp.Bases, "gemini-2.5-pro")
	assert.Contains(t, resp.Preferred, "gemini-2.5-ultra")
	assert.Contains(t, resp.ExistingBase, "gemini-2.5-pro")
	assert.NotContains(t, resp.Missing, "gemini-2.5-pro", "existing bases should not appear in missing list")
	assert.Contains(t, resp.Missing, "gemini-2.5-ultra", "preferred bases absent from registry should be suggested")
}

func writeCredentialFile(t *testing.T, dir, name string, payload map[string]any) {
	t.Helper()
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0o600))
	time.Sleep(10 * time.Millisecond)
}
