package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	antitrunc "gcli2api-go/internal/antitrunc"
	feat "gcli2api-go/internal/features"
	common "gcli2api-go/internal/handlers/common"
	logx "gcli2api-go/internal/logging"
	mw "gcli2api-go/internal/middleware"
	"gcli2api-go/internal/models"
	tr "gcli2api-go/internal/translator"
	upstream "gcli2api-go/internal/upstream"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Completions(c *gin.Context) {
	var raw map[string]any
	if err := c.ShouldBindJSON(&raw); err != nil {
		common.AbortWithError(c, http.StatusBadRequest, "invalid_request_error", "invalid json")
		return
	}
	if normalized, status, msg := validateAndNormalizeOpenAI(raw, false); status != 0 {
		common.AbortWithError(c, status, "invalid_request_error", msg)
		return
	} else {
		raw = normalized
	}
	model, _ := raw["model"].(string)
	if model == "" {
		model = "gemini-2.5-pro"
	}
	stream, _ := raw["stream"].(bool)
	baseModel := models.BaseFromFeature(model)
	c.Set("model", model)
	c.Set("base_model", baseModel)
	rawJSON, _ := json.Marshal(raw)
	reqJSON := tr.OpenAICompletionsToGeminiRequest(baseModel, rawJSON, stream)
	var gemReq map[string]any
	_ = json.Unmarshal(reqJSON, &gemReq)
	client, usedCred := h.getUpstreamClient(c.Request.Context())
	effProject := h.cfg.GoogleProjID
	if usedCred != nil && usedCred.ProjectID != "" {
		effProject = usedCred.ProjectID
	}
	payload := map[string]any{"model": baseModel, "project": effProject, "request": gemReq}
	b, _ := json.Marshal(payload)
	if stream {
		resp, err := client.Stream(upstream.WithHeaderOverrides(c.Request.Context(), c.Request.Header), b)
		if common.HandleUpstreamErrorAbort(c, resp, err, usedCred, h.credMgr, h.router, "upstream_error") {
			return
		}
		w, fl := common.PrepareSSE(c)
		defer resp.Body.Close()
		scanner := common.NewSSEScanner(resp.Body)
		var totalPrompt, totalCompletion, reasoningTokens int64
		var finishReason string
		var aggText strings.Builder
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

			// Extract parsed data and usage
			parsed, usage := common.ExtractFromResponse(event.Data)

			// Update usage metadata
			if usage.PromptTokens > 0 {
				totalPrompt = usage.PromptTokens
			}
			if usage.CandidatesTokens > 0 {
				totalCompletion = usage.CandidatesTokens
			}
			if usage.ThoughtsTokens > 0 {
				reasoningTokens = usage.ThoughtsTokens
			}

			// Update finish reason
			if parsed.FinishReason != "" {
				finishReason = parsed.FinishReason
			}

			// Write text chunks
			if parsed.Text != "" {
				chunk := map[string]any{
					"id":      "cmpl-1",
					"object":  "text_completion.chunk",
					"created": time.Now().Unix(),
					"model":   model,
					"choices": []any{
						map[string]any{
							"index":         0,
							"text":          parsed.Text,
							"logprobs":      nil,
							"finish_reason": nil,
						},
					},
				}
				jb, _ := json.Marshal(chunk)
				w.Write([]byte("data: "))
				w.Write(jb)
				w.Write([]byte("\n\n"))
				fl.Flush()
				aggText.WriteString(parsed.Text)
			}
		}
		done := map[string]any{"id": "cmpl-1", "object": "text_completion.chunk", "created": time.Now().Unix(), "model": model, "choices": []any{map[string]any{"index": 0, "text": "", "logprobs": nil, "finish_reason": finishReason}}, "usage": common.BuildOpenAIChatUsageFromGemini(map[string]any{"promptTokenCount": float64(totalPrompt), "candidatesTokenCount": float64(totalCompletion), "thoughtsTokenCount": float64(reasoningTokens), "totalTokenCount": float64(totalPrompt + totalCompletion + reasoningTokens)})}
		jb, _ := json.Marshal(done)
		w.Write([]byte("data: "))
		w.Write(jb)
		w.Write([]byte("\n\n"))
		_ = common.SSEWriteDone(w, fl)
		if usedCred != nil {
			common.MarkCredentialSuccess(h.credMgr, h.router, usedCred, http.StatusOK)
		}
		return
	}

	ctx, cancel := common.WithUpstreamTimeout(c.Request.Context(), false)
	defer cancel()
	resp, usedModel2, err := h.tryGenerateWithFallback(upstream.WithHeaderOverrides(ctx, c.Request.Header), &usedCred, baseModel, h.cfg.GoogleProjID, gemReq)
	if common.HandleUpstreamErrorAbort(c, resp, err, usedCred, h.credMgr, h.router, "upstream_error") {
		return
	}
	by, err := upstream.ReadAll(resp)
	if err != nil {
		common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	logx.WithReq(c, map[string]interface{}{
		"upstream":        "gemini",
		"upstream_model":  usedModel2,
		"upstream_status": resp.StatusCode,
		"upstream_stream": false,
	}).Info("upstream_completed")
	var obj map[string]any
	_ = json.Unmarshal(by, &obj)
	if usedModel2 != "" {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if usedModel2 != baseModel {
			mw.RecordFallback("openai", path, baseModel, usedModel2)
		}
	}
	var textOut, finish string
	var totalPrompt, totalCompletion, reasoningTokens int64
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
	if models.IsAntiTruncation(model) || h.cfg.AntiTruncationEnabled {
		sh := feat.NewStreamHandler(feat.AntiTruncationConfig{MaxAttempts: h.cfg.AntiTruncationMax, Enabled: true})
		contFn := func(ctx context.Context) (string, error) {
			cont := cloneMap(gemReq)
			if cont == nil {
				cont = map[string]any{"contents": []any{}}
			}
			carr, _ := cont["contents"].([]any)
			if seed := antitrunc.CleanContinuationText(textOut); seed != "" {
				carr = append(carr, map[string]any{"role": "model", "parts": []any{map[string]any{"text": seed}}})
			}
			carr = append(carr, map[string]any{"role": "user", "parts": []any{map[string]any{"text": "continue"}}})
			cont["contents"] = carr
			effProject := h.cfg.GoogleProjID
			if usedCred != nil && usedCred.ProjectID != "" {
				effProject = usedCred.ProjectID
			}
			payload2 := map[string]any{"model": baseModel, "project": effProject, "request": cont}
			b2, _ := json.Marshal(payload2)
			r2, err := h.baseClient.Generate(upstream.WithHeaderOverrides(c.Request.Context(), c.Request.Header), b2)
			if err != nil {
				return "", err
			}
			by2, err := upstream.ReadAll(r2)
			if err != nil {
				return "", err
			}
			var o map[string]any
			if json.Unmarshal(by2, &o) != nil {
				return "", nil
			}
			parsed, _ := common.ExtractFromResponse(o)
			mw.RecordAntiTruncAttempt("openai", c.FullPath(), 1)
			return parsed.Text, nil
		}
		if full, err := sh.DetectAndHandle(c.Request.Context(), textOut, func(ctx context.Context) (string, error) { return contFn(ctx) }); err == nil && full != "" {
			textOut = full
		}
	}
	usageMap := common.BuildOpenAIChatUsageFromGemini(map[string]any{"promptTokenCount": float64(totalPrompt), "candidatesTokenCount": float64(totalCompletion), "thoughtsTokenCount": float64(reasoningTokens), "totalTokenCount": float64(totalPrompt + totalCompletion + reasoningTokens)})
	if usedCred != nil {
		common.MarkCredentialSuccess(h.credMgr, h.router, usedCred, http.StatusOK)
	}
	c.JSON(http.StatusOK, gin.H{"id": "cmpl-1", "object": "text_completion", "created": time.Now().Unix(), "model": model, "choices": []any{map[string]any{"index": 0, "text": textOut, "logprobs": nil, "finish_reason": finish}}, "usage": usageMap})
}
