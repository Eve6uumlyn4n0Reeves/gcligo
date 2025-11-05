package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "gcli2api-go/internal/errors"
	"gcli2api-go/internal/httpformat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAbortWithAPIError_OpenAIFormat(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		err            *apperrors.APIError
		expectedStatus int
		expectedType   string
		expectedCode   string
	}{
		{
			name:           "openai_chat_completions_error",
			path:           "/v1/chat/completions",
			err:            apperrors.New(http.StatusBadRequest, "invalid_request_error", "invalid_request_error", "missing model parameter"),
			expectedStatus: http.StatusBadRequest,
			expectedType:   "invalid_request_error",
			expectedCode:   "invalid_request_error",
		},
		{
			name:           "openai_models_error",
			path:           "/v1/models",
			err:            apperrors.New(http.StatusInternalServerError, "server_error", "server_error", "internal error"),
			expectedStatus: http.StatusInternalServerError,
			expectedType:   "server_error",
			expectedCode:   "server_error",
		},
		{
			name:           "openai_completions_error",
			path:           "/v1/completions",
			err:            apperrors.New(http.StatusUnauthorized, "authentication_error", "authentication_error", "invalid api key"),
			expectedStatus: http.StatusUnauthorized,
			expectedType:   "authentication_error",
			expectedCode:   "authentication_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", tt.path, nil)

			AbortWithAPIError(c, tt.err)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.True(t, c.IsAborted())

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// OpenAI format: {"error": {"message": "...", "type": "...", "code": "..."}}
			errorObj, ok := response["error"].(map[string]interface{})
			require.True(t, ok, "response should have 'error' object")

			assert.Equal(t, tt.err.Message, errorObj["message"])
			assert.Equal(t, tt.expectedType, errorObj["type"])
			assert.Equal(t, tt.expectedCode, errorObj["code"])
		})
	}
}

func TestAbortWithAPIError_GeminiFormat(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		err            *apperrors.APIError
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "gemini_generate_content_error",
			path:           "/v1/models/gemini-2.5-pro:generateContent",
			err:            apperrors.New(http.StatusBadRequest, "invalid_request", "invalid_request", "invalid json"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "invalid_request",
		},
		{
			name:           "gemini_stream_generate_error",
			path:           "/v1/models/gemini-2.5-flash:streamGenerateContent",
			err:            apperrors.New(http.StatusTooManyRequests, "rate_limit_exceeded", "rate_limit_exceeded", "quota exceeded"),
			expectedStatus: http.StatusTooManyRequests,
			expectedCode:   "rate_limit_exceeded",
		},
		{
			name:           "gemini_beta_models_error",
			path:           "/v1beta/models",
			err:            apperrors.New(http.StatusServiceUnavailable, "service_unavailable", "service_unavailable", "service temporarily unavailable"),
			expectedStatus: http.StatusServiceUnavailable,
			expectedCode:   "service_unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", tt.path, nil)

			AbortWithAPIError(c, tt.err)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.True(t, c.IsAborted())

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Gemini format: {"error": {"message": "...", "code": "...", "status": "..."}}
			errorObj, ok := response["error"].(map[string]interface{})
			require.True(t, ok, "response should have 'error' object")

			assert.Equal(t, tt.err.Message, errorObj["message"])
			assert.NotEmpty(t, errorObj["code"])
		})
	}
}

func TestAbortWithError(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		status         int
		typ            string
		message        string
		expectedStatus int
	}{
		{
			name:           "bad_request",
			path:           "/v1/chat/completions",
			status:         http.StatusBadRequest,
			typ:            "invalid_request_error",
			message:        "missing required field",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized",
			path:           "/v1/models",
			status:         http.StatusUnauthorized,
			typ:            "authentication_error",
			message:        "invalid credentials",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "empty_message_fallback",
			path:           "/v1/chat/completions",
			status:         http.StatusInternalServerError,
			typ:            "server_error",
			message:        "",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", tt.path, nil)

			AbortWithError(c, tt.status, tt.typ, tt.message)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.True(t, c.IsAborted())

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			errorObj, ok := response["error"].(map[string]interface{})
			require.True(t, ok)

			if tt.message != "" {
				assert.Equal(t, tt.message, errorObj["message"])
			} else {
				// Should fallback to "internal error"
				assert.Equal(t, "internal error", errorObj["message"])
			}
		})
	}
}

func TestAbortWithUpstreamError(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		status         int
		typ            string
		message        string
		upstream       []byte
		expectUpstream bool
	}{
		{
			name:           "with_json_upstream",
			path:           "/v1/chat/completions",
			status:         http.StatusBadGateway,
			typ:            "upstream_error",
			message:        "upstream failed",
			upstream:       []byte(`{"error": "upstream service error"}`),
			expectUpstream: true,
		},
		{
			name:           "with_text_upstream",
			path:           "/v1/chat/completions",
			status:         http.StatusBadGateway,
			typ:            "upstream_error",
			message:        "upstream failed",
			upstream:       []byte("plain text error"),
			expectUpstream: true,
		},
		{
			name:           "without_upstream",
			path:           "/v1/chat/completions",
			status:         http.StatusBadGateway,
			typ:            "upstream_error",
			message:        "upstream failed",
			upstream:       nil,
			expectUpstream: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", tt.path, nil)

			AbortWithUpstreamError(c, tt.status, tt.typ, tt.message, tt.upstream)

			assert.Equal(t, tt.status, w.Code)
			assert.True(t, c.IsAborted())

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			errorObj, ok := response["error"].(map[string]interface{})
			require.True(t, ok)

			assert.Equal(t, tt.message, errorObj["message"])

			if tt.expectUpstream {
				// Should have upstream details
				assert.NotNil(t, errorObj["details"])
			}
		})
	}
}

func TestDetectErrorFormat(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedFormat apperrors.ErrorFormat
	}{
		{
			name:           "openai_chat_completions",
			path:           "/v1/chat/completions",
			expectedFormat: apperrors.FormatOpenAI,
		},
		{
			name:           "openai_models",
			path:           "/v1/models",
			expectedFormat: apperrors.FormatOpenAI,
		},
		{
			name:           "gemini_generate_content",
			path:           "/v1/models/gemini-2.5-pro:generateContent",
			expectedFormat: apperrors.FormatGemini,
		},
		{
			name:           "gemini_stream_generate",
			path:           "/v1/models/gemini-2.5-flash:streamGenerateContent",
			expectedFormat: apperrors.FormatGemini,
		},
		{
			name:           "gemini_beta",
			path:           "/v1beta/models",
			expectedFormat: apperrors.FormatGemini,
		},
		{
			name:           "gemini_internal",
			path:           "/v1internal/models",
			expectedFormat: apperrors.FormatGemini,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", tt.path, nil)

			format := httpformat.DetectFromContext(c)
			assert.Equal(t, tt.expectedFormat, format)
		})
	}
}

func TestAbortWithAPIError_NilError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)

	// Should handle nil error gracefully
	AbortWithAPIError(c, nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.True(t, c.IsAborted())

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorObj, ok := response["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "unknown error", errorObj["message"])
}
