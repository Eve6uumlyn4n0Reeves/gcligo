package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func invokeGenerateContent(t *testing.T, handler *Handler, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-pro:generateContent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "model", Value: "gemini-2.5-pro"}}
	handler.GenerateContent(c)
	return w
}

func invokeCountTokens(t *testing.T, handler *Handler, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-pro:countTokens", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "model", Value: "gemini-2.5-pro"}}
	handler.CountTokens(c)
	return w
}

func TestGenerateContent_SuccessWithFallback(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{GoogleProjID: "proj-123"}
	attempts := make([]string, 0, 2)
	stub := &stubUpstream{
		generateFunc: func(_ context.Context, payload []byte) (*http.Response, error) {
			var req map[string]any
			_ = json.Unmarshal(payload, &req)
			if m, ok := req["model"].(string); ok {
				attempts = append(attempts, m)
			}
			if len(attempts) == 1 {
				return newHTTPResponse(http.StatusInternalServerError, []byte(`{"error":"fail"}`)), nil
			}
			resp := map[string]any{
				"response": map[string]any{
					"candidates": []any{
						map[string]any{
							"content": map[string]any{
								"parts": []any{map[string]any{"text": "hello"}},
							},
						},
					},
				},
			}
			data, _ := json.Marshal(resp)
			return newHTTPResponse(http.StatusOK, data), nil
		},
	}

	handler := newHandlerForTests(cfg, stub)

	w := invokeGenerateContent(t, handler, []byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"candidates"`)
	require.GreaterOrEqual(t, len(attempts), 2)
}

func TestGenerateContent_UpstreamError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	stub := &stubUpstream{
		generateFunc: func(context.Context, []byte) (*http.Response, error) {
			return newHTTPResponse(http.StatusBadGateway, []byte(`{"error":"boom"}`)), nil
		},
	}
	handler := newHandlerForTests(&config.Config{}, stub)

	w := invokeGenerateContent(t, handler, []byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`))
	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "upstream error")
}

func TestGenerateContent_InvalidJSON(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	handler := newHandlerForTests(&config.Config{}, &stubUpstream{})

	w := invokeGenerateContent(t, handler, []byte("{"))
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid json")
}

func TestCountTokens_Success(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	resp := map[string]any{
		"response": map[string]any{"totalTokens": 42},
	}
	data, _ := json.Marshal(resp)
	stub := &stubUpstream{
		countTokensFunc: func(context.Context, []byte) (*http.Response, error) {
			return newHTTPResponse(http.StatusOK, data), nil
		},
	}
	handler := newHandlerForTests(&config.Config{GoogleProjID: "proj-123"}, stub)

	w := invokeCountTokens(t, handler, []byte(`{"contents":[{"role":"user","parts":[{"text":"count me"}]}]}`))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"totalTokens":42`)
}

func TestCountTokens_UpstreamError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	stub := &stubUpstream{
		countTokensFunc: func(context.Context, []byte) (*http.Response, error) {
			return newHTTPResponse(http.StatusBadGateway, []byte(`{"error":"fail"}`)), nil
		},
	}
	handler := newHandlerForTests(&config.Config{}, stub)

	w := invokeCountTokens(t, handler, []byte(`{"contents":[{"role":"user","parts":[{"text":"count"}]}]}`))
	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "upstream error")
}
