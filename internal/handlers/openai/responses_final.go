package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	common "gcli2api-go/internal/handlers/common"
	"gcli2api-go/internal/oauth"
	upstream "gcli2api-go/internal/upstream"
	upgem "gcli2api-go/internal/upstream/gemini"
	"github.com/gin-gonic/gin"
)

// 非流式：聚合上游响应为 OpenAI Responses 对象
func (h *Handler) responsesFinal(c *gin.Context, baseModel string, gemReq map[string]any, model string) {
	ctx, cancel := common.WithUpstreamTimeout(c.Request.Context(), false)
	defer cancel()

	payload := map[string]any{"model": baseModel, "project": h.cfg.GoogleProjID, "request": gemReq}
	body, _ := json.Marshal(payload)

	client, usedCred := h.getUpstreamClient(ctx)
	resp, err := client.Generate(upstream.WithHeaderOverrides(ctx, c.Request.Header), body)
	if err != nil {
		common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	// 429 单次切换备选凭证
	if resp != nil && resp.StatusCode == 429 && usedCred != nil && h.credMgr != nil {
		byFirst, _ := upstream.ReadAll(resp)
		common.MarkCredentialFailure(h.credMgr, h.router, usedCred, "upstream_429", http.StatusTooManyRequests)
		if alt, errAlt := h.credMgr.GetAlternateCredential(usedCred.ID); errAlt == nil {
			oc := &oauth.Credentials{AccessToken: alt.AccessToken, ProjectID: alt.ProjectID}
			client = upgem.NewWithCredential(h.cfg, oc).WithCaller("openai")
			usedCred = alt
			resp, err = client.Generate(upstream.WithHeaderOverrides(ctx, c.Request.Header), body)
		}
		if err != nil {
			common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
			return
		}
		if resp != nil && resp.StatusCode >= 400 {
			if usedCred != nil {
				common.MarkCredentialFailure(h.credMgr, h.router, usedCred, "upstream_error", resp.StatusCode)
			}
			common.AbortWithUpstreamError(c, http.StatusBadGateway, "upstream_error", "upstream error", byFirst)
			return
		}
	}

	by, err := upstream.ReadAll(resp)
	if err != nil {
		common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	if resp != nil && resp.StatusCode >= 400 {
		if usedCred != nil {
			common.MarkCredentialFailure(h.credMgr, h.router, usedCred, "upstream_error", resp.StatusCode)
		}
		common.AbortWithUpstreamError(c, http.StatusBadGateway, "upstream_error", "upstream error", by)
		return
	}

	var obj map[string]any
	_ = json.Unmarshal(by, &obj)

	response := map[string]any{"id": fmt.Sprintf("resp_%x", time.Now().UnixNano()), "object": "response", "created_at": time.Now().Unix(), "status": "completed", "background": false, "error": nil}
	var outputs []any
	if r2, ok := obj["response"].(map[string]any); ok {
		if cands, ok := r2["candidates"].([]any); ok && len(cands) > 0 {
			if cand, ok := cands[0].(map[string]any); ok {
				if content, ok := cand["content"].(map[string]any); ok {
					if parts, ok := content["parts"].([]any); ok {
						var text strings.Builder
						var images []map[string]any
						for _, pp := range parts {
							if p0, ok := pp.(map[string]any); ok {
								if t, ok := p0["text"].(string); ok && t != "" {
									text.WriteString(t)
								}
								if in, ok := p0["inlineData"].(map[string]any); ok {
									mime := "image/png"
									if v, ok := in["mimeType"].(string); ok && v != "" {
										mime = v
									}
									if dataB64, _ := in["data"].(string); dataB64 != "" {
										imageURL := "data:" + mime + ";base64," + dataB64
										images = append(images, map[string]any{"type": "output_image", "image_url": map[string]any{"url": imageURL}})
									}
								}
								if fc, ok := p0["functionCall"].(map[string]any); ok {
									name, _ := fc["name"].(string)
									args := fc["args"]
									callID := fmt.Sprintf("call_%x", time.Now().UnixNano())
									outputs = append(outputs, map[string]any{"id": "fc_" + callID, "type": "function_call", "status": "completed", "arguments": jsonString(args), "call_id": callID, "name": name})
								}
							}
						}
						var contentParts []any
						if text.Len() > 0 {
							contentParts = append(contentParts, map[string]any{"type": "output_text", "text": text.String()})
						}
						for _, ip := range images {
							contentParts = append(contentParts, ip)
						}
						if len(contentParts) > 0 {
							outputs = append(outputs, map[string]any{"id": response["id"].(string) + "_0", "type": "message", "status": "completed", "content": contentParts, "role": "assistant"})
						}
					}
				}
				if um, ok := r2["usageMetadata"].(map[string]any); ok {
					if usage := common.BuildOpenAIUsageFromGemini(um); usage != nil {
						response["usage"] = usage
					}
				}
			}
		}
	}
	if len(outputs) > 0 {
		response["output"] = outputs
	}
	if usedCred != nil {
		common.MarkCredentialSuccess(h.credMgr, h.router, usedCred, http.StatusOK)
	}
	c.JSON(http.StatusOK, response)
}
