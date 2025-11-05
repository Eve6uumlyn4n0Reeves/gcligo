package openai

import (
	"encoding/json"
)

// validateAndNormalizeOpenAI returns a possibly-normalized map and
// an HTTP status/message when invalid. When status==0, the map is valid.
func validateAndNormalizeOpenAI(raw map[string]any, isChat bool) (map[string]any, int, string) {
	if raw == nil {
		return map[string]any{"model": "gemini-2.5-pro"}, 0, ""
	}
	// model defaulting
	if m, ok := raw["model"].(string); !ok || m == "" {
		raw["model"] = "gemini-2.5-pro"
	}
	if isChat {
		// require messages array with at least one item
		msgs, ok := raw["messages"].([]any)
		if !ok || len(msgs) == 0 {
			return nil, 400, "messages must be a non-empty array"
		}
		// normalize message.content: if object with type=text, convert to string; if array of items, leave to translator
		norm := make([]any, 0, len(msgs))
		for _, mm := range msgs {
			m, ok := mm.(map[string]any)
			if !ok {
				return nil, 400, "each message must be an object"
			}
			role, _ := m["role"].(string)
			if role == "" {
				return nil, 400, "message.role is required"
			}
			if content, ok := m["content"].(map[string]any); ok {
				if content["type"] == "text" {
					if t, ok := content["text"].(string); ok {
						m["content"] = t
					}
				}
			}
			norm = append(norm, m)
		}
		raw["messages"] = norm
		return raw, 0, ""
	}
	// completions style: require prompt (string or array -> join)
	if p, ok := raw["prompt"].(string); ok && p != "" {
		return raw, 0, ""
	}
	if arr, ok := raw["prompt"].([]any); ok && len(arr) > 0 {
		// join as single string to keep behavior predictable
		var buf []byte
		// marshal back into a single string (simple + portable)
		b, _ := json.Marshal(arr)
		buf = append(buf, b...)
		raw["prompt"] = string(buf)
		return raw, 0, ""
	}
	return nil, 400, "prompt is required"
}
