package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gcli2api-go/internal/credential"
	common "gcli2api-go/internal/handlers/common"
	logx "gcli2api-go/internal/logging"
	mw "gcli2api-go/internal/middleware"
	upstream "gcli2api-go/internal/upstream"
	"github.com/gin-gonic/gin"
)

// 假流式：先输出 created/in_progress，再进行一次非流式上游调用，然后按小块输出文本/图片/工具事件，最后 completed/done
func (h *Handler) responsesFakeStream(c *gin.Context, baseModel string, gemReq map[string]any, model string) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	w := c.Writer
	fl, _ := w.(http.Flusher)

	respID := fmt.Sprintf("resp_%x", time.Now().UnixNano())
	_ = common.SSEWriteEvent(w, fl, "response.created", map[string]any{"type": "response.created", "sequence_number": 1, "response": map[string]any{"id": respID, "object": "response", "created_at": time.Now().Unix(), "status": "in_progress", "background": false, "error": nil}})
	_ = common.SSEWriteEvent(w, fl, "response.in_progress", map[string]any{"type": "response.in_progress", "sequence_number": 2, "response": map[string]any{"id": respID, "object": "response", "created_at": time.Now().Unix(), "status": "in_progress"}})

	// 走路由器（若启用）选择凭证
	var usedCred *credential.Credential
	if h.router != nil {
		ctxWith := upstream.WithHeaderOverrides(c.Request.Context(), c.Request.Header)
		if cred, info := h.router.PickWithInfo(ctxWith, upstream.HeaderOverrides(ctxWith)); cred != nil {
			usedCred = cred
			if h.cfg.RoutingDebugHeaders {
				if info != nil {
					c.Writer.Header().Set("X-Routing-Credential", info.CredID)
					c.Writer.Header().Set("X-Routing-Reason", info.Reason)
					if info.StickySource != "" {
						c.Writer.Header().Set("X-Routing-Sticky-Source", info.StickySource)
					}
				} else {
					c.Writer.Header().Set("X-Routing-Credential", cred.ID)
				}
			}
		}
	}
	if usedCred == nil {
		_, usedCred = h.getUpstreamClient(c.Request.Context())
	}

	ctx, cancel := common.WithUpstreamTimeout(c.Request.Context(), false)
	defer cancel()
	resp, usedModel, err := h.tryGenerateWithFallback(upstream.WithHeaderOverrides(ctx, c.Request.Header), &usedCred, baseModel, h.cfg.GoogleProjID, gemReq)
	if err != nil {
		common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	logx.WithReq(c, map[string]interface{}{"upstream": "gemini", "upstream_model": usedModel, "upstream_status": resp.StatusCode, "upstream_stream": false}).Info("upstream_completed")

	by, err := upstream.ReadAll(resp)
	if err != nil {
		common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	var obj map[string]any
	_ = json.Unmarshal(by, &obj)

	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	if usedModel != "" && usedModel != baseModel {
		mw.RecordFallback("openai", path, baseModel, usedModel)
	}
	if usedCred != nil && h.router != nil {
		h.router.OnResult(usedCred.ID, 200)
	}

	// 汇总首个候选的文本/图片/工具
	var text string
	var imgParts []map[string]any
	var outItems []any
	if r2, ok := obj["response"].(map[string]any); ok {
		if cands, ok := r2["candidates"].([]any); ok && len(cands) > 0 {
			if cand, ok := cands[0].(map[string]any); ok {
				if content, ok := cand["content"].(map[string]any); ok {
					if parts, ok := content["parts"].([]any); ok {
						for _, pp := range parts {
							if p0, ok := pp.(map[string]any); ok {
								if t, ok := p0["text"].(string); ok && t != "" {
									text += t
								}
								if in, ok := p0["inlineData"].(map[string]any); ok {
									mime := "image/png"
									if v, ok := in["mimeType"].(string); ok && v != "" {
										mime = v
									}
									if dataB64, _ := in["data"].(string); dataB64 != "" {
										imageURL := "data:" + mime + ";base64," + dataB64
										// 直接输出 image.delta
										_ = common.SSEWriteEvent(w, fl, "response.output_image.delta", map[string]any{"type": "response.output_image.delta", "sequence_number": 3, "item_id": "msg_" + respID, "output_index": 0, "content_index": 0, "delta": map[string]any{"image_url": map[string]any{"url": imageURL}}})
										imgParts = append(imgParts, map[string]any{"type": "output_image", "image_url": map[string]any{"url": imageURL}})
									}
								}
								if fc, ok := p0["functionCall"].(map[string]any); ok {
									name, _ := fc["name"].(string)
									args := fc["args"]
									callID := fmt.Sprintf("call_%x", time.Now().UnixNano())
									outItems = append(outItems, map[string]any{"id": "fc_" + callID, "type": "function_call", "status": "completed", "arguments": jsonString(args), "call_id": callID, "name": name})
								}
							}
						}
					}
				}
			}
		}
	}

	// 文本按小块 delta 输出
	if text != "" {
		for _, piece := range chunkText(text, h.fakeChunkSize()) {
			if piece == "" {
				continue
			}
			_ = common.SSEWriteEvent(w, fl, "response.output_text.delta", map[string]any{"type": "response.output_text.delta", "sequence_number": 3, "item_id": "msg_" + respID, "output_index": 0, "content_index": 0, "delta": piece, "logprobs": []any{}})
		}
	}
	// 工具：输出 added + arguments.delta
	for _, it := range outItems {
		if m, ok := it.(map[string]any); ok {
			id, _ := m["id"].(string)
			name, _ := m["name"].(string)
			args, _ := m["arguments"].(string)
			_ = common.SSEWriteEvent(w, fl, "response.output_item.added", map[string]any{"type": "response.output_item.added", "sequence_number": 3, "output_index": 0, "item": map[string]any{"id": id, "type": "function_call", "status": "in_progress", "arguments": "", "call_id": m["call_id"], "name": name}})
			_ = common.SSEWriteEvent(w, fl, "response.function_call_arguments.delta", map[string]any{"type": "response.function_call_arguments.delta", "sequence_number": 3, "item_id": id, "output_index": 0, "delta": args})
		}
	}
	// 图片 done
	for range imgParts {
		_ = common.SSEWriteEvent(w, fl, "response.output_image.done", map[string]any{"type": "response.output_image.done", "sequence_number": 4, "item_id": "msg_" + respID, "output_index": 0, "content_index": 0})
	}

	completed := map[string]any{"type": "response.completed", "sequence_number": 4, "response": map[string]any{"id": respID, "object": "response", "created_at": time.Now().Unix(), "status": "completed", "background": false, "error": nil}}
	_ = common.SSEWriteEvent(w, fl, "response.completed", completed)
	_ = common.SSEWriteEvent(w, fl, "done", map[string]any{})

	mw.RecordSSELines("openai", path, 1)
	mw.RecordSSEClose("openai", path, "completed")
}
