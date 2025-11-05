package gemini

import (
	"strings"

	"github.com/tidwall/sjson"
)

// fixGeminiCLIImageHints ensures minimal CLI-aligned hints for image models.
//   - For flash-image variants, enforce responseModalities to include Image to avoid text-only defaults.
//     This is a no-op when the field already exists.
func fixGeminiCLIImageHints(model string, raw []byte) []byte {
	lower := strings.ToLower(model)
	if strings.Contains(lower, "flash-image") {
		if out, err := sjson.SetBytes(raw, "generationConfig.responseModalities", []string{"Image"}); err == nil {
			return out
		}
	}
	return raw
}

// deleteJSONField removes a JSON path (dot notation) from a payload using sjson.
func deleteJSONField(body []byte, path string) []byte {
	if strings.TrimSpace(path) == "" {
		return body
	}
	out, err := sjson.DeleteBytes(body, path)
	if err != nil {
		return body
	}
	return out
}

// geminiModelDisallowsThinking returns true for models where thinkingConfig should be stripped.
func geminiModelDisallowsThinking(model string) bool {
	if model == "" {
		return false
	}
	lower := strings.ToLower(model)
	if strings.Contains(lower, "gemini-2.5-flash-image-preview") || strings.Contains(lower, "gemini-2.5-flash-image") {
		return true
	}
	return false
}
