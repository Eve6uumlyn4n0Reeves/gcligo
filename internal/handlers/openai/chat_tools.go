package openai

func injectSearchTool(gemReq map[string]any) {
	tool := map[string]any{"googleSearch": map[string]any{}}
	if tools, ok := gemReq["tools"].([]any); ok {
		gemReq["tools"] = append(tools, tool)
		return
	}
	gemReq["tools"] = []any{tool}
}

func mergeToolResponses(raw map[string]any, gemReq map[string]any) {
	tcToName := extractAssistantToolCalls(raw)
	if len(tcToName) == 0 {
		return
	}
	toolResponses := extractToolResponses(raw)
	if len(toolResponses) == 0 {
		return
	}

	parts := []any{}
	for id, name := range tcToName {
		resp := toolResponses[id]
		if resp == "" {
			continue
		}
		parts = append(parts, map[string]any{
			"functionResponse": map[string]any{
				"name": name,
				"response": map[string]any{
					"result": resp,
				},
			},
		})
	}
	if len(parts) == 0 {
		return
	}

	toolNode := map[string]any{
		"role":  "tool",
		"parts": parts,
	}

	if contents, ok := gemReq["contents"].([]any); ok {
		gemReq["contents"] = append(contents, toolNode)
		return
	}
	gemReq["contents"] = []any{toolNode}
}

func extractAssistantToolCalls(raw map[string]any) map[string]string {
	result := map[string]string{}
	messages, ok := raw["messages"].([]any)
	if !ok {
		return result
	}
	for _, mm := range messages {
		msg, ok := mm.(map[string]any)
		if !ok || msg["role"] != "assistant" {
			continue
		}
		calls, _ := msg["tool_calls"].([]any)
		for _, tc := range calls {
			tcm, ok := tc.(map[string]any)
			if !ok || tcm["type"] != "function" {
				continue
			}
			id, _ := tcm["id"].(string)
			fn, _ := tcm["function"].(map[string]any)
			name, _ := fn["name"].(string)
			if id != "" && name != "" {
				result[id] = name
			}
		}
	}
	return result
}

func extractToolResponses(raw map[string]any) map[string]string {
	out := map[string]string{}
	messages, ok := raw["messages"].([]any)
	if !ok {
		return out
	}
	for _, mm := range messages {
		msg, ok := mm.(map[string]any)
		if !ok || msg["role"] != "tool" {
			continue
		}
		id, _ := msg["tool_call_id"].(string)
		if id == "" {
			continue
		}
		switch v := msg["content"].(type) {
		case string:
			out[id] = v
		case map[string]any:
			if v["type"] == "text" {
				if text, ok := v["text"].(string); ok {
					out[id] = text
				}
			}
		}
	}
	return out
}
