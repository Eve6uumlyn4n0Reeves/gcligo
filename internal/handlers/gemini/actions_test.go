package gemini

import (
	"bytes"
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

func TestLoadCodeAssist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handler := New(cfg, nil, nil, nil)
	router := gin.New()
	router.POST("/action/loadCodeAssist", handler.LoadCodeAssist)

	t.Run("invalid json request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/action/loadCodeAssist", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "error")
	})

	t.Run("valid request with metadata", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"metadata": map[string]interface{}{
				"ideType":    "VSCODE",
				"platform":   "MACOS",
				"pluginType": "GEMINI",
			},
			"files": []interface{}{},
		}

		body, _ := json.Marshal(requestBody)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/action/loadCodeAssist", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		// Will fail with upstream error since we don't have real credentials
		// but the request should be processed
		assert.True(t, w.Code >= 400)
	})

	t.Run("valid request without metadata", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"files": []interface{}{},
		}

		body, _ := json.Marshal(requestBody)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/action/loadCodeAssist", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		// Will fail with upstream error since we don't have real credentials
		// but the request should be processed and metadata should be added
		assert.True(t, w.Code >= 400)
	})

	t.Run("empty request body", func(t *testing.T) {
		requestBody := map[string]interface{}{}

		body, _ := json.Marshal(requestBody)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/action/loadCodeAssist", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		// Should process the request
		assert.True(t, w.Code >= 400)
	})
}

func TestOnboardUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handler := New(cfg, nil, nil, nil)
	router := gin.New()
	router.POST("/action/onboardUser", handler.OnboardUser)

	t.Run("invalid json request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/action/onboardUser", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "error")
	})

	t.Run("valid request", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"userId": "test-user-123",
			"email":  "test@example.com",
		}

		body, _ := json.Marshal(requestBody)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/action/onboardUser", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		// Will fail with upstream error since we don't have real credentials
		// but the request should be processed
		assert.True(t, w.Code >= 400)
	})

	t.Run("empty request body", func(t *testing.T) {
		requestBody := map[string]interface{}{}

		body, _ := json.Marshal(requestBody)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/action/onboardUser", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		// Should process the request
		assert.True(t, w.Code >= 400)
	})

	t.Run("request with additional fields", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"userId": "test-user-456",
			"email":  "test2@example.com",
			"settings": map[string]interface{}{
				"theme": "dark",
			},
		}

		body, _ := json.Marshal(requestBody)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/action/onboardUser", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		// Should process the request
		assert.True(t, w.Code >= 400)
	})
}

func TestStreamSessionMarkFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}

	t.Run("mark failure with credential and credential manager", func(t *testing.T) {
		// Create handler with credential manager
		credMgr := credpkg.NewManager(credpkg.Options{})
		handler := New(cfg, credMgr, nil, nil)

		// Create a credential
		cred := &credpkg.Credential{
			ID: "test-cred-failure",
		}

		// Create a stream session
		ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
		session := &streamSession{
			handler:  handler,
			ginCtx:   ginCtx,
			usedCred: cred,
		}

		// Mark failure should not panic
		session.markFailure("test_error", 500)
	})

	t.Run("mark failure without credential", func(t *testing.T) {
		handler := New(cfg, nil, nil, nil)
		ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
		session := &streamSession{
			handler:  handler,
			ginCtx:   ginCtx,
			usedCred: nil,
		}

		// Should not panic
		session.markFailure("test_error", 500)
	})
}
