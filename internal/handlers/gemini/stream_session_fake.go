package gemini

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	common "gcli2api-go/internal/handlers/common"
	mw "gcli2api-go/internal/middleware"
	upstream "gcli2api-go/internal/upstream"
)

func (s *streamSession) streamFake() {
	c := s.ginCtx
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	writer := c.Writer
	flusher, _ := writer.(http.Flusher)

	sseCount := 0
	toolCount := 0

	resp, err := s.client.Generate(s.ctx, s.payloadBytes)
	if err != nil {
		errObj := gin.H{"error": gin.H{"message": err.Error(), "type": "api_error"}}
		bj, _ := json.Marshal(errObj)
		writer.Write([]byte("data: "))
		writer.Write(bj)
		writer.Write([]byte("\n\n"))
		_ = common.SSEWriteDone(writer, flusher)
		mw.RecordSSEClose("gemini", s.path, "error")
		s.markFailure("upstream_error", 0)
		return
	}

	body, err := upstream.ReadAll(resp)
	if err != nil {
		_ = common.SSEWriteDone(writer, flusher)
		mw.RecordSSEClose("gemini", s.path, "error")
		s.markFailure("read_error", 0)
		return
	}

	if resp.StatusCode >= 400 {
		writer.Write([]byte("data: "))
		writer.Write(body)
		writer.Write([]byte("\n\n"))
		_ = common.SSEWriteDone(writer, flusher)
		mw.RecordSSEClose("gemini", s.path, "error")
		s.markFailure("upstream_error", resp.StatusCode)
		return
	}

	var obj map[string]any
	if json.Unmarshal(body, &obj) == nil {
		if r, ok := obj["response"].(map[string]any); ok {
			obj = r
		}
	}

	text, funcCalls, imgParts := splitFakeResponse(obj)

	chunkSize := 20
	if s.handler.cfg.FakeStreamingChunkSize > 0 {
		chunkSize = s.handler.cfg.FakeStreamingChunkSize
	}
	delay := time.Duration(0)
	if s.handler.cfg.FakeStreamingDelayMs > 0 {
		delay = time.Duration(s.handler.cfg.FakeStreamingDelayMs) * time.Millisecond
	}

	runes := []rune(text)
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		piece := string(runes[i:end])
		evt := map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": piece}}, "role": "model"}}}}
		sendSSEPayload(writer, flusher, evt)
		sseCount++
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	for _, fc := range funcCalls {
		evt := map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"functionCall": fc}}, "role": "model"}}}}
		sendSSEPayload(writer, flusher, evt)
		sseCount++
	}

	for _, img := range imgParts {
		evt := map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{img}, "role": "model"}}}}
		sendSSEPayload(writer, flusher, evt)
		sseCount++
	}

	_ = common.SSEWriteDone(writer, flusher)

	mw.RecordSSELines("gemini", s.path, sseCount)
	mw.RecordToolCalls("gemini", s.path, toolCount)
	mw.RecordSSEClose("gemini", s.path, "completed")

	if s.usedCred != nil {
		s.handler.credMgr.MarkSuccess(s.usedCred.ID)
	}
}

func splitFakeResponse(obj map[string]any) (string, []map[string]any, []map[string]any) {
	text := ""
	var funcCalls []map[string]any
	var imgParts []map[string]any

	if cands, ok := obj["candidates"].([]any); ok && len(cands) > 0 {
		if cand, ok := cands[0].(map[string]any); ok {
			if content, ok := cand["content"].(map[string]any); ok {
				if parts, ok := content["parts"].([]any); ok {
					for _, partRaw := range parts {
						part, ok := partRaw.(map[string]any)
						if !ok {
							continue
						}
						if t, ok := part["text"].(string); ok {
							text += t
						}
						if fc, ok := part["functionCall"].(map[string]any); ok {
							funcCalls = append(funcCalls, fc)
						}
						if in, ok := part["inlineData"].(map[string]any); ok {
							imgParts = append(imgParts, map[string]any{"inlineData": in})
						}
					}
				}
			}
		}
	}

	return text, funcCalls, imgParts
}

func sendSSEPayload(w gin.ResponseWriter, fl http.Flusher, payload any) {
	bytes, _ := json.Marshal(payload)
	w.Write([]byte("data: "))
	w.Write(bytes)
	w.Write([]byte("\n\n"))
	if fl != nil {
		fl.Flush()
	}
}
