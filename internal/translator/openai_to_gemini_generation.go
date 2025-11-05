package translator

import (
	"strings"

	"gcli2api-go/internal/constants"
	"github.com/tidwall/gjson"
)

func buildGenerationConfig(rawJSON []byte) map[string]interface{} {
	genConfig := make(map[string]interface{})
	genConfig["candidateCount"] = 1

	if temp := gjson.GetBytes(rawJSON, "temperature"); temp.Exists() {
		genConfig["temperature"] = temp.Value()
	}
	if topP := gjson.GetBytes(rawJSON, "top_p"); topP.Exists() {
		genConfig["topP"] = topP.Value()
	}
	topKValue := constants.DefaultTopK
	if topK := gjson.GetBytes(rawJSON, "top_k"); topK.Exists() {
		value := int(topK.Int())
		if value <= 0 {
			value = constants.DefaultTopK
		}
		if value > constants.MaxTopK {
			value = constants.MaxTopK
		}
		topKValue = value
	}
	genConfig["topK"] = topKValue

	maxTokensValue := -1
	if maxTokens := gjson.GetBytes(rawJSON, "max_tokens"); maxTokens.Exists() {
		maxTokensValue = int(maxTokens.Int())
	}
	if maxCompTokens := gjson.GetBytes(rawJSON, "max_completion_tokens"); maxCompTokens.Exists() {
		maxTokensValue = int(maxCompTokens.Int())
	}
	if maxTokensValue > 0 {
		if maxTokensValue > constants.MaxOutputTokens {
			maxTokensValue = constants.MaxOutputTokens
		}
		genConfig["maxOutputTokens"] = maxTokensValue
	}

	// Additional OpenAI params → Gemini generationConfig
	if fp := gjson.GetBytes(rawJSON, "frequency_penalty"); fp.Exists() {
		genConfig["frequencyPenalty"] = fp.Value()
	}
	if pp := gjson.GetBytes(rawJSON, "presence_penalty"); pp.Exists() {
		genConfig["presencePenalty"] = pp.Value()
	}
	if n := gjson.GetBytes(rawJSON, "n"); n.Exists() {
		genConfig["candidateCount"] = int(n.Int())
	}
	if seed := gjson.GetBytes(rawJSON, "seed"); seed.Exists() {
		genConfig["seed"] = int(seed.Int())
	}

	if reasoningEffort := gjson.GetBytes(rawJSON, "reasoning_effort"); reasoningEffort.Exists() {
		genConfig["thinkingConfig"] = buildThinkingConfig(reasoningEffort.String())
	}

	if mods := gjson.GetBytes(rawJSON, "modalities"); mods.Exists() {
		if responseMods := mapModalities(mods.Array()); len(responseMods) > 0 {
			genConfig["responseModalities"] = responseMods
		}
	}

	if imgCfg := gjson.GetBytes(rawJSON, "image_config"); imgCfg.Exists() {
		if aspect := imgCfg.Get("aspect_ratio"); aspect.Exists() {
			genConfig["responseImageAspectRatio"] = aspect.String()
		}
	}

	if stop := gjson.GetBytes(rawJSON, "stop"); stop.Exists() {
		if stopSeqs := collectStopSequences(stop); len(stopSeqs) > 0 {
			genConfig["stopSequences"] = stopSeqs
		}
	}

	return genConfig
}

func buildThinkingConfig(effort string) map[string]interface{} {
	thinkingConfig := make(map[string]interface{})

	switch effort {
	case "none":
		thinkingConfig["thinkingBudget"] = 0
	case "auto":
		thinkingConfig["thinkingBudget"] = -1
		thinkingConfig["includeThoughts"] = true
	case "low":
		thinkingConfig["thinkingBudget"] = 1024
		thinkingConfig["includeThoughts"] = true
	case "medium":
		thinkingConfig["thinkingBudget"] = 8192
		thinkingConfig["includeThoughts"] = true
	case "high":
		thinkingConfig["thinkingBudget"] = 24576
		thinkingConfig["includeThoughts"] = true
	default:
		thinkingConfig["thinkingBudget"] = -1
		thinkingConfig["includeThoughts"] = true
	}
	return thinkingConfig
}

func mapModalities(mods []gjson.Result) []string {
	var responseMods []string
	for _, m := range mods {
		switch strings.ToLower(m.String()) {
		case "text":
			responseMods = append(responseMods, "Text")
		case "image":
			responseMods = append(responseMods, "Image")
		}
	}
	return responseMods
}

func collectStopSequences(stop gjson.Result) []string {
	var stopSeqs []string
	if stop.IsArray() {
		for _, s := range stop.Array() {
			stopSeqs = append(stopSeqs, s.String())
		}
	} else {
		stopSeqs = append(stopSeqs, stop.String())
	}
	return stopSeqs
}

func shouldMergeAdjacent(rawJSON []byte) bool {
	merge := true
	if v := gjson.GetBytes(rawJSON, "compat_merge_adjacent"); v.Exists() {
		if v.Type == gjson.False {
			merge = false
		}
	}
	return merge
}
