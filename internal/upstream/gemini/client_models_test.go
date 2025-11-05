package gemini

import (
	"testing"
)

func TestExtractModelIDsFromJSON(t *testing.T) {
	payload := []byte(`{
		"models": [
			{"name": "projects/demo/models/gemini-1.5-pro"},
			{"name": "Gemini-2.0-flash-exp"},
			{"metadata": {"latest": "GEMINI-2.5-pro-preview"}}
		],
		"notes": "other text gemini-1.5-flash kept here"
	}`)

	ids := extractModelIDsFromJSON(payload)
	if len(ids) == 0 {
		t.Fatalf("expected model IDs, got none")
	}
	want := map[string]bool{
		"gemini-1.5-pro":         true,
		"gemini-2.0-flash-exp":   true,
		"gemini-2.5-pro-preview": true,
		"gemini-1.5-flash":       true,
	}
	for _, id := range ids {
		delete(want, id)
	}
	if len(want) != 0 {
		t.Fatalf("missing IDs: %v", want)
	}
}

func TestCollectModelIDsDeeply(t *testing.T) {
	dest := make(map[string]struct{})
	collectModelIDs(map[string]any{
		"response": map[string]any{
			"candidates": []any{
				map[string]any{"model": "Gemini-1.5-pro"},
				map[string]any{"extra": map[string]any{"best_model": "GEMINI-1.5-FLASH"}},
			},
		},
	}, dest)

	if _, ok := dest["gemini-1.5-pro"]; !ok {
		t.Fatalf("missing gemini-1.5-pro")
	}
	if _, ok := dest["gemini-1.5-flash"]; !ok {
		t.Fatalf("missing gemini-1.5-flash")
	}
}
