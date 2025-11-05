package gemini

import (
	"encoding/json"
	"testing"
)

func TestFixGeminiCLIImageHints_AddsImageResponseModality(t *testing.T) {
	model := "gemini-2.5-flash-image-preview"
	raw := []byte(`{"generationConfig":{},"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`)
	out := fixGeminiCLIImageHints(model, raw)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	gc, _ := m["generationConfig"].(map[string]any)
	if gc == nil {
		t.Fatalf("generationConfig missing")
	}
	mods, _ := gc["responseModalities"].([]any)
	if len(mods) == 0 || mods[0] != "Image" {
		t.Fatalf("responseModalities not set to [Image], got=%v", mods)
	}
}
