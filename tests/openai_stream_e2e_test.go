package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gcli2api-go/internal/config"
	oh "gcli2api-go/internal/handlers/openai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 最小 e2e：验证 /v1/chat/completions 在 stream=true 时返回 SSE 头且包含基本 data: 块
func TestOpenAIChatCompletions_Stream_SSEHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 上游 SSE 假服务：模拟 Code Assist 的 streamGenerateContent 输出
	upstream := startTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "v1internal:streamGenerateContent") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// 构造最小 Gemini 流式片段（一个文本分片 + DONE）
		chunk := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "hello"},
						},
					},
				},
			},
		}
		b, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", string(b))
		fmt.Fprint(w, "data: [DONE]\n\n")
		// 不要关闭：由 httptest 控制
	}))
	defer upstream.Close()

	cfg := &config.Config{CodeAssist: upstream.URL}
	h := oh.New(cfg, nil, nil, nil, nil)

	r := gin.New()
	r.POST("/v1/chat/completions", h.ChatCompletions)

	// 最小请求体（stream=true）
	body := map[string]any{
		"model":    "gemini-2.5-pro",
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
		"stream":   true,
	}
	raw, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")
	// 至少包含一个 data: 块与 DONE 标记
	require.Contains(t, w.Body.String(), "data:")
	require.Contains(t, w.Body.String(), "[DONE]")
}
