package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gcli2api-go/internal/config"
	credpkg "gcli2api-go/internal/credential"
)

func setupTestHandler(t *testing.T) (*Handler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handler := New(cfg, nil, nil, nil)

	router := gin.New()
	return handler, router
}

func TestModels(t *testing.T) {
	handler, router := setupTestHandler(t)
	router.GET("/v1/models", handler.Models)

	t.Run("list models successfully", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/models", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "models")
		models, ok := response["models"].([]interface{})
		assert.True(t, ok)
		assert.GreaterOrEqual(t, len(models), 1)

		// Check first model structure
		if len(models) > 0 {
			model, ok := models[0].(map[string]interface{})
			assert.True(t, ok)
			assert.Contains(t, model, "name")
			assert.Contains(t, model, "baseModelId")
			assert.Contains(t, model, "version")
			assert.Contains(t, model, "displayName")
			assert.Contains(t, model, "description")
			assert.Contains(t, model, "inputTokenLimit")
			assert.Contains(t, model, "outputTokenLimit")
			assert.Contains(t, model, "supportedGenerationMethods")
		}
	})
}

func TestModelInfo(t *testing.T) {
	handler, router := setupTestHandler(t)
	router.GET("/v1/models/:model", handler.ModelInfo)

	t.Run("get model info successfully", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/models/gemini-2.0-flash-exp", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "name")
		assert.Contains(t, response["name"], "gemini-2.0-flash-exp")
		assert.Equal(t, "001", response["version"])
		assert.Equal(t, "gemini-2.0-flash-exp", response["displayName"])
		assert.Contains(t, response["description"], "Gemini model")
		assert.Equal(t, float64(1048576), response["inputTokenLimit"])
		assert.Equal(t, float64(8192), response["outputTokenLimit"])

		methods, ok := response["supportedGenerationMethods"].([]interface{})
		assert.True(t, ok)
		assert.Contains(t, methods, "generateContent")
		assert.Contains(t, methods, "streamGenerateContent")
		assert.Contains(t, methods, "countTokens")
	})

	t.Run("get model info for different model", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/models/gemini-1.5-pro", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["name"], "gemini-1.5-pro")
		assert.Equal(t, "gemini-1.5-pro", response["displayName"])
	})
}

func TestListModels(t *testing.T) {
	handler, router := setupTestHandler(t)
	router.GET("/v1/models", handler.ListModels)

	t.Run("list models delegates to Models", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/models", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "models")
	})
}

func TestGetModel(t *testing.T) {
	handler, router := setupTestHandler(t)
	router.GET("/v1/models/:model", handler.GetModel)

	t.Run("get model delegates to ModelInfo", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/models/gemini-2.0-flash-exp", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		_ = err

		assert.Contains(t, response, "name")
	})
}

func TestInvalidateCacheFor(t *testing.T) {
	handler, _ := setupTestHandler(t)

	t.Run("invalidate cache for credential", func(t *testing.T) {
		// Add a credential to cache
		handler.clientCache["test-cred-1"] = handler.cl

		// Verify it's in cache
		handler.cacheMu.RLock()
		_, exists := handler.clientCache["test-cred-1"]
		handler.cacheMu.RUnlock()
		assert.True(t, exists)

		// Invalidate cache
		handler.InvalidateCacheFor("test-cred-1")

		// Verify it's removed
		handler.cacheMu.RLock()
		_, exists = handler.clientCache["test-cred-1"]
		handler.cacheMu.RUnlock()
		assert.False(t, exists)
	})

	t.Run("invalidate cache with empty credential ID", func(t *testing.T) {
		// Should not panic
		handler.InvalidateCacheFor("")
	})

	t.Run("invalidate cache for non-existent credential", func(t *testing.T) {
		// Should not panic
		handler.InvalidateCacheFor("non-existent-cred")
	})
}

func TestGetClientFor(t *testing.T) {
	handler, _ := setupTestHandler(t)

	t.Run("get client for nil credential", func(t *testing.T) {
		client := handler.getClientFor(nil)
		assert.NotNil(t, client)
		assert.Equal(t, handler.cl, client)
	})

	t.Run("get client for credential with empty ID", func(t *testing.T) {
		cred := &credpkg.Credential{ID: ""}
		client := handler.getClientFor(cred)
		assert.NotNil(t, client)
		assert.Equal(t, handler.cl, client)
	})

	t.Run("get client for valid credential creates and caches", func(t *testing.T) {
		cred := &credpkg.Credential{
			ID:          "test-cred-2",
			AccessToken: "test-token",
		}

		// First call should create and cache
		client1 := handler.getClientFor(cred)
		assert.NotNil(t, client1)

		// Verify it's cached
		handler.cacheMu.RLock()
		cached, exists := handler.clientCache["test-cred-2"]
		handler.cacheMu.RUnlock()
		assert.True(t, exists)
		assert.Equal(t, client1, cached)

		// Second call should return cached client
		client2 := handler.getClientFor(cred)
		assert.Equal(t, client1, client2)
	})
}

func TestGetUpstreamClient(t *testing.T) {
	handler, _ := setupTestHandler(t)

	t.Run("get upstream client without credential manager", func(t *testing.T) {
		handler.credMgr = nil
		handler.router = nil

		ctx := context.Background()
		client, cred := handler.getUpstreamClient(ctx)

		assert.NotNil(t, client)
		assert.Nil(t, cred)
		assert.Equal(t, handler.cl, client)
	})

	t.Run("get upstream client with credential manager but no router", func(t *testing.T) {
		handler.credMgr = credpkg.NewManager(credpkg.Options{})
		handler.router = nil

		ctx := context.Background()
		client, _ := handler.getUpstreamClient(ctx)

		assert.NotNil(t, client)
		// cred may be nil if no credentials available
	})
}

func TestShouldRefreshAhead(t *testing.T) {
	handler, _ := setupTestHandler(t)

	t.Run("should refresh ahead for credential", func(t *testing.T) {
		cred := &credpkg.Credential{
			ID:          "test-cred",
			AccessToken: "test-token",
		}

		// Should not panic
		result := handler.shouldRefreshAhead(cred)
		assert.IsType(t, false, result)
	})
}

func TestNewWithStrategy(t *testing.T) {
	cfg := &config.Config{}

	t.Run("create handler with nil strategy", func(t *testing.T) {
		handler := NewWithStrategy(cfg, nil, nil, nil, nil)
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.router)
	})

	t.Run("create handler with existing strategy", func(t *testing.T) {
		// Create a strategy first
		handler1 := New(cfg, nil, nil, nil)
		strategy := handler1.router

		// Create handler with existing strategy
		handler2 := NewWithStrategy(cfg, nil, nil, nil, strategy)
		assert.NotNil(t, handler2)
		assert.Equal(t, strategy, handler2.router)
	})
}
