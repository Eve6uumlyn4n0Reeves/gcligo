package gemini

import (
	"strings"

	"gcli2api-go/internal/models"
	util "gcli2api-go/internal/utils"
)

// applyRequestDecorators injects safety, thinking, search, and placeholder helpers based on model features.
func (h *Handler) applyRequestDecorators(model string, req map[string]any) map[string]any {
	if req == nil {
		req = map[string]any{}
	}
	gc, _ := req["generationConfig"].(map[string]any)
	if gc == nil {
		gc = map[string]any{}
		req["generationConfig"] = gc
	}
	// thinking
	tc, _ := gc["thinkingConfig"].(map[string]any)
	if tc == nil {
		tc = map[string]any{}
		gc["thinkingConfig"] = tc
	}
	if models.IsNoThinking(model) {
		tc["includeThoughts"] = false
		tc["thinkingBudget"] = 0
	}
	if models.IsMaxThinking(model) {
		tc["includeThoughts"] = true
		tc["thinkingBudget"] = 32768
	}
	// Some image models disallow thinkingConfig; strip for flash-image variants to avoid upstream errors.
	base := strings.ToLower(models.BaseFromFeature(model))
	if strings.Contains(base, "flash-image") {
		delete(gc, "thinkingConfig")
	}
	// Inject placeholder for flash-image-preview when needed.
	_ = util.ApplyFlashImagePreviewPlaceholder(req, base, h.cfg.AutoImagePlaceholder)
	// safety default
	if _, ok := req["safetySettings"]; !ok {
		req["safetySettings"] = []map[string]any{{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_NONE"}}
	}
	// search tools
	if models.IsSearch(model) {
		if _, ok := req["tools"]; !ok {
			req["tools"] = []any{}
		}
		req["tools"] = append(req["tools"].([]any), map[string]any{"googleSearch": map[string]any{}})
	}
	// anti-truncation done marker injection (optional)
	if h.cfg.AntiTruncationEnabled || models.IsAntiTruncation(model) {
		sys, _ := req["systemInstruction"].(map[string]any)
		if sys == nil {
			sys = map[string]any{}
		}
		parts, _ := sys["parts"].([]any)
		text := "当你完成完整回答时，请在输出最后单独一行输出：[done]"
		parts = append(parts, map[string]any{"text": text})
		sys["parts"] = parts
		req["systemInstruction"] = sys
	}
	return req
}
