package translator

import (
	"encoding/json"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func applyToolDeclarations(out string, rawJSON []byte) string {
	if tools := gjson.GetBytes(rawJSON, "tools"); tools.Exists() {
		var geminiTools []interface{}
		for _, tool := range tools.Array() {
			if tool.Get("type").String() == "function" {
				fn := tool.Get("function")
				geminiTools = append(geminiTools, map[string]interface{}{
					"functionDeclarations": []interface{}{
						map[string]interface{}{
							"name":        fn.Get("name").String(),
							"description": fn.Get("description").String(),
							"parameters":  json.RawMessage(fn.Get("parameters").Raw),
						},
					},
				})
			}
		}
		if len(geminiTools) > 0 {
			toolsJSON, _ := json.Marshal(geminiTools)
			out, _ = sjson.SetRaw(out, "tools", string(toolsJSON))
		}
	}
	return out
}

func applyResponseFormat(out string, rawJSON []byte) string {
	if respFormat := gjson.GetBytes(rawJSON, "response_format"); respFormat.Exists() {
		switch respFormat.Get("type").String() {
		case "json_object":
			out, _ = sjson.Set(out, "generationConfig.responseMimeType", "application/json")
		case "json_schema":
			out, _ = sjson.Set(out, "generationConfig.responseMimeType", "application/json")
			if schema := respFormat.Get("json_schema.schema"); schema.Exists() {
				out, _ = sjson.SetRaw(out, "generationConfig.responseSchema", schema.Raw)
			}
		}
	}
	return out
}
