package models

import (
	cfgpkg "gcli2api-go/internal/config"
	"strings"
)

// ✅ ModelVariant represents different model variants and features
type ModelVariant struct {
	BaseName        string
	IsThinking      bool
	ThinkingMode    string // "max", "no", "auto"
	IsSearch        bool
	IsFakeStreaming bool
	IsAntiTrunc     bool
}

// ✅ ParseModelName parses a model name and extracts variants/features
func ParseModelName(modelName string) ModelVariant {
	variant := ModelVariant{
		BaseName: modelName,
	}

	// Check for fake streaming prefix
	if strings.HasPrefix(modelName, "假流式/") {
		variant.IsFakeStreaming = true
		modelName = strings.TrimPrefix(modelName, "假流式/")
	}

	// Check for anti-truncation prefix
	if strings.HasPrefix(modelName, "流式抗截断/") {
		variant.IsAntiTrunc = true
		modelName = strings.TrimPrefix(modelName, "流式抗截断/")
	}

	// Check for thinking mode suffixes
	if strings.HasSuffix(modelName, "-maxthinking") {
		variant.IsThinking = true
		variant.ThinkingMode = "max"
		modelName = strings.TrimSuffix(modelName, "-maxthinking")
	} else if strings.HasSuffix(modelName, "-nothinking") {
		variant.IsThinking = true
		variant.ThinkingMode = "no"
		modelName = strings.TrimSuffix(modelName, "-nothinking")
	}

	// Check for search mode suffix
	if strings.HasSuffix(modelName, "-search") {
		variant.IsSearch = true
		modelName = strings.TrimSuffix(modelName, "-search")
	}

	variant.BaseName = modelName
	return variant
}

// ✅ ApplyThinkingConfig applies thinking configuration to a request
func (v ModelVariant) ApplyThinkingConfig(genConfig map[string]interface{}) map[string]interface{} {
	if !v.IsThinking {
		return genConfig
	}

	thinkingConfig := make(map[string]interface{})

	switch v.ThinkingMode {
	case "max":
		// Maximum thinking budget
		thinkingConfig["thinkingBudget"] = 24576
		thinkingConfig["includeThoughts"] = true
	case "no":
		// No thinking
		thinkingConfig["thinkingBudget"] = 0
		// Don't include include_thoughts field
	default:
		// Auto/default thinking
		thinkingConfig["thinkingBudget"] = -1
		thinkingConfig["includeThoughts"] = true
	}

	genConfig["thinkingConfig"] = thinkingConfig
	return genConfig
}

// ✅ ApplySearchConfig applies search configuration to a request
func (v ModelVariant) ApplySearchConfig(request map[string]interface{}) map[string]interface{} {
	if !v.IsSearch {
		return request
	}

	// Enable Google Search grounding
	searchConfig := map[string]interface{}{
		"googleSearchRetrieval": map[string]interface{}{
			"dynamicRetrievalConfig": map[string]interface{}{
				"mode": "MODE_DYNAMIC",
			},
		},
	}

	if tools, ok := request["tools"].([]interface{}); ok {
		request["tools"] = append(tools, searchConfig)
	} else {
		request["tools"] = []interface{}{searchConfig}
	}

	return request
}

// DefaultBaseModels returns the fallback set of upstream base models.
func DefaultBaseModels() []string {
	return []string{
		"gemini-2.5-pro",
		"gemini-2.5-pro-preview-06-05",
		"gemini-2.5-pro-preview-05-06",
		"gemini-2.5-flash",
		"gemini-2.5-flash-preview-09-2025",
		"gemini-2.5-flash-image",
		"gemini-2.5-flash-image-preview",
	}
}

// ✅ GetAvailableModels returns list of all available models with variants
func GetAvailableModels() []string {
	baseModels := DefaultBaseModels()

	models := make([]string, 0)

	for _, base := range baseModels {
		// Base model
		models = append(models, base)

		// Thinking variants
		models = append(models, base+"-maxthinking")
		models = append(models, base+"-nothinking")

		// Search variant
		models = append(models, base+"-search")

		// Fake streaming variants
		models = append(models, "假流式/"+base)
		models = append(models, "假流式/"+base+"-maxthinking")
		models = append(models, "假流式/"+base+"-nothinking")
		models = append(models, "假流式/"+base+"-search")

		// Anti-truncation variants
		models = append(models, "流式抗截断/"+base)
		models = append(models, "流式抗截断/"+base+"-maxthinking")
		models = append(models, "流式抗截断/"+base+"-search")
	}

	return models
}

// ✅ IsValidModel checks if a model name is valid (after parsing)
func IsValidModel(modelName string) bool {
	variant := ParseModelName(modelName)

	validBases := map[string]bool{
		"gemini-2.5-pro":                   true,
		"gemini-2.5-pro-preview-06-05":     true,
		"gemini-2.5-pro-preview-05-06":     true,
		"gemini-2.5-flash":                 true,
		"gemini-2.5-flash-preview-09-2025": true,
		"gemini-2.5-flash-image":           true,
		"gemini-2.5-flash-image-preview":   true,
	}

	return validBases[variant.BaseName]
}

// FallbackBases returns base-model fallback order for a given base name (no prefixes/suffixes).
// Example:
//
//	FallbackBases("gemini-2.5-pro") => [pro, pro-preview-06-05, pro-preview-05-06, flash]
//	FallbackBases("gemini-2.5-flash") => [flash, flash-preview-09-2025]
//	FallbackBases("gemini-2.5-flash-image") => [flash-image, flash-image-preview]
func FallbackBases(base string) []string {
	order := []string{}
	push := func(s string) {
		if s == "" {
			return
		}
		for _, e := range order {
			if e == s {
				return
			}
		}
		order = append(order, s)
	}
	lower := strings.ToLower(base)
	// Config override: allow admin to define replacement list
	if cfg := cfgpkg.Load(); cfg != nil {
		// expose via PreferredBaseModels semantics? keep a dedicated override when added to Config in future
		// For now, no dynamic override here to avoid nil fields; keep default table
	}
	switch lower {
	case "gemini-2.5-pro":
		push("gemini-2.5-pro")
		push("gemini-2.5-pro-preview-06-05")
		push("gemini-2.5-pro-preview-05-06")
		push("gemini-2.5-flash")
	case "gemini-2.5-pro-preview-06-05", "gemini-2.5-pro-preview-05-06":
		push(lower)
		if lower == "gemini-2.5-pro-preview-06-05" {
			push("gemini-2.5-pro-preview-05-06")
		} else {
			push("gemini-2.5-pro-preview-06-05")
		}
		push("gemini-2.5-pro")
		push("gemini-2.5-flash")
	case "gemini-2.5-flash":
		push("gemini-2.5-flash")
		push("gemini-2.5-flash-preview-09-2025")
	case "gemini-2.5-flash-image":
		push("gemini-2.5-flash-image")
		push("gemini-2.5-flash-image-preview")
	case "gemini-2.5-flash-image-preview":
		push("gemini-2.5-flash-image-preview")
		push("gemini-2.5-flash-image")
	default:
		push(base)
	}
	return order
}

// FallbackOrder returns full-feature fallback order, preserving prefixes/suffixes of the requested model.
func FallbackOrder(model string) []string {
	v := ParseModelName(model)
	bases := FallbackBases(v.BaseName)

	// Re-apply suffixes
	suffix := ""
	if v.IsThinking {
		switch v.ThinkingMode {
		case "max":
			suffix += "-maxthinking"
		case "no":
			suffix += "-nothinking"
		}
	}
	if v.IsSearch {
		suffix += "-search"
	}

	// Re-apply prefix
	prefix := ""
	if v.IsFakeStreaming {
		prefix = "假流式/"
	}
	if v.IsAntiTrunc {
		prefix = "流式抗截断/"
	}

	out := make([]string, 0, len(bases))
	for _, b := range bases {
		out = append(out, prefix+b+suffix)
	}
	return out
}
