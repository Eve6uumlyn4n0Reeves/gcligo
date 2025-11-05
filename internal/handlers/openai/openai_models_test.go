package openai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_ListModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("list models with default config", func(t *testing.T) {
		cfg := &config.Config{
			PreferredBaseModels: []string{"gemini-2.0-flash-exp", "gemini-1.5-pro"},
		}
		handler := &Handler{
			cfg: cfg,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/models", nil)

		handler.ListModels(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "list", response["object"])
		data, ok := response["data"].([]interface{})
		require.True(t, ok)
		assert.Greater(t, len(data), 0, "should have at least one model")
	})

	t.Run("list models with variants enabled", func(t *testing.T) {
		cfg := &config.Config{
			PreferredBaseModels:  []string{"gemini-2.0-flash-exp"},
			DisableModelVariants: false, // explicitly enable variants
		}
		handler := &Handler{
			cfg: cfg,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/models", nil)

		handler.ListModels(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].([]interface{})
		require.True(t, ok)

		// With variants enabled, should have multiple variants of the base model
		assert.Greater(t, len(data), 1, "should have multiple model variants")

		// Check that variants are present
		modelIDs := make([]string, 0)
		for _, item := range data {
			model := item.(map[string]interface{})
			modelIDs = append(modelIDs, model["id"].(string))
		}

		// Should have some variants like fake-, anti-, -maxthinking, -search, etc.
		hasVariant := false
		for _, id := range modelIDs {
			if id != "gemini-2.0-flash-exp" {
				hasVariant = true
				break
			}
		}
		assert.True(t, hasVariant, "should have at least one variant")
	})

	t.Run("list models with variants disabled", func(t *testing.T) {
		cfg := &config.Config{
			PreferredBaseModels:  []string{"gemini-2.0-flash-exp"},
			DisableModelVariants: true,
		}
		handler := &Handler{
			cfg: cfg,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/models", nil)

		handler.ListModels(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].([]interface{})
		require.True(t, ok)

		// With variants disabled, should only have base models (at least 1)
		assert.GreaterOrEqual(t, len(data), 1, "should have at least one base model")

		// Verify first model has expected structure
		model := data[0].(map[string]interface{})
		assert.NotEmpty(t, model["id"])
		assert.Equal(t, "model", model["object"])
	})

	t.Run("list models with disabled models filter", func(t *testing.T) {
		cfg := &config.Config{
			PreferredBaseModels:  []string{"gemini-2.0-flash-exp", "gemini-1.5-pro"},
			DisabledModels:       []string{"gemini-1.5-pro"},
			DisableModelVariants: true, // disable variants for simpler test
		}
		handler := &Handler{
			cfg: cfg,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/models", nil)

		handler.ListModels(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].([]interface{})
		require.True(t, ok)

		// Should have at least one model, and gemini-1.5-pro should not be in the list
		assert.GreaterOrEqual(t, len(data), 1)

		// Verify gemini-1.5-pro is not in the list
		for _, item := range data {
			model := item.(map[string]interface{})
			assert.NotEqual(t, "gemini-1.5-pro", model["id"], "disabled model should not appear")
		}
	})

	t.Run("list models with empty preferred models uses defaults", func(t *testing.T) {
		cfg := &config.Config{
			PreferredBaseModels:  []string{}, // empty
			DisableModelVariants: true,
		}
		handler := &Handler{
			cfg: cfg,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/models", nil)

		handler.ListModels(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].([]interface{})
		require.True(t, ok)

		// Should use default models (at least one)
		assert.GreaterOrEqual(t, len(data), 1, "should have at least one default model")
	})

	t.Run("model response structure", func(t *testing.T) {
		cfg := &config.Config{
			PreferredBaseModels:  []string{"gemini-2.0-flash-exp"},
			DisableModelVariants: true,
		}
		handler := &Handler{
			cfg: cfg,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/models", nil)

		handler.ListModels(c)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].([]interface{})
		require.Greater(t, len(data), 0, "should have at least one model")
		model := data[0].(map[string]interface{})

		// Verify model structure
		assert.NotEmpty(t, model["id"])
		assert.Equal(t, "model", model["object"])
		assert.Equal(t, "gcli2api-go", model["owned_by"])
		assert.NotNil(t, model["created"])
		assert.NotNil(t, model["modalities"])
		assert.NotNil(t, model["capabilities"])
		assert.Equal(t, float64(1048576), model["context_length"])
	})
}

func TestHandler_GetModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("get existing model", func(t *testing.T) {
		cfg := &config.Config{}
		handler := &Handler{
			cfg: cfg,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: "gemini-2.0-flash-exp"}}
		c.Request = httptest.NewRequest("GET", "/v1/models/gemini-2.0-flash-exp", nil)

		handler.GetModel(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "gemini-2.0-flash-exp", response["id"])
		assert.Equal(t, "model", response["object"])
		assert.Equal(t, "gcli2api-go", response["owned_by"])
	})

	t.Run("get model with variant", func(t *testing.T) {
		cfg := &config.Config{}
		handler := &Handler{
			cfg: cfg,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: "fake-gemini-2.0-flash-exp"}}
		c.Request = httptest.NewRequest("GET", "/v1/models/fake-gemini-2.0-flash-exp", nil)

		handler.GetModel(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "fake-gemini-2.0-flash-exp", response["id"])
	})

	t.Run("get model with missing id returns error", func(t *testing.T) {
		cfg := &config.Config{}
		handler := &Handler{
			cfg: cfg,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: ""}}
		c.Request = httptest.NewRequest("GET", "/v1/models/", nil)

		handler.GetModel(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get image model has correct modalities", func(t *testing.T) {
		cfg := &config.Config{}
		handler := &Handler{
			cfg: cfg,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: "gemini-2.0-flash-image-exp"}}
		c.Request = httptest.NewRequest("GET", "/v1/models/gemini-2.0-flash-image-exp", nil)

		handler.GetModel(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		modalities := response["modalities"].([]interface{})
		assert.Contains(t, modalities, "image")
		assert.Contains(t, modalities, "text")

		capabilities := response["capabilities"].(map[string]interface{})
		assert.True(t, capabilities["images"].(bool))
	})
}

func TestMapFinishReason(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"max tokens", "MAX_TOKENS", "length"},
		{"stop", "STOP", "stop"},
		{"stopped", "STOPPED", "stop"},
		{"safety", "SAFETY", "content_filter"},
		{"blocklist", "BLOCKLIST", "content_filter"},
		{"prohibited content", "PROHIBITED_CONTENT", "content_filter"},
		{"recitation", "RECITATION", "content_filter"},
		{"unknown defaults to stop", "UNKNOWN", "stop"},
		{"empty defaults to stop", "", "stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapFinishReason(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	t.Run("contains existing element", func(t *testing.T) {
		arr := []string{"text", "image", "audio"}
		assert.True(t, contains(arr, "image"))
	})

	t.Run("does not contain non-existing element", func(t *testing.T) {
		arr := []string{"text", "image"}
		assert.False(t, contains(arr, "audio"))
	})

	t.Run("case insensitive match", func(t *testing.T) {
		arr := []string{"text", "IMAGE"}
		assert.True(t, contains(arr, "image"))
		assert.True(t, contains(arr, "Image"))
	})

	t.Run("empty array", func(t *testing.T) {
		arr := []string{}
		assert.False(t, contains(arr, "text"))
	})

	t.Run("empty string", func(t *testing.T) {
		arr := []string{"text", ""}
		assert.True(t, contains(arr, ""))
	})
}
