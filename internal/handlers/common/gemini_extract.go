package common

import (
	"encoding/json"
)

type FunctionCall struct {
	Name     string
	ArgsJSON string
}

type UsageMeta struct {
	PromptTokens     int64
	CandidatesTokens int64
	ThoughtsTokens   int64
	TotalTokens      int64
}

type ParsedCandidate struct {
	Text          string
	Thought       string
	Images        []map[string]any
	FunctionCalls []FunctionCall
	FinishReason  string
}

// ExtractFromResponse extracts candidate text/images/functionCalls/finishReason and usage from a Gemini response-like object.
// The input may be either an envelope `{ "response": { ... } }` or a direct `{ "candidates": [...] }` body.
func ExtractFromResponse(obj map[string]any) (ParsedCandidate, UsageMeta) {
	parsed := ParsedCandidate{}
	usage := UsageMeta{}

	// unwrap {response: {...}}
	if r, ok := obj["response"].(map[string]any); ok {
		obj = r
	}

	// usageMetadata
	if um, ok := obj["usageMetadata"].(map[string]any); ok {
		if v, ok := um["promptTokenCount"].(float64); ok {
			usage.PromptTokens = int64(v)
		}
		if v, ok := um["candidatesTokenCount"].(float64); ok {
			usage.CandidatesTokens = int64(v)
		}
		if v, ok := um["thoughtsTokenCount"].(float64); ok {
			usage.ThoughtsTokens = int64(v)
		}
		if v, ok := um["totalTokenCount"].(float64); ok {
			usage.TotalTokens = int64(v)
		}
	}

	// candidates -> first
	cands, ok := obj["candidates"].([]any)
	if !ok || len(cands) == 0 {
		return parsed, usage
	}
	cand, ok := cands[0].(map[string]any)
	if !ok {
		return parsed, usage
	}

	if fr, ok := cand["finishReason"].(string); ok && fr != "" {
		parsed.FinishReason = mapFinishReason(fr)
	}
	content, _ := cand["content"].(map[string]any)
	parts, _ := content["parts"].([]any)
	for _, p := range parts {
		m, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if t, ok := m["text"].(string); ok && t != "" {
			parsed.Text += t
			continue
		}
		if th, ok := m["thought"].(string); ok && th != "" {
			parsed.Thought += th
			continue
		}
		if in, ok := m["inlineData"].(map[string]any); ok {
			mime := "image/png"
			if v, ok := in["mimeType"].(string); ok && v != "" {
				mime = v
			}
			dataB64, _ := in["data"].(string)
			if dataB64 != "" {
				// Consumers can format to OpenAI or Responses image events as needed
				parsed.Images = append(parsed.Images, map[string]any{
					"mime": mime,
					"data": dataB64,
				})
			}
			continue
		}
		if fc, ok := m["functionCall"].(map[string]any); ok {
			fname, _ := fc["name"].(string)
			var argsJSON string
			if raw, ok := fc["args"]; ok {
				if b, err := json.Marshal(raw); err == nil {
					argsJSON = string(b)
				}
			}
			parsed.FunctionCalls = append(parsed.FunctionCalls, FunctionCall{Name: fname, ArgsJSON: argsJSON})
			continue
		}
	}
	return parsed, usage
}

// mapFinishReason converts Gemini finish reasons to OpenAI format
func mapFinishReason(geminiReason string) string {
	switch geminiReason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY", "RECITATION":
		return "content_filter"
	case "OTHER":
		return "stop"
	default:
		return geminiReason
	}
}

// StreamDeltaExtractor processes SSE events and extracts deltas for streaming responses
type StreamDeltaExtractor struct {
	model string
}

// NewStreamDeltaExtractor creates a new stream delta extractor
func NewStreamDeltaExtractor(model string) *StreamDeltaExtractor {
	return &StreamDeltaExtractor{model: model}
}

// SSEChunk represents a single chunk of streaming data
type SSEChunk struct {
	Type string // "delta_content", "delta_image", "tool_call", "finish"
	Data []byte
}

// ExtractDelta extracts streaming deltas from an SSE event
func (e *StreamDeltaExtractor) ExtractDelta(event *SSEEvent) []SSEChunk {
	if event == nil {
		return nil
	}

	parsed, _ := ExtractFromResponse(event.Data)
	chunks := []SSEChunk{}

	// Text delta
	if parsed.Text != "" {
		chunks = append(chunks, SSEChunk{
			Type: "delta_content",
			Data: BuildDeltaContent(e.model, parsed.Text),
		})
	}

	// Image deltas
	for _, img := range parsed.Images {
		mime, _ := img["mime"].(string)
		if mime == "" {
			mime = "image/png"
		}
		dataB64, _ := img["data"].(string)
		if dataB64 != "" {
			url := "data:" + mime + ";base64," + dataB64
			imgData := map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{"url": url},
			}
			imgJSON, _ := json.Marshal(imgData)
			chunks = append(chunks, SSEChunk{
				Type: "delta_image",
				Data: BuildDeltaContent(e.model, string(imgJSON)),
			})
		}
	}

	// Tool call deltas
	for _, fc := range parsed.FunctionCalls {
		chunks = append(chunks, SSEChunk{
			Type: "tool_call",
			Data: BuildToolCallDelta(e.model, fc.Name, fc.ArgsJSON),
		})
	}

	return chunks
}

// BuildToolCallDelta builds an OpenAI tool call delta chunk
func BuildToolCallDelta(model, name, argsJSON string) []byte {
	evt := map[string]any{
		"id":      "chatcmpl-tool",
		"object":  "chat.completion.chunk",
		"created": 0,
		"model":   model,
		"choices": []any{
			map[string]any{
				"index": 0,
				"delta": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"index": 0,
							"id":    "call_" + name,
							"type":  "function",
							"function": map[string]any{
								"name":      name,
								"arguments": argsJSON,
							},
						},
					},
				},
				"finish_reason": nil,
			},
		},
	}
	b, _ := json.Marshal(evt)
	return b
}
