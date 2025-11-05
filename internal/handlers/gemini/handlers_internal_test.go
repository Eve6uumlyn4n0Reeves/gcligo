package gemini

import (
	"gcli2api-go/internal/config"
	"testing"
)

func TestApplyRequestDecorators_ImagePlaceholder(t *testing.T) {
	cfg := &config.Config{AutoImagePlaceholder: true}
	h := New(cfg, nil, nil, nil)
	req := map[string]any{
		"generationConfig": map[string]any{
			"imageConfig": map[string]any{"aspectRatio": "16:9"},
		},
		"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": "请生成图片"}}}},
	}
	out := h.applyRequestDecorators("gemini-2.5-flash-image-preview", req)
	gc := out["generationConfig"].(map[string]any)
	if _, ok := gc["imageConfig"]; ok {
		t.Fatalf("imageConfig should be removed after placeholder injection")
	}
	if mods, ok := gc["responseModalities"].([]any); !ok || len(mods) == 0 {
		t.Fatalf("responseModalities should be set to include Image/Text")
	}
	contents := out["contents"].([]any)
	first := contents[0].(map[string]any)
	parts := first["parts"].([]any)
	// first two parts should be guide text and inlineData
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts after injection, got %d", len(parts))
	}
	if _, ok := parts[1].(map[string]any)["inlineData"]; !ok {
		t.Fatalf("second injected part should contain inlineData")
	}
}
