package features

import (
	"encoding/json"
	"fmt"

	"gcli2api-go/internal/constants"
)

// RequestFormat represents the format of the request
type RequestFormat string

const (
	FormatOpenAI RequestFormat = "openai"
	FormatGemini RequestFormat = "gemini"
)

// FormatDetector detects and converts between request formats
type FormatDetector struct{}

// NewFormatDetector creates a new format detector
func NewFormatDetector() *FormatDetector {
	return &FormatDetector{}
}

// DetectFormat detects the format of a request body
func (fd *FormatDetector) DetectFormat(body []byte) (RequestFormat, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Check for Gemini format indicators
	if _, hasContents := data["contents"]; hasContents {
		return FormatGemini, nil
	}
	if _, hasSystemInstruction := data["systemInstruction"]; hasSystemInstruction {
		return FormatGemini, nil
	}
	if genConfig, ok := data["generationConfig"].(map[string]interface{}); ok {
		if _, hasMaxTokens := genConfig["maxOutputTokens"]; hasMaxTokens {
			return FormatGemini, nil
		}
	}

	// Check for OpenAI format indicators
	if _, hasMessages := data["messages"]; hasMessages {
		return FormatOpenAI, nil
	}

	// Default to OpenAI format
	return FormatOpenAI, nil
}

// ConvertOpenAIToGemini converts OpenAI format to Gemini format
func (fd *FormatDetector) ConvertOpenAIToGemini(body []byte) ([]byte, error) {
	var openaiReq map[string]interface{}
	if err := json.Unmarshal(body, &openaiReq); err != nil {
		return nil, fmt.Errorf("invalid OpenAI request: %w", err)
	}

	geminiReq := make(map[string]interface{})

	// Convert messages to contents
	if messages, ok := openaiReq["messages"].([]interface{}); ok {
		contents := make([]interface{}, 0)
		var systemInstruction map[string]interface{}

		for _, msg := range messages {
			msgMap, ok := msg.(map[string]interface{})
			if !ok {
				continue
			}

			role, _ := msgMap["role"].(string)
			content := msgMap["content"]

			// Handle system message
			if role == "system" {
				if contentStr, ok := content.(string); ok && contentStr != "" {
					systemInstruction = map[string]interface{}{
						"role": "user",
						"parts": []interface{}{
							map[string]interface{}{"text": contentStr},
						},
					}
				}
				continue
			}

			// Convert role
			geminiRole := role
			if role == "assistant" {
				geminiRole = "model"
			}

			// Build parts
			parts := make([]interface{}, 0)

			switch v := content.(type) {
			case string:
				if v != "" {
					parts = append(parts, map[string]interface{}{"text": v})
				}
			case []interface{}:
				for _, part := range v {
					partMap, ok := part.(map[string]interface{})
					if !ok {
						continue
					}

					partType, _ := partMap["type"].(string)
					switch partType {
					case "text":
						if text, ok := partMap["text"].(string); ok && text != "" {
							parts = append(parts, map[string]interface{}{"text": text})
						}
					case "image_url":
						if imgURL, ok := partMap["image_url"].(map[string]interface{}); ok {
							if url, ok := imgURL["url"].(string); ok {
								// Extract base64 data
								parts = append(parts, fd.convertImageURL(url))
							}
						}
					}
				}
			}

			if len(parts) > 0 {
				contents = append(contents, map[string]interface{}{
					"role":  geminiRole,
					"parts": parts,
				})
			}
		}

		geminiReq["contents"] = contents
		if systemInstruction != nil {
			geminiReq["systemInstruction"] = systemInstruction
		}
	}

	// Convert generation config
	genConfig := make(map[string]interface{})
	if temp, ok := openaiReq["temperature"].(float64); ok {
		genConfig["temperature"] = temp
	}
	if topP, ok := openaiReq["top_p"].(float64); ok {
		genConfig["topP"] = topP
	}
	topKValue := constants.DefaultTopK
	if raw, ok := openaiReq["top_k"]; ok {
		switch v := raw.(type) {
		case float64:
			topKValue = int(v)
		case int:
			topKValue = v
		case int64:
			topKValue = int(v)
		}
		if topKValue <= 0 {
			topKValue = constants.DefaultTopK
		}
		if topKValue > constants.MaxTopK {
			topKValue = constants.MaxTopK
		}
	}
	genConfig["topK"] = topKValue

	if maxTokens, ok := openaiReq["max_tokens"].(float64); ok {
		value := int(maxTokens)
		if value > constants.MaxOutputTokens {
			value = constants.MaxOutputTokens
		} else if value <= 0 {
			value = constants.MaxOutputTokens
		}
		genConfig["maxOutputTokens"] = value
	}
	if len(genConfig) > 0 {
		geminiReq["generationConfig"] = genConfig
	}

	// Convert tools
	if tools, ok := openaiReq["tools"].([]interface{}); ok && len(tools) > 0 {
		fd.convertTools(tools, geminiReq)
	}

	return json.Marshal(geminiReq)
}

// ConvertGeminiToOpenAI converts Gemini format to OpenAI format
func (fd *FormatDetector) ConvertGeminiToOpenAI(body []byte) ([]byte, error) {
	var geminiReq map[string]interface{}
	if err := json.Unmarshal(body, &geminiReq); err != nil {
		return nil, fmt.Errorf("invalid Gemini request: %w", err)
	}

	openaiReq := make(map[string]interface{})

	// Convert contents to messages
	messages := make([]interface{}, 0)

	// Add system instruction if present
	if sysInst, ok := geminiReq["systemInstruction"].(map[string]interface{}); ok {
		if parts, ok := sysInst["parts"].([]interface{}); ok && len(parts) > 0 {
			if textPart, ok := parts[0].(map[string]interface{}); ok {
				if text, ok := textPart["text"].(string); ok {
					messages = append(messages, map[string]interface{}{
						"role":    "system",
						"content": text,
					})
				}
			}
		}
	}

	// Convert contents
	if contents, ok := geminiReq["contents"].([]interface{}); ok {
		for _, content := range contents {
			contentMap, ok := content.(map[string]interface{})
			if !ok {
				continue
			}

			role, _ := contentMap["role"].(string)
			if role == "model" {
				role = "assistant"
			}

			parts, _ := contentMap["parts"].([]interface{})
			contentStr := fd.extractTextFromParts(parts)

			messages = append(messages, map[string]interface{}{
				"role":    role,
				"content": contentStr,
			})
		}
	}

	openaiReq["messages"] = messages

	// Convert generation config
	if genConfig, ok := geminiReq["generationConfig"].(map[string]interface{}); ok {
		if temp, ok := genConfig["temperature"].(float64); ok {
			openaiReq["temperature"] = temp
		}
		if topP, ok := genConfig["topP"].(float64); ok {
			openaiReq["top_p"] = topP
		}
		if maxTokens, ok := genConfig["maxOutputTokens"].(float64); ok {
			openaiReq["max_tokens"] = int(maxTokens)
		}
	}

	return json.Marshal(openaiReq)
}

// Helper methods
func (fd *FormatDetector) convertImageURL(url string) map[string]interface{} {
	// Simple implementation - extract base64 data from data URL
	// Format: data:image/png;base64,iVBORw0KG...
	if len(url) > 22 && url[:5] == "data:" {
		// Find mime type and data
		parts := url[5:] // Remove "data:"
		semiIdx := 0
		for i, c := range parts {
			if c == ';' {
				semiIdx = i
				break
			}
		}
		if semiIdx > 0 {
			mimeType := parts[:semiIdx]
			commaIdx := semiIdx
			for i := semiIdx; i < len(parts); i++ {
				if parts[i] == ',' {
					commaIdx = i
					break
				}
			}
			if commaIdx > semiIdx {
				data := parts[commaIdx+1:]
				return map[string]interface{}{
					"inlineData": map[string]interface{}{
						"mimeType": mimeType,
						"data":     data,
					},
				}
			}
		}
	}
	return map[string]interface{}{}
}

func (fd *FormatDetector) convertTools(tools []interface{}, geminiReq map[string]interface{}) {
	functionDeclarations := make([]interface{}, 0)

	for _, tool := range tools {
		toolMap, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}

		if toolType, _ := toolMap["type"].(string); toolType != "function" {
			continue
		}

		if fn, ok := toolMap["function"].(map[string]interface{}); ok {
			functionDeclarations = append(functionDeclarations, map[string]interface{}{
				"name":                 fn["name"],
				"description":          fn["description"],
				"parametersJsonSchema": fn["parameters"],
			})
		}
	}

	if len(functionDeclarations) > 0 {
		geminiReq["tools"] = []interface{}{
			map[string]interface{}{
				"functionDeclarations": functionDeclarations,
			},
		}
	}
}

func (fd *FormatDetector) extractTextFromParts(parts []interface{}) string {
	text := ""
	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}

		if textVal, ok := partMap["text"].(string); ok {
			text += textVal
		}
	}
	return text
}
