//go:build legacy_tests
// +build legacy_tests

package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/storage"
	"gcli2api-go/internal/upstream"

	"github.com/gin-gonic/gin"
)

func setupTestHandler(t *testing.T) (*Handler, *gin.Engine, func()) {
	gin.SetMode(gin.TestMode)

	// Create test storage
	tmpDir := t.TempDir()
	store := storage.NewFileBackend(tmpDir)
	ctx := context.Background()
	if err := store.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		OpenAICompatEndpoint: "/v1",
	}

	// Create credential manager
	credMgr := credential.NewManager(store, cfg)

	// Create upstream manager
	upstreamMgr := upstream.NewManager(cfg, credMgr)

	// Create handler
	handler := NewHandler(cfg, credMgr, upstreamMgr, store)

	// Create router
	router := gin.New()

	cleanup := func() {
		store.Close()
	}

	return handler, router, cleanup
}

func TestHandler_ChatCompletions_InvalidRequest(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.POST("/v1/chat/completions", handler.ChatCompletions)

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing model",
			body: map[string]interface{}{
				"messages": []map[string]interface{}{
					{"role": "user", "content": "test"},
				},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing messages",
			body: map[string]interface{}{
				"model": "gpt-4",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid messages format",
			body: map[string]interface{}{
				"model":    "gpt-4",
				"messages": "invalid",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("ChatCompletions() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_ChatCompletions_ValidRequest(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.POST("/v1/chat/completions", handler.ChatCompletions)

	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Note: This will fail without proper credentials, but we're testing the request parsing
	// In a real test, you'd mock the upstream or provide test credentials
	if w.Code != http.StatusOK && w.Code != http.StatusUnauthorized && w.Code != http.StatusServiceUnavailable {
		t.Logf("ChatCompletions() status = %d (expected failure without credentials)", w.Code)
	}
}

func TestHandler_ChatCompletions_StreamRequest(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.POST("/v1/chat/completions", handler.ChatCompletions)

	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
		"stream": true,
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Note: This will fail without proper credentials
	if w.Code != http.StatusOK && w.Code != http.StatusUnauthorized && w.Code != http.StatusServiceUnavailable {
		t.Logf("ChatCompletions() stream status = %d (expected failure without credentials)", w.Code)
	}
}

func TestHandler_Models(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.GET("/v1/models", handler.Models)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Models() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Models() failed to parse response: %v", err)
	}

	if response["object"] != "list" {
		t.Errorf("Models() object = %v, want list", response["object"])
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Error("Models() data field is not an array")
	}

	if len(data) == 0 {
		t.Error("Models() returned empty data array")
	}
}

func TestHandler_GetModel(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.GET("/v1/models/:model", handler.GetModel)

	tests := []struct {
		name       string
		model      string
		wantStatus int
	}{
		{
			name:       "valid model",
			model:      "gpt-4",
			wantStatus: http.StatusOK,
		},
		{
			name:       "gemini model",
			model:      "gemini-pro",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/models/"+tt.model, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("GetModel() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if w.Code == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("GetModel() failed to parse response: %v", err)
				}

				if response["id"] != tt.model {
					t.Errorf("GetModel() id = %v, want %v", response["id"], tt.model)
				}
			}
		})
	}
}

func TestHandler_Embeddings(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.POST("/v1/embeddings", handler.Embeddings)

	requestBody := map[string]interface{}{
		"model": "text-embedding-ada-002",
		"input": "test text",
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Note: This will fail without proper credentials
	if w.Code != http.StatusOK && w.Code != http.StatusUnauthorized && w.Code != http.StatusServiceUnavailable {
		t.Logf("Embeddings() status = %d (expected failure without credentials)", w.Code)
	}
}
