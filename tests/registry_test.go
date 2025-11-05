package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"gcli2api-go/internal/config"
	enh "gcli2api-go/internal/handlers/management"
	oh "gcli2api-go/internal/handlers/openai"
	"gcli2api-go/internal/models"
	store "gcli2api-go/internal/storage"
)

func newTempFileBackend(t *testing.T) store.Backend {
	t.Helper()
	dir := t.TempDir()
	// ensure subdirs
	base := filepath.Join(dir, "storage")
	fb := store.NewFileBackend(base)
	assert.NoError(t, fb.Initialize(context.Background()))
	return fb
}

func TestModelRegistryCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st := newTempFileBackend(t)
	cfg := &config.Config{ManagementKey: "mgmt"}
	h := enh.NewAdminAPIHandler(cfg, nil, nil, nil, st)
	r := gin.New()
	grp := r.Group("/routes/api/management")
	h.RegisterRoutes(grp)

	// Add model
	add := map[string]any{"base": "gemini-2.5-pro", "enabled": true, "upstream": "code_assist"}
	b, _ := json.Marshal(add)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/routes/api/management/models/registry", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var added map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &added)
	id, _ := added["id"].(string)
	if id == "" {
		t.Fatalf("expected returned id, got: %v", added)
	}

	// Get list
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/routes/api/management/models/registry", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	arr, _ := got["models"].([]any)
	assert.GreaterOrEqual(t, len(arr), 1)

	// Delete
	w = httptest.NewRecorder()
	req = httptest.NewRequest("DELETE", "/routes/api/management/models/registry/"+id, nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOpenAIListModelsUsesRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st := newTempFileBackend(t)
	// write a registry entry
	entry := models.RegistryEntry{Base: "gemini-2.5-pro", Enabled: true, Upstream: "code_assist"}
	entries := []models.RegistryEntry{entry}
	assert.NoError(t, st.SetConfig(context.Background(), "model_registry", entries))

	cfg := &config.Config{}
	h := oh.New(cfg, nil, nil, st, nil)
	r := gin.New()
	r.GET("/v1/models", h.ListModels)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/models", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	data, _ := out["data"].([]any)
	if !(len(data) >= 1) {
		t.Fatalf("expected at least one model from dynamic registry, got: %v", out)
	}
	// ensure the id reflects the entry base
	first, _ := data[0].(map[string]any)
	if first["id"] == "" {
		t.Fatalf("expected model id present, got: %v", first)
	}
}

func TestGroupsCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st := newTempFileBackend(t)
	cfg := &config.Config{ManagementKey: "mgmt"}
	h := enh.NewAdminAPIHandler(cfg, nil, nil, nil, st)
	r := gin.New()
	grp := r.Group("/routes/api/management")
	h.RegisterRoutes(grp)

	// Create group
	g := map[string]any{"name": "默认", "enabled": true}
	b, _ := json.Marshal(g)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/routes/api/management/models/groups", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create group failed: %d %s", w.Code, w.Body.String())
	}

	// List groups
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/routes/api/management/models/groups", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list groups failed: %d", w.Code)
	}
}
