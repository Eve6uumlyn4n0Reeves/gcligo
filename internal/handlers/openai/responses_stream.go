package openai

import (
	"encoding/json"
	"net/http"

	common "gcli2api-go/internal/handlers/common"
	mw "gcli2api-go/internal/middleware"
	upstream "gcli2api-go/internal/upstream"
	"github.com/gin-gonic/gin"
)

// 真流式：将上游 Gemini SSE 映射为 OpenAI Responses 事件的近似集合
func (h *Handler) responsesStream(c *gin.Context, baseModel string, gemReq map[string]any, model string) {
	client, usedCred := h.getUpstreamClient(c.Request.Context())
	payload := map[string]any{"model": baseModel, "project": h.cfg.GoogleProjID, "request": gemReq}
	body, _ := json.Marshal(payload)

	resp, usedModel, err := h.tryStreamWithFallback(upstream.WithHeaderOverrides(c.Request.Context(), c.Request.Header), &usedCred, baseModel, h.cfg.GoogleProjID, gemReq)
	if common.HandleUpstreamErrorAbort(c, resp, err, usedCred, h.credMgr, h.router, "upstream_stream_error") {
		return
	}

	w, fl := common.PrepareSSE(c)
	defer resp.Body.Close()

	// 响应起始事件
	created := map[string]any{"type": "response.created", "sequence_number": 1, "response": map[string]any{"id": "stream", "object": "response", "created_at": 0, "status": "in_progress", "background": false, "error": nil}}
	_ = common.SSEWriteEvent(w, fl, "response.created", created)
	_ = common.SSEWriteEvent(w, fl, "response.in_progress", map[string]any{"type": "response.in_progress", "sequence_number": 2, "response": map[string]any{"id": "stream", "object": "response", "created_at": 0, "status": "in_progress"}})

	scanner := common.NewSSEScanner(resp.Body)
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	sseCount := 2
	toolCount := 0

	for {
		event, done, err := scanner.Next()
		if err != nil {
			common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
			return
		}
		if done {
			break
		}
		if event == nil {
			continue
		}
		obj := event.Data
		if rpart, ok := obj["response"].(map[string]any); ok {
			obj = rpart
		}

		parsed, _ := common.ExtractFromResponse(obj)
		if parsed.Text != "" {
			_ = common.SSEWriteEvent(w, fl, "response.output_text.delta", map[string]any{"type": "response.output_text.delta", "sequence_number": 3, "item_id": "msg_stream", "output_index": 0, "content_index": 0, "delta": parsed.Text, "logprobs": []any{}})
			sseCount++
		}
		for _, im := range parsed.Images {
			mime, _ := im["mime"].(string)
			if mime == "" {
				mime = "image/png"
			}
			dataB64, _ := im["data"].(string)
			if dataB64 != "" {
				imageURL := "data:" + mime + ";base64," + dataB64
				_ = common.SSEWriteEvent(w, fl, "response.output_image.delta", map[string]any{"type": "response.output_image.delta", "sequence_number": 3, "item_id": "msg_stream", "output_index": 0, "content_index": 0, "delta": map[string]any{"image_url": map[string]any{"url": imageURL}}})
				sseCount++
			}
		}
		for _, fc := range parsed.FunctionCalls {
			// 简化：一次性输出 added + 完整 arguments.delta
			callID := "call_stream"
			itemID := "fc_" + callID
			_ = common.SSEWriteEvent(w, fl, "response.output_item.added", map[string]any{"type": "response.output_item.added", "sequence_number": 3, "output_index": 0, "item": map[string]any{"id": itemID, "type": "function_call", "status": "in_progress", "arguments": "", "call_id": callID, "name": fc.Name}})
			_ = common.SSEWriteEvent(w, fl, "response.function_call_arguments.delta", map[string]any{"type": "response.function_call_arguments.delta", "sequence_number": 3, "item_id": itemID, "output_index": 0, "delta": fc.ArgsJSON})
			toolCount++
			sseCount += 2
		}
	}

	// 结束事件
	_ = common.SSEWriteEvent(w, fl, "response.completed", map[string]any{"type": "response.completed", "sequence_number": 4, "response": map[string]any{"id": "stream", "object": "response", "created_at": 0, "status": "completed", "background": false, "error": nil}})
	_ = common.SSEWriteEvent(w, fl, "done", map[string]any{})
	sseCount += 2
	mw.RecordSSELines("openai", path, sseCount)
	mw.RecordToolCalls("openai", path, toolCount)
	mw.RecordSSEClose("openai", path, "completed")
	_ = usedModel // 保留以便后续扩展回退记录
	_ = client
	_ = body // 引用以避免未使用警告（tryStreamWithFallback 已处理请求）
}
