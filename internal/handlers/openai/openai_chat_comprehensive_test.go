package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gcli2api-go/internal/config"
	upstream "gcli2api-go/internal/upstream"
	upgem "gcli2api-go/internal/upstream/gemini"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeProvider struct {
	name           string
	generateFunc   func(upstream.RequestContext) upstream.ProviderResponse
	streamFunc     func(upstream.RequestContext) upstream.ProviderResponse
	listModelsFunc func(upstream.RequestContext) upstream.ProviderListResponse
}

func (f *fakeProvider) Name() string {
	if f.name != "" {
		return f.name
	}
	return "fake"
}

func (f *fakeProvider) SupportsModel(string) bool { return true }

func (f *fakeProvider) Stream(ctx upstream.RequestContext) upstream.ProviderResponse {
	if f.streamFunc != nil {
		return f.streamFunc(ctx)
	}
	return upstream.ProviderResponse{}
}

func (f *fakeProvider) Generate(ctx upstream.RequestContext) upstream.ProviderResponse {
	if f.generateFunc != nil {
		return f.generateFunc(ctx)
	}
	return upstream.ProviderResponse{}
}

func (f *fakeProvider) ListModels(ctx upstream.RequestContext) upstream.ProviderListResponse {
	if f.listModelsFunc != nil {
		return f.listModelsFunc(ctx)
	}
	return upstream.ProviderListResponse{}
}

func (f *fakeProvider) Invalidate(string) {}

func newTestHandler(cfg *config.Config, prov upstream.Provider) *Handler {
	return &Handler{
		cfg:         cfg,
		providers:   upstream.NewManager(prov),
		baseClient:  upgem.New(cfg).WithCaller("openai-test"),
		clientCache: make(map[string]geminiClient),
	}
}

func postJSON(t *testing.T, router *gin.Engine, path string, body map[string]any) *httptest.ResponseRecorder {
	data, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestChatCompletions_Success(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	prov := &fakeProvider{
		generateFunc: func(ctx upstream.RequestContext) upstream.ProviderResponse {
			payload := map[string]any{
				"response": map[string]any{
					"candidates": []any{
						map[string]any{
							"finishReason": "STOP",
							"content": map[string]any{
								"parts": []any{
									map[string]any{"text": "Hello from Gemini"},
								},
							},
						},
					},
					"usageMetadata": map[string]any{
						"promptTokenCount":     float64(12),
						"candidatesTokenCount": float64(7),
						"thoughtsTokenCount":   float64(0),
					},
				},
			}
			raw, err := json.Marshal(payload)
			require.NoError(t, err)
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(raw)),
				Header:     make(http.Header),
			}
			return upstream.ProviderResponse{Resp: resp, UsedModel: ctx.BaseModel}
		},
	}
	handler := newTestHandler(cfg, prov)

	router := gin.New()
	router.POST("/v1/chat/completions", handler.ChatCompletions)

	body := map[string]any{
		"model": "gemini-2.5-pro",
		"messages": []any{
			map[string]any{"role": "user", "content": "Hi"},
		},
	}
	w := postJSON(t, router, "/v1/chat/completions", body)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	require.Equal(t, "gemini-2.5-pro", resp["model"])

	choices, ok := resp["choices"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, choices)

	firstChoice, ok := choices[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Hello from Gemini", firstChoice["text"])
	require.Equal(t, "stop", firstChoice["finish_reason"])

	usage, ok := resp["usage"].(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 12, usage["prompt_tokens"])
	require.EqualValues(t, 7, usage["completion_tokens"])
}

func TestChatCompletions_UpstreamError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	prov := &fakeProvider{
		generateFunc: func(ctx upstream.RequestContext) upstream.ProviderResponse {
			resp := &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"error":"upstream failure"}`)),
				Header:     make(http.Header),
			}
			return upstream.ProviderResponse{Resp: resp, UsedModel: ctx.BaseModel}
		},
	}
	handler := newTestHandler(cfg, prov)

	router := gin.New()
	router.POST("/v1/chat/completions", handler.ChatCompletions)

	body := map[string]any{
		"model": "gemini-2.5-pro",
		"messages": []any{
			map[string]any{"role": "user", "content": "Hi"},
		},
	}
	w := postJSON(t, router, "/v1/chat/completions", body)
	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "upstream error")
}

func TestChatCompletions_InvalidPayload(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handler := newTestHandler(cfg, &fakeProvider{})

	router := gin.New()
	router.POST("/v1/chat/completions", handler.ChatCompletions)

	body := map[string]any{
		"model": "gemini-2.5-pro",
	}
	w := postJSON(t, router, "/v1/chat/completions", body)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "messages")
}

func TestChatCompletions_StreamSuccess(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	streamBody := "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"partial\"}]}}]}}\n\ndata: [DONE]\n"
	prov := &fakeProvider{
		streamFunc: func(ctx upstream.RequestContext) upstream.ProviderResponse {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(streamBody)),
				Header:     make(http.Header),
			}
			return upstream.ProviderResponse{Resp: resp, UsedModel: ctx.BaseModel}
		},
	}
	handler := newTestHandler(cfg, prov)

	router := gin.New()
	router.POST("/v1/chat/completions", handler.ChatCompletions)

	body := map[string]any{
		"model":  "gemini-2.5-pro",
		"stream": true,
		"messages": []any{
			map[string]any{"role": "user", "content": "Hi"},
		},
	}

	w := postJSON(t, router, "/v1/chat/completions", body)
	require.Equal(t, http.StatusOK, w.Code)
	output := w.Body.String()
	require.Contains(t, output, "data: ")
	require.Contains(t, output, "partial")
	require.Contains(t, output, "data: [DONE]")
}

func TestChatCompletions_StreamError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	prov := &fakeProvider{
		streamFunc: func(ctx upstream.RequestContext) upstream.ProviderResponse {
			return upstream.ProviderResponse{Err: errors.New("boom")}
		},
	}
	handler := newTestHandler(cfg, prov)

	router := gin.New()
	router.POST("/v1/chat/completions", handler.ChatCompletions)

	body := map[string]any{
		"model":  "gemini-2.5-pro",
		"stream": true,
		"messages": []any{
			map[string]any{"role": "user", "content": "Hi"},
		},
	}

	w := postJSON(t, router, "/v1/chat/completions", body)
	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "upstream_error")
}
