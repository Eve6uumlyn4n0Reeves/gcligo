package openai

import (
	"net/http/httptest"
	"strings"
	"testing"

	"gcli2api-go/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildChatRequest_InsertsSearchTool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{cfg: &config.Config{}}

	body := `{"model":"gemini-2.0-search","messages":[{"role":"user","content":"hi"}],"stream":false}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	req, errResp := buildChatRequest(h, c)
	require.Nil(t, errResp)
	require.NotNil(t, req)

	tools, ok := req.gemReq["tools"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, tools)

	tool, ok := tools[0].(map[string]any)
	require.True(t, ok)
	_, hasSearch := tool["googleSearch"]
	require.True(t, hasSearch)
}

func TestMergeToolResponses_AppendsFunctionResponse(t *testing.T) {
	raw := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"tool_calls": []any{
					map[string]any{
						"id":   "call-1",
						"type": "function",
						"function": map[string]any{
							"name": "lookup",
						},
					},
				},
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": "call-1",
				"content":      "result-text",
			},
		},
	}
	gemReq := map[string]any{
		"contents": []any{
			map[string]any{"role": "user"},
		},
	}

	mergeToolResponses(raw, gemReq)

	contents, ok := gemReq["contents"].([]any)
	require.True(t, ok)
	require.Len(t, contents, 2)

	toolEntry, ok := contents[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "tool", toolEntry["role"])

	parts, ok := toolEntry["parts"].([]any)
	require.True(t, ok)
	require.Len(t, parts, 1)

	part, ok := parts[0].(map[string]any)
	require.True(t, ok)

	fr, ok := part["functionResponse"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "lookup", fr["name"])

	resp, ok := fr["response"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "result-text", resp["result"])
}
