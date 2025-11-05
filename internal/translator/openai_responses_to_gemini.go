package translator

import (
	"encoding/json"
	"strings"

	"gcli2api-go/internal/constants"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// OpenAIResponsesToGeminiRequest converts OpenAI Responses API request to Gemini request JSON.
func OpenAIResponsesToGeminiRequest(model string, rawJSON []byte, _ bool) []byte {
	out := `{"contents":[]}`

	// generation config
	gen := map[string]any{"candidateCount": 1}
	if v := gjson.GetBytes(rawJSON, "temperature"); v.Exists() {
		gen["temperature"] = v.Value()
	}
	if v := gjson.GetBytes(rawJSON, "top_p"); v.Exists() {
		gen["topP"] = v.Value()
	}
	topKValue := constants.DefaultTopK
	if v := gjson.GetBytes(rawJSON, "top_k"); v.Exists() {
		value := int(v.Int())
		if value <= 0 {
			value = constants.DefaultTopK
		}
		if value > constants.MaxTopK {
			value = constants.MaxTopK
		}
		topKValue = value
	}
	gen["topK"] = topKValue

	maxTokensValue := -1
	if v := gjson.GetBytes(rawJSON, "max_output_tokens"); v.Exists() {
		maxTokensValue = int(v.Int())
	} else if v := gjson.GetBytes(rawJSON, "max_tokens"); v.Exists() {
		maxTokensValue = int(v.Int())
	}
	if maxTokensValue > 0 {
		if maxTokensValue > constants.MaxOutputTokens {
			maxTokensValue = constants.MaxOutputTokens
		}
		gen["maxOutputTokens"] = maxTokensValue
	}
	if v := gjson.GetBytes(rawJSON, "frequency_penalty"); v.Exists() {
		gen["frequencyPenalty"] = v.Value()
	}
	if v := gjson.GetBytes(rawJSON, "presence_penalty"); v.Exists() {
		gen["presencePenalty"] = v.Value()
	}
	if v := gjson.GetBytes(rawJSON, "n"); v.Exists() {
		gen["candidateCount"] = int(v.Int())
	}
	if v := gjson.GetBytes(rawJSON, "seed"); v.Exists() {
		gen["seed"] = int(v.Int())
	}
	if stop := gjson.GetBytes(rawJSON, "stop"); stop.Exists() {
		if stop.IsArray() {
			var arr []any
			for _, s := range stop.Array() {
				arr = append(arr, s.String())
			}
			gen["stopSequences"] = arr
		} else {
			gen["stopSequences"] = []any{stop.String()}
		}
	}
	genJSON, _ := json.Marshal(gen)
	out, _ = sjson.SetRaw(out, "generationConfig", string(genJSON))

	// system instruction
	if inst := gjson.GetBytes(rawJSON, "instructions"); inst.Exists() && inst.String() != "" {
		sys := map[string]any{"parts": []any{map[string]any{"text": inst.String()}}}
		sysJSON, _ := json.Marshal(sys)
		out, _ = sjson.SetRaw(out, "systemInstruction", string(sysJSON))
	}

	// input: string or array of typed items
	var contents []any
	if in := gjson.GetBytes(rawJSON, "input"); in.Exists() {
		if in.Type == gjson.String {
			contents = append(contents, map[string]any{"role": "user", "parts": []any{map[string]any{"text": in.String()}}})
		} else if in.IsArray() {
			node := map[string]any{"role": "user", "parts": []any{}}
			for _, it := range in.Array() {
				t := it.Get("type").String()
				switch t {
				case "message":
					// message has role and content array
					role := strings.ToLower(it.Get("role").String())
					if role == "assistant" || role == "model" {
						node["role"] = "model"
					} else {
						node["role"] = "user"
					}
					if content := it.Get("content"); content.Exists() && content.IsArray() {
						for _, ci := range content.Array() {
							if txt := ci.Get("text"); txt.Exists() && txt.String() != "" {
								node["parts"] = append(node["parts"].([]any), map[string]any{"text": txt.String()})
							}
						}
					}
				case "input_text", "text", "output_text":
					if txt := it.Get("text").String(); txt != "" {
						node["parts"] = append(node["parts"].([]any), map[string]any{"text": txt})
					}
				case "input_image", "image_url":
					url := it.Get("image_url.url").String()
					if strings.HasPrefix(url, "data:") {
						rest := strings.TrimPrefix(url, "data:")
						semi := strings.Index(rest, ";")
						comma := strings.LastIndex(rest, ",")
						if semi > 0 && comma > semi {
							mime := rest[:semi]
							data := rest[comma+1:]
							node["parts"] = append(node["parts"].([]any), map[string]any{"inlineData": map[string]any{"mimeType": mime, "data": data}})
						}
					} else if url != "" {
						node["parts"] = append(node["parts"].([]any), map[string]any{"fileData": map[string]any{"fileUri": url}})
					}
				}
			}
			if parts, _ := node["parts"].([]any); len(parts) > 0 {
				contents = append(contents, node)
			}
		}
	}

	// tools -> functionDeclarations
	if tools := gjson.GetBytes(rawJSON, "tools"); tools.Exists() && tools.IsArray() {
		var fdecl []any
		for _, t := range tools.Array() {
			if t.Get("type").String() != "function" {
				continue
			}
			fn := t.Get("function")
			fdecl = append(fdecl, map[string]any{
				"name":                 fn.Get("name").String(),
				"description":          fn.Get("description").String(),
				"parametersJsonSchema": json.RawMessage(fn.Get("parameters").Raw),
			})
		}
		if len(fdecl) > 0 {
			out, _ = sjson.SetRaw(out, "tools", mustJSON([]any{map[string]any{"functionDeclarations": fdecl}}))
		}
	}

	if len(contents) > 0 {
		out, _ = sjson.SetRaw(out, "contents", mustJSON(contents))
	}
	return []byte(out)
}

// OpenAICompletionsToGeminiRequest converts legacy OpenAI completions prompt into Gemini request.
func OpenAICompletionsToGeminiRequest(model string, rawJSON []byte, _ bool) []byte {
	out := `{"contents":[]}`
	gen := map[string]any{"candidateCount": 1}
	if v := gjson.GetBytes(rawJSON, "temperature"); v.Exists() {
		gen["temperature"] = v.Value()
	}
	if v := gjson.GetBytes(rawJSON, "top_p"); v.Exists() {
		gen["topP"] = v.Value()
	}
	topKValue := constants.DefaultTopK
	if v := gjson.GetBytes(rawJSON, "top_k"); v.Exists() {
		value := int(v.Int())
		if value <= 0 {
			value = constants.DefaultTopK
		}
		if value > constants.MaxTopK {
			value = constants.MaxTopK
		}
		topKValue = value
	}
	gen["topK"] = topKValue

	if v := gjson.GetBytes(rawJSON, "max_tokens"); v.Exists() {
		value := int(v.Int())
		if value > constants.MaxOutputTokens {
			value = constants.MaxOutputTokens
		} else if value <= 0 {
			value = constants.MaxOutputTokens
		}
		gen["maxOutputTokens"] = value
	}
	if v := gjson.GetBytes(rawJSON, "frequency_penalty"); v.Exists() {
		gen["frequencyPenalty"] = v.Value()
	}
	if v := gjson.GetBytes(rawJSON, "presence_penalty"); v.Exists() {
		gen["presencePenalty"] = v.Value()
	}
	if v := gjson.GetBytes(rawJSON, "n"); v.Exists() {
		gen["candidateCount"] = int(v.Int())
	}
	if v := gjson.GetBytes(rawJSON, "seed"); v.Exists() {
		gen["seed"] = int(v.Int())
	}
	if stop := gjson.GetBytes(rawJSON, "stop"); stop.Exists() {
		if stop.IsArray() {
			var arr []any
			for _, s := range stop.Array() {
				arr = append(arr, s.String())
			}
			gen["stopSequences"] = arr
		} else {
			gen["stopSequences"] = []any{stop.String()}
		}
	}
	out, _ = sjson.SetRaw(out, "generationConfig", mustJSON(gen))
	prompt := gjson.GetBytes(rawJSON, "prompt").String()
	if prompt != "" {
		out, _ = sjson.SetRaw(out, "contents", mustJSON([]any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": prompt}}}}))
	}
	return []byte(out)
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
