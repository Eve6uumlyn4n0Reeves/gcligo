package gemini

func countFunctionCalls(obj map[string]any) int {
	total := 0
	cands, ok := obj["candidates"].([]any)
	if !ok || len(cands) == 0 {
		return 0
	}
	for _, candRaw := range cands {
		cand, ok := candRaw.(map[string]any)
		if !ok {
			continue
		}
		content, ok := cand["content"].(map[string]any)
		if !ok {
			continue
		}
		parts, ok := content["parts"].([]any)
		if !ok {
			continue
		}
		for _, partRaw := range parts {
			part, ok := partRaw.(map[string]any)
			if !ok {
				continue
			}
			if _, ok := part["functionCall"].(map[string]any); ok {
				total++
			}
		}
	}
	return total
}
