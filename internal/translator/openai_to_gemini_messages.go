package translator

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
)

func translateMessages(rawJSON []byte) ([]interface{}, []interface{}) {
	messages := gjson.GetBytes(rawJSON, "messages")
	var contents []interface{}
	var systemInstructions []interface{}

	for _, msg := range messages.Array() {
		role := msg.Get("role").String()
		content := msg.Get("content")

		switch role {
		case "system":
			if content.IsArray() {
				for _, part := range content.Array() {
					systemInstructions = append(systemInstructions, convertContentPart(part))
				}
			} else {
				systemInstructions = append(systemInstructions, map[string]interface{}{
					"text": sanitizeText(content.String()),
				})
			}

		case "user":
			geminiMsg := map[string]interface{}{
				"role":  "user",
				"parts": []interface{}{},
			}
			if content.IsArray() {
				var parts []interface{}
				for _, part := range content.Array() {
					parts = append(parts, convertContentPart(part))
				}
				geminiMsg["parts"] = parts
			} else {
				geminiMsg["parts"] = []interface{}{
					map[string]interface{}{"text": sanitizeText(content.String())},
				}
			}
			contents = append(contents, geminiMsg)

		case "assistant":
			geminiMsg := map[string]interface{}{
				"role":  "model",
				"parts": []interface{}{},
			}

			if toolCalls := msg.Get("tool_calls"); toolCalls.Exists() && toolCalls.IsArray() {
				var parts []interface{}
				for _, tc := range toolCalls.Array() {
					if tc.Get("type").String() == "function" {
						fnName := tc.Get("function.name").String()
						fnArgs := tc.Get("function.arguments").String()
						var argsObj interface{}
						if err := json.Unmarshal([]byte(fnArgs), &argsObj); err == nil {
							parts = append(parts, map[string]interface{}{
								"functionCall": map[string]interface{}{
									"name": fnName,
									"args": argsObj,
								},
							})
						}
					}
				}

				if content.Exists() && content.String() != "" {
					parts = append([]interface{}{
						map[string]interface{}{"text": sanitizeText(content.String())},
					}, parts...)
				}

				geminiMsg["parts"] = parts
			} else if content.Exists() {
				if content.IsArray() {
					var parts []interface{}
					for _, part := range content.Array() {
						parts = append(parts, convertContentPart(part))
					}
					geminiMsg["parts"] = parts
				} else if content.String() != "" {
					geminiMsg["parts"] = []interface{}{
						map[string]interface{}{"text": sanitizeText(content.String())},
					}
				}
			}

			if parts, ok := geminiMsg["parts"].([]interface{}); ok && len(parts) > 0 {
				contents = append(contents, geminiMsg)
			}

		case "tool":
			toolCallID := msg.Get("tool_call_id").String()
			name := msg.Get("name").String()

			var responseContent interface{}
			contentStr := sanitizeText(content.String())
			if err := json.Unmarshal([]byte(contentStr), &responseContent); err != nil {
				responseContent = map[string]interface{}{
					"result": contentStr,
				}
			}

			funcResp := map[string]interface{}{
				"functionResponse": map[string]interface{}{
					"name":     name,
					"response": responseContent,
				},
			}

			if toolCallID != "" {
				funcResp["functionResponse"].(map[string]interface{})["id"] = toolCallID
			}

			geminiMsg := map[string]interface{}{
				"role":  "user",
				"parts": []interface{}{funcResp},
			}
			contents = append(contents, geminiMsg)
		}
	}

	contents = sanitizeMessages(contents)
	ensureDoneInstruction(&systemInstructions)
	systemInstructions = sanitizeParts(systemInstructions)
	return contents, systemInstructions
}

// convertContentPart converts an OpenAI content part to Gemini format (enhanced).
func convertContentPart(part gjson.Result) interface{} {
	partType := part.Get("type").String()

	switch partType {
	case "text":
		return map[string]interface{}{
			"text": sanitizeText(part.Get("text").String()),
		}

	case "image_url":
		imageURL := part.Get("image_url.url").String()
		detail := part.Get("image_url.detail").String()

		if strings.HasPrefix(imageURL, "data:") {
			parts := strings.SplitN(imageURL, ",", 2)
			if len(parts) == 2 {
				mimeType := detectImageMIME(parts[0])
				inlineData := map[string]interface{}{
					"mimeType": mimeType,
					"data":     parts[1],
				}
				return map[string]interface{}{"inlineData": inlineData}
			}
		}

		fileData := map[string]interface{}{
			"fileUri": imageURL,
		}
		if detail != "" {
			fileData["detail"] = detail
		}
		return map[string]interface{}{"fileData": fileData}

	case "audio":
		if audioData := part.Get("audio"); audioData.Exists() {
			if audioData.Get("data").Exists() {
				return map[string]interface{}{
					"inlineData": map[string]interface{}{
						"mimeType": part.Get("audio.format").String(),
						"data":     part.Get("audio.data").String(),
					},
				}
			}
		}

	case "video":
		if videoURL := part.Get("video.url"); videoURL.Exists() {
			return map[string]interface{}{
				"fileData": map[string]interface{}{
					"fileUri": videoURL.String(),
				},
			}
		}
	}

	var result interface{}
	if err := json.Unmarshal([]byte(part.Raw), &result); err == nil {
		return result
	}

	return map[string]interface{}{
		"text": sanitizeText(part.Raw),
	}
}

func mergeConsecutiveMessages(contents []interface{}) []interface{} {
	if len(contents) <= 1 {
		return contents
	}

	merged := make([]interface{}, 0, len(contents))
	var current map[string]interface{}

	for i, item := range contents {
		msg, ok := item.(map[string]interface{})
		if !ok {
			merged = append(merged, item)
			continue
		}

		role, hasRole := msg["role"].(string)
		if !hasRole {
			merged = append(merged, msg)
			continue
		}

		if current == nil || current["role"].(string) != role {
			if current != nil {
				merged = append(merged, current)
			}
			current = msg
			continue
		}

		currentParts, hasParts := current["parts"].([]interface{})
		msgParts, hasMsgParts := msg["parts"].([]interface{})

		if hasParts && hasMsgParts {
			current["parts"] = append(currentParts, msgParts...)
		} else if hasMsgParts {
			current["parts"] = msgParts
		}

		if i == len(contents)-1 {
			merged = append(merged, current)
		}
	}

	if current != nil {
		merged = append(merged, current)
	}

	return merged
}

func detectImageMIME(prefix string) string {
	switch {
	case strings.Contains(prefix, "image/png"):
		return "image/png"
	case strings.Contains(prefix, "image/webp"):
		return "image/webp"
	case strings.Contains(prefix, "image/gif"):
		return "image/gif"
	case strings.Contains(prefix, "image/heic"):
		return "image/heic"
	case strings.Contains(prefix, "image/heif"):
		return "image/heif"
	default:
		return "image/jpeg"
	}
}
