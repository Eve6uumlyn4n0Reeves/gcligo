package usage

import (
	"encoding/json"
)

// ExtractTokenUsageFromGeminiResponse extracts token usage from a Gemini API response
func ExtractTokenUsageFromGeminiResponse(body []byte) *TokenUsage {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil
	}

	// Try to extract from response.usageMetadata
	var usageMetadata map[string]any
	if resp, ok := obj["response"].(map[string]any); ok {
		if um, ok := resp["usageMetadata"].(map[string]any); ok {
			usageMetadata = um
		}
	} else if um, ok := obj["usageMetadata"].(map[string]any); ok {
		// Direct usageMetadata field
		usageMetadata = um
	}

	if usageMetadata == nil {
		return nil
	}

	tokens := &TokenUsage{}

	if v, ok := usageMetadata["promptTokenCount"].(float64); ok {
		tokens.InputTokens = int64(v)
	}
	if v, ok := usageMetadata["candidatesTokenCount"].(float64); ok {
		tokens.OutputTokens = int64(v)
	}
	if v, ok := usageMetadata["thoughtsTokenCount"].(float64); ok {
		tokens.ReasoningTokens = int64(v)
	}
	if v, ok := usageMetadata["cachedContentTokenCount"].(float64); ok {
		tokens.CachedTokens = int64(v)
	}
	if v, ok := usageMetadata["totalTokenCount"].(float64); ok {
		tokens.TotalTokens = int64(v)
	} else {
		// Calculate total if not provided
		tokens.TotalTokens = tokens.InputTokens + tokens.OutputTokens + tokens.ReasoningTokens
	}

	return tokens
}

// ExtractModelFromRequest extracts the model name from a request payload
func ExtractModelFromRequest(payload []byte) string {
	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		return ""
	}

	if model, ok := obj["model"].(string); ok {
		return model
	}

	return ""
}

