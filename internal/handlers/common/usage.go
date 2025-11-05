package common

// BuildOpenAIUsageFromGemini maps Gemini usageMetadata to OpenAI-style usage block.
// Expects fields: promptTokenCount, candidatesTokenCount, thoughtsTokenCount, totalTokenCount.
func BuildOpenAIUsageFromGemini(um map[string]any) map[string]any {
	if um == nil {
		return nil
	}
	usage := map[string]any{}
	inputTokens := int64(0)
	if v, ok := um["promptTokenCount"].(float64); ok {
		inputTokens += int64(v)
	}
	if v, ok := um["thoughtsTokenCount"].(float64); ok {
		inputTokens += int64(v)
	}
	usage["input_tokens"] = inputTokens
	usage["input_tokens_details"] = map[string]any{"cached_tokens": 0}
	if v, ok := um["candidatesTokenCount"].(float64); ok {
		usage["output_tokens"] = int64(v)
	}
	if v, ok := um["thoughtsTokenCount"].(float64); ok {
		usage["output_tokens_details"] = map[string]any{"reasoning_tokens": int64(v)}
	}
	if v, ok := um["totalTokenCount"].(float64); ok {
		usage["total_tokens"] = int64(v)
	}
	return usage
}

// BuildOpenAIChatUsageFromGemini maps to OpenAI chat usage fields (prompt_tokens, completion_tokens, total_tokens, completion_tokens_details.reasoning_tokens)
func BuildOpenAIChatUsageFromGemini(um map[string]any) map[string]any {
	if um == nil {
		return nil
	}
	prompt := int64(0)
	if v, ok := um["promptTokenCount"].(float64); ok {
		prompt = int64(v)
	}
	completion := int64(0)
	if v, ok := um["candidatesTokenCount"].(float64); ok {
		completion = int64(v)
	}
	reasoning := int64(0)
	if v, ok := um["thoughtsTokenCount"].(float64); ok {
		reasoning = int64(v)
	}
	total := int64(0)
	if v, ok := um["totalTokenCount"].(float64); ok {
		total = int64(v)
	}
	return map[string]any{
		"prompt_tokens":     prompt + reasoning,
		"completion_tokens": completion,
		"total_tokens": func() int64 {
			if total > 0 {
				return total
			}
			return prompt + completion + reasoning
		}(),
		"completion_tokens_details": map[string]any{"reasoning_tokens": reasoning},
	}
}
