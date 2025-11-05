package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"gcli2api-go/internal/credential"
	feat "gcli2api-go/internal/features"
	common "gcli2api-go/internal/handlers/common"
	logx "gcli2api-go/internal/logging"
	mw "gcli2api-go/internal/middleware"
	"gcli2api-go/internal/models"
	upstream "gcli2api-go/internal/upstream"
	"github.com/gin-gonic/gin"
)

func (h *Handler) streamChatCompletions(c *gin.Context, req *chatRequestContext, client geminiClient, usedCred **credential.Credential) *chatError {
	baseCtx := upstream.WithHeaderOverrides(c.Request.Context(), c.Request.Header)
	ctxStream, cancelStream := common.WithUpstreamTimeout(baseCtx, true)
	defer cancelStream()

	resp, usedModel, err := h.tryStreamWithFallback(ctxStream, usedCred, req.baseModel, h.cfg.GoogleProjID, req.gemReq)
	if err != nil {
		return newChatError(http.StatusBadGateway, err.Error(), "upstream_error")
	}
	if resp != nil && resp.StatusCode >= 400 {
		body, _ := upstream.ReadAll(resp)
		if cred := *usedCred; cred != nil {
			common.MarkCredentialFailure(h.credMgr, h.router, cred, "upstream_stream_error", resp.StatusCode)
		}
		return newChatErrorWithBody(http.StatusBadGateway, "upstream error", "upstream_error", body)
	}

	logx.WithReq(c, map[string]interface{}{
		"upstream":        "gemini",
		"upstream_model":  usedModel,
		"upstream_status": resp.StatusCode,
		"upstream_stream": true,
	}).Info("upstream_connected")

	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	if usedModel != "" && usedModel != req.baseModel {
		mw.RecordFallback("openai", path, req.baseModel, usedModel)
	}

	w, fl := common.PrepareSSE(c)
	defer resp.Body.Close()

	var wrapped io.Reader = resp.Body
	if models.IsAntiTruncation(req.model) || h.cfg.AntiTruncationEnabled {
		sh := feat.NewStreamHandler(feat.AntiTruncationConfig{MaxAttempts: h.cfg.AntiTruncationMax, Enabled: true})
		contFn := func(ctx context.Context) (io.Reader, error) {
			cont := req.cloneForContinuation()
			if cont == nil {
				cont = map[string]any{"contents": []any{}}
			}
			carr, _ := cont["contents"].([]any)
			carr = append(carr, map[string]any{"role": "user", "parts": []any{map[string]any{"text": "continue"}}})
			cont["contents"] = carr
			project := h.cfg.GoogleProjID
			if cred := *usedCred; cred != nil && cred.ProjectID != "" {
				project = cred.ProjectID
			}
			payload := map[string]any{"model": req.baseModel, "project": project, "request": cont}
			b, _ := json.Marshal(payload)
			r2, err := client.Stream(upstream.WithHeaderOverrides(c.Request.Context(), c.Request.Header), b)
			if err != nil {
				return nil, err
			}
			return r2.Body, nil
		}
		if wrappedStream, err := sh.WrapStream(c.Request.Context(), wrapped, contFn); err == nil && wrappedStream != nil {
			wrapped = wrappedStream
		}
	}

	scanner := common.NewSSEScanner(wrapped)
	extractor := common.NewStreamDeltaExtractor(req.model)
	sseCount := 0

	path = c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}

	for {
		event, done, err := scanner.Next()
		if err != nil {
			return newChatError(http.StatusBadGateway, err.Error(), "stream_error")
		}
		if done {
			mw.RecordSSEClose("openai", path, "done")
			sseCount++
			break
		}
		if event == nil {
			continue
		}

		// Use unified extractor
		chunks := extractor.ExtractDelta(event)
		for _, chunk := range chunks {
			w.Write([]byte("data: "))
			w.Write(chunk.Data)
			w.Write([]byte("\n\n"))
			fl.Flush()
			sseCount++
		}
	}

	common.SSEWriteDone(w, fl)
	mw.RecordSSELines("openai", path, sseCount)
	if cred := *usedCred; cred != nil {
		common.MarkCredentialSuccess(h.credMgr, h.router, cred, http.StatusOK)
	}
	return nil
}
