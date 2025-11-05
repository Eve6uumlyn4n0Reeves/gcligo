package common

import (
	"encoding/json"
	"strconv"
	"sync/atomic"
	"time"
)

var sseSeq uint64

func nextChunkID() string {
	// chatcmpl-<unix>-<seq>
	n := atomic.AddUint64(&sseSeq, 1)
	return "chatcmpl-" + strconv.FormatInt(time.Now().Unix(), 10) + "-" + strconv.FormatUint(n, 10)
}

// BuildDeltaRole builds an OpenAI chat.completion.chunk JSON with only role delta.
func BuildDeltaRole(model, role string) []byte {
	evt := map[string]any{
		"id":      nextChunkID(),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": role}, "finish_reason": nil}},
	}
	b, _ := json.Marshal(evt)
	return b
}

// BuildDeltaContent builds an OpenAI chat.completion.chunk JSON with text delta content.
func BuildDeltaContent(model, content string) []byte {
	evt := map[string]any{
		"id":      nextChunkID(),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": content}, "finish_reason": nil}},
	}
	b, _ := json.Marshal(evt)
	return b
}

// BuildFinal builds the final OpenAI chat.completion.chunk JSON with finish_reason and optional usage.
func BuildFinal(model, finish string, usage map[string]any) []byte {
	evt := map[string]any{
		"id":      nextChunkID(),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}},
	}
	if usage != nil {
		evt["usage"] = usage
	}
	b, _ := json.Marshal(evt)
	return b
}
