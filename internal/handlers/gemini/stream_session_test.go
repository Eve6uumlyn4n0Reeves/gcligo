package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSplitFakeResponse(t *testing.T) {
	input := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{"text": "hello"},
						map[string]any{"text": " world"},
						map[string]any{"functionCall": map[string]any{"name": "foo"}},
						map[string]any{"inlineData": map[string]any{"mimeType": "image/png"}},
					},
				},
			},
		},
	}

	text, calls, imgs := splitFakeResponse(input)

	require.Equal(t, "hello world", text)
	require.Len(t, calls, 1)
	require.Equal(t, "foo", calls[0]["name"])
	require.Len(t, imgs, 1)
	require.Equal(t, "image/png", imgs[0]["inlineData"].(map[string]any)["mimeType"])
}

func TestCountFunctionCalls(t *testing.T) {
	obj := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{"functionCall": map[string]any{"name": "a"}},
						map[string]any{"text": "no-call"},
						map[string]any{"functionCall": map[string]any{"name": "b"}},
					},
				},
			},
		},
	}

	require.Equal(t, 2, countFunctionCalls(obj))
}

func TestSendSSEPayload(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]string{"hello": "world"}

	sendSSEPayload(c.Writer, w, payload)

	require.Equal(t, "data: {\"hello\":\"world\"}\n\n", w.Body.String())
}

func TestSplitFakeResponseHandlesEmpty(t *testing.T) {
	text, calls, imgs := splitFakeResponse(map[string]any{})
	require.Empty(t, text)
	require.Len(t, calls, 0)
	require.Len(t, imgs, 0)
}

func TestSendSSEPayloadMarshallingError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// json.Marshal should fail on channel type which results in empty payload.
	sendSSEPayload(c.Writer, w, map[string]any{"ch": make(chan int)})

	// The helper should still write SSE prefix/suffix.
	require.Contains(t, w.Body.String(), "data: ")
	require.Contains(t, w.Body.String(), "\n\n")
}

func TestStreamGenerateContent_Success(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	stream := "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"chunk\"}]}}]}}\n\n"
	stream += "data: [DONE]\n\n"

	stub := &stubUpstream{
		streamFunc: func(context.Context, []byte) (*http.Response, error) {
			return newHTTPResponse(http.StatusOK, []byte(stream)), nil
		},
	}
	handler := newHandlerForTests(&config.Config{}, stub)

	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-pro:streamGenerateContent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "model", Value: "gemini-2.5-pro"}}

	handler.StreamGenerateContent(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"text":"chunk"`)
}

func TestStreamGenerateContent_FallbackOn404(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	stream := "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"fallback\"}]}}]}}\n\n"
	stream += "data: [DONE]\n\n"

	attemptModels := make([]string, 0, 2)
	stub := &stubUpstream{
		streamFunc: func(_ context.Context, payload []byte) (*http.Response, error) {
			var req map[string]any
			_ = json.Unmarshal(payload, &req)
			if m, ok := req["model"].(string); ok {
				attemptModels = append(attemptModels, m)
			}
			if len(attemptModels) == 1 {
				return newHTTPResponse(http.StatusNotFound, []byte(`{"error":"missing"}`)), nil
			}
			return newHTTPResponse(http.StatusOK, []byte(stream)), nil
		},
	}
	handler := newHandlerForTests(&config.Config{}, stub)

	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-pro:streamGenerateContent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "model", Value: "gemini-2.5-pro"}}

	handler.StreamGenerateContent(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"text":"fallback"`)
	require.GreaterOrEqual(t, len(attemptModels), 2)
	require.NotEqual(t, attemptModels[0], attemptModels[1])
}

func TestStreamGenerateContent_InvalidJSON(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	handler := newHandlerForTests(&config.Config{}, &stubUpstream{})

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.0:streamGenerateContent", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "model", Value: "gemini-2.0"}}

	handler.StreamGenerateContent(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid json")
}

func TestStreamGenerateContent_UpstreamError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	stub := &stubUpstream{
		streamFunc: func(context.Context, []byte) (*http.Response, error) {
			return nil, errors.New("boom")
		},
	}
	handler := newHandlerForTests(&config.Config{}, stub)

	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.0:streamGenerateContent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "model", Value: "gemini-2.0"}}

	handler.StreamGenerateContent(c)

	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "boom")
}

func TestStreamGenerateContent_UpstreamHTTPError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	stub := &stubUpstream{
		streamFunc: func(context.Context, []byte) (*http.Response, error) {
			return newHTTPResponse(http.StatusInternalServerError, []byte(`{"error":"fail"}`)), nil
		},
	}
	handler := newHandlerForTests(&config.Config{}, stub)

	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.0:streamGenerateContent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "model", Value: "gemini-2.0"}}

	handler.StreamGenerateContent(c)

	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "fail")
}
