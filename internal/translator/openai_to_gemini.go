package translator

import (
	"encoding/json"

	"github.com/tidwall/sjson"
)

func init() {
	// Register OpenAI → Gemini translators
	Register(FormatOpenAI, FormatGemini, TranslatorConfig{
		RequestTransform: OpenAIToGeminiRequest,
	})
}

// OpenAIToGeminiRequest converts OpenAI chat completions request to Gemini format.
func OpenAIToGeminiRequest(model string, rawJSON []byte, stream bool) []byte { // stream kept for interface compatibility
	out := `{"contents":[]}`

	genConfig := buildGenerationConfig(rawJSON)
	genConfigJSON, _ := json.Marshal(genConfig)
	out, _ = sjson.SetRaw(out, "generationConfig", string(genConfigJSON))

	contents, systemInstructions := translateMessages(rawJSON)
	if shouldMergeAdjacent(rawJSON) {
		contents = mergeConsecutiveMessages(contents)
	}

	contentsJSON, _ := json.Marshal(contents)
	out, _ = sjson.SetRaw(out, "contents", string(contentsJSON))

	if len(systemInstructions) > 0 {
		sysJSON, _ := json.Marshal(map[string]interface{}{"parts": systemInstructions})
		out, _ = sjson.SetRaw(out, "systemInstruction", string(sysJSON))
	}

	out = applyToolDeclarations(out, rawJSON)
	out = applyResponseFormat(out, rawJSON)

	return []byte(out)
}
