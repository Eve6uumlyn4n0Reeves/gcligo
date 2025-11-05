package common

// Builders for OpenAI Responses SSE function-call events

// BuildFunctionCallAdded builds a response.output_item.added event payload for a function call item.
func BuildFunctionCallAdded(seq int, outputIndex int, itemID, callID, name string) map[string]any {
	return map[string]any{
		"type":            "response.output_item.added",
		"sequence_number": seq,
		"output_index":    outputIndex,
		"item": map[string]any{
			"id":        itemID,
			"type":      "function_call",
			"status":    "in_progress",
			"arguments": "",
			"call_id":   callID,
			"name":      name,
		},
	}
}

// BuildFunctionCallArgumentsDelta builds a response.function_call_arguments.delta event payload.
func BuildFunctionCallArgumentsDelta(seq int, outputIndex int, itemID, delta string) map[string]any {
	return map[string]any{
		"type":            "response.function_call_arguments.delta",
		"sequence_number": seq,
		"item_id":         itemID,
		"output_index":    outputIndex,
		"delta":           delta,
	}
}

// BuildFunctionCallItemDone builds a response.output_item.done for finalized function call item.
func BuildFunctionCallItemDone(seq int, outputIndex int, item map[string]any) map[string]any {
	return map[string]any{
		"type":            "response.output_item.done",
		"sequence_number": seq,
		"output_index":    outputIndex,
		"item":            item,
	}
}
