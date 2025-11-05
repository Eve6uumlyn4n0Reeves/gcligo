package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"gcli2api-go/internal/config"
	upstream "gcli2api-go/internal/upstream"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCompletions_Success(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	respObj := map[string]any{
		"response": map[string]any{
			"candidates": []any{
				map[string]any{
					"finishReason": "STOP",
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "hello world"},
						},
					},
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(5),
				"candidatesTokenCount": float64(7),
				"thoughtsTokenCount":   float64(0),
			},
		},
	}
	body, _ := json.Marshal(respObj)

	provider := &stubProvider{
		generateFunc: func(ctx upstream.RequestContext) upstream.ProviderResponse {
			return stubProviderResponse(body, http.StatusOK, ctx.BaseModel, nil)
		},
	}

	handler := newHandlerForTests(&config.Config{GoogleProjID: "proj-123"}, provider, nil)

	router := gin.New()
	router.POST("/v1/completions", handler.Completions)

	reqBody := map[string]any{
		"model":  "gemini-2.5-pro",
		"prompt": "hello",
	}

	w := postJSON(t, router, "/v1/completions", reqBody)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "text_completion", resp["object"])

	choices := resp["choices"].([]any)
	require.Len(t, choices, 1)
	first := choices[0].(map[string]any)
	require.Equal(t, "hello world", first["text"])
	require.Equal(t, "stop", first["finish_reason"])

	usage := resp["usage"].(map[string]any)
	require.EqualValues(t, 5, usage["prompt_tokens"])
	require.EqualValues(t, 7, usage["completion_tokens"])
}

func TestCompletions_InvalidPayload(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	handler := newHandlerForTests(&config.Config{}, nil, nil)
	router := gin.New()
	router.POST("/v1/completions", handler.Completions)

	// Missing prompt should trigger validation error
	w := postJSON(t, router, "/v1/completions", map[string]any{"model": "gemini-2.5-pro"})
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "prompt is required")
}

func TestCompletions_StreamSuccess(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	stream := "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"chunk\"}]}}],\"usageMetadata\":{\"promptTokenCount\":3,\"candidatesTokenCount\":2}}}\n\n"
	stream += "data: {\"response\":{\"candidates\":[{\"finishReason\":\"STOP\"}]}}\n\n"
	stream += "data: [DONE]\n\n"

	client := &stubGeminiClient{
		streamFunc: func(_ context.Context, _ []byte) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(stream)),
				Header:     make(http.Header),
			}, nil
		},
	}

	handler := newHandlerForTests(&config.Config{}, nil, client)
	router := gin.New()
	router.POST("/v1/completions", handler.Completions)

	w := postJSON(t, router, "/v1/completions", map[string]any{"model": "gemini-2.5-pro", "prompt": "hi", "stream": true})
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"object":"text_completion.chunk"`)
	require.Contains(t, w.Body.String(), `"finish_reason":"stop"`)
}

func TestCompletions_UpstreamError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	provider := &stubProvider{
		generateFunc: func(ctx upstream.RequestContext) upstream.ProviderResponse {
			return stubProviderResponse([]byte(`{"error":"boom"}`), http.StatusInternalServerError, ctx.BaseModel, nil)
		},
	}
	handler := newHandlerForTests(&config.Config{}, provider, nil)
	router := gin.New()
	router.POST("/v1/completions", handler.Completions)

	w := postJSON(t, router, "/v1/completions", map[string]any{"prompt": "hello"})
	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "upstream error")
}

func TestCompletions_StreamUpstreamError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	client := &stubGeminiClient{
		streamFunc: func(_ context.Context, _ []byte) (*http.Response, error) {
			return nil, errors.New("stream failure")
		},
	}
	handler := newHandlerForTests(&config.Config{}, nil, client)
	router := gin.New()
	router.POST("/v1/completions", handler.Completions)

	w := postJSON(t, router, "/v1/completions", map[string]any{"prompt": "hello", "stream": true})
	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "upstream_error")
}

func TestCompletions_StreamAggregatesUsage(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	stream := bytes.Buffer{}
	// First chunk with text and usage
	firstChunk := map[string]any{
		"response": map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "hello "},
						},
					},
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(5),
				"thoughtsTokenCount":   float64(2),
			},
		},
	}
	b1, _ := json.Marshal(firstChunk)
	stream.WriteString("data: ")
	stream.Write(b1)
	stream.WriteString("\n\n")
	// Final chunk to close stream
	finalChunk := map[string]any{
		"response": map[string]any{
			"candidates": []any{
				map[string]any{
					"finishReason": "STOP",
				},
			},
		},
	}
	b2, _ := json.Marshal(finalChunk)
	stream.WriteString("data: ")
	stream.Write(b2)
	stream.WriteString("\n\n")
	stream.WriteString("data: [DONE]\n\n")

	client := &stubGeminiClient{
		streamFunc: func(_ context.Context, _ []byte) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(stream.Bytes())),
				Header:     make(http.Header),
			}, nil
		},
	}

	handler := newHandlerForTests(&config.Config{}, nil, client)
	router := gin.New()
	router.POST("/v1/completions", handler.Completions)

	w := postJSON(t, router, "/v1/completions", map[string]any{"prompt": "hello", "stream": true})
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"prompt_tokens":12`)
	require.Contains(t, w.Body.String(), `"completion_tokens":5`)
	require.Contains(t, w.Body.String(), `"reasoning_tokens":2`)
}

func TestCompletions_AntiTruncationDisabled(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	respObj := map[string]any{
		"response": map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "short"},
						},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(respObj)
	provider := &stubProvider{
		generateFunc: func(ctx upstream.RequestContext) upstream.ProviderResponse {
			return stubProviderResponse(body, http.StatusOK, ctx.BaseModel, nil)
		},
	}
	handler := newHandlerForTests(&config.Config{}, provider, nil)
	handler.cfg.AntiTruncationEnabled = false
	router := gin.New()
	router.POST("/v1/completions", handler.Completions)

	w := postJSON(t, router, "/v1/completions", map[string]any{"prompt": "hi"})
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"text":"short"`)
}

func TestCompletions_StreamWritesDoneMarker(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	stream := bytes.Buffer{}
	stream.WriteString("data: {\"response\":{}}\n\n")
	stream.WriteString("data: [DONE]\n\n")

	client := &stubGeminiClient{
		streamFunc: func(_ context.Context, _ []byte) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(stream.Bytes())),
				Header:     make(http.Header),
			}, nil
		},
	}
	handler := newHandlerForTests(&config.Config{}, nil, client)
	router := gin.New()
	router.POST("/v1/completions", handler.Completions)

	start := time.Now()
	w := postJSON(t, router, "/v1/completions", map[string]any{"prompt": "hi", "stream": true})
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "[DONE]")
	require.Less(t, time.Since(start), 2*time.Second)
}
