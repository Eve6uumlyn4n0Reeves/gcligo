package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	antitrunc "gcli2api-go/internal/antitrunc"
	"gcli2api-go/internal/credential"
	feat "gcli2api-go/internal/features"
	common "gcli2api-go/internal/handlers/common"
	logx "gcli2api-go/internal/logging"
	mw "gcli2api-go/internal/middleware"
	"gcli2api-go/internal/models"
	upstream "gcli2api-go/internal/upstream"
	"github.com/gin-gonic/gin"
)

func (h *Handler) completeChat(c *gin.Context, req *chatRequestContext, usedCred **credential.Credential) *chatError {
	ctx, cancel := common.WithUpstreamTimeout(c.Request.Context(), false)
	defer cancel()

	resp, usedModel, err := h.tryGenerateWithFallback(upstream.WithHeaderOverrides(ctx, c.Request.Header), usedCred, req.baseModel, h.cfg.GoogleProjID, req.gemReq)
	if err != nil {
		return newChatError(http.StatusBadGateway, err.Error(), "upstream_error")
	}
	body, err := upstream.ReadAll(resp)
	if err != nil {
		return newChatError(http.StatusBadGateway, err.Error(), "upstream_error")
	}
	if resp != nil && resp.StatusCode >= 400 {
		if cred := *usedCred; cred != nil {
			common.MarkCredentialFailure(h.credMgr, h.router, cred, "upstream_error", resp.StatusCode)
		}
		return newChatErrorWithBody(http.StatusBadGateway, "upstream error", "upstream_error", body)
	}

	logx.WithReq(c, map[string]interface{}{
		"upstream":        "gemini",
		"upstream_model":  usedModel,
		"upstream_status": resp.StatusCode,
		"upstream_stream": false,
	}).Info("upstream_completed")

	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	if usedModel != "" && usedModel != req.baseModel {
		mw.RecordFallback("openai", path, req.baseModel, usedModel)
	}

	var obj map[string]any
	_ = json.Unmarshal(body, &obj)

	var (
		textOut         string
		finish          string
		totalPrompt     int64
		totalCompletion int64
		reasoningTokens int64
	)

	if r, ok := obj["response"].(map[string]any); ok {
		if um, ok := r["usageMetadata"].(map[string]any); ok {
			if v, ok := um["promptTokenCount"].(float64); ok {
				totalPrompt = int64(v)
			}
			if v, ok := um["candidatesTokenCount"].(float64); ok {
				totalCompletion = int64(v)
			}
			if v, ok := um["thoughtsTokenCount"].(float64); ok {
				reasoningTokens = int64(v)
			}
		}
		if cands, ok := r["candidates"].([]any); ok && len(cands) > 0 {
			if cand, ok := cands[0].(map[string]any); ok {
				if fr, ok := cand["finishReason"].(string); ok && fr != "" {
					finish = mapFinishReason(fr)
				}
				if content, ok := cand["content"].(map[string]any); ok {
					if parts, ok := content["parts"].([]any); ok {
						for _, pp := range parts {
							if m, ok := pp.(map[string]any); ok {
								if t, ok := m["text"].(string); ok {
									textOut += t
								}
							}
						}
					}
				}
			}
		}
	}

	if models.IsAntiTruncation(req.model) || h.cfg.AntiTruncationEnabled {
		sh := feat.NewStreamHandler(feat.AntiTruncationConfig{MaxAttempts: h.cfg.AntiTruncationMax, Enabled: true})
		contFn := func(ctx context.Context) (string, error) {
			cont := req.cloneForContinuation()
			if cont == nil {
				cont = map[string]any{"contents": []any{}}
			}
			carr, _ := cont["contents"].([]any)
			if seed := antitrunc.CleanContinuationText(textOut); seed != "" {
				carr = append(carr, map[string]any{"role": "model", "parts": []any{map[string]any{"text": seed}}})
			}
			carr = append(carr, map[string]any{"role": "user", "parts": []any{map[string]any{"text": "continue"}}})
			cont["contents"] = carr
			project := h.cfg.GoogleProjID
			if cred := *usedCred; cred != nil && cred.ProjectID != "" {
				project = cred.ProjectID
			}
			payload := map[string]any{"model": req.baseModel, "project": project, "request": cont}
			b, _ := json.Marshal(payload)
			r2, err := h.baseClient.Generate(upstream.WithHeaderOverrides(c.Request.Context(), c.Request.Header), b)
			if err != nil {
				return "", err
			}
			body2, err := upstream.ReadAll(r2)
			if err != nil {
				return "", err
			}
			var parsedObj map[string]any
			if json.Unmarshal(body2, &parsedObj) != nil {
				return "", nil
			}
			parsed, _ := common.ExtractFromResponse(parsedObj)
			mw.RecordAntiTruncAttempt("openai", c.FullPath(), 1)
			return parsed.Text, nil
		}
		if full, err := sh.DetectAndHandle(c.Request.Context(), textOut, contFn); err == nil && full != "" {
			textOut = full
		}
	}

	usageMap := common.BuildOpenAIChatUsageFromGemini(map[string]any{
		"promptTokenCount":     float64(totalPrompt),
		"candidatesTokenCount": float64(totalCompletion),
		"thoughtsTokenCount":   float64(reasoningTokens),
		"totalTokenCount":      float64(totalPrompt + totalCompletion + reasoningTokens),
	})

	if cred := *usedCred; cred != nil {
		common.MarkCredentialSuccess(h.credMgr, h.router, cred, http.StatusOK)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      "cmpl-1",
		"object":  "text_completion",
		"created": time.Now().Unix(),
		"model":   req.model,
		"choices": []any{
			map[string]any{
				"index":         0,
				"text":          textOut,
				"logprobs":      nil,
				"finish_reason": finish,
			},
		},
		"usage": usageMap,
	})
	return nil
}
