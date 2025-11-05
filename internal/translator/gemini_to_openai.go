package translator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func init() {
	// Register Gemini → OpenAI translators
	Register(FormatGemini, FormatOpenAI, TranslatorConfig{
		ResponseTransform: GeminiToOpenAIResponse,
		StreamTransform:   GeminiToOpenAIStream,
	})
}

// GeminiToOpenAIResponse converts a non-streaming Gemini response to OpenAI format.
func GeminiToOpenAIResponse(ctx context.Context, model string, responseBody []byte) ([]byte, error) {
	result := gjson.ParseBytes(responseBody)

	// Check for errors
	if errMsg := result.Get("error"); errMsg.Exists() {
		return responseBody, nil // Pass through errors
	}

	// Extract candidates
	candidates := result.Get("candidates")
	if !candidates.Exists() {
		return responseBody, nil
	}

	var choices []map[string]interface{}
	var totalPromptTokens, totalCompletionTokens, reasoningTokens int64

	for idx, candidate := range candidates.Array() {
		content := candidate.Get("content")
		parts := content.Get("parts").Array()

		var messageContent strings.Builder
		var reasoningContent strings.Builder
		var toolCalls []map[string]interface{}
		hasThinking := false

		for _, part := range parts {
			// ✅ Check if this is a thinking/reasoning part
			if thought := part.Get("thought"); thought.Exists() {
				reasoningContent.WriteString(thought.String())
				hasThinking = true
				continue
			}

			// ✅ Check for reasoning metadata
			if execResult := part.Get("executableCode"); execResult.Exists() {
				// Code execution results are part of reasoning
				reasoningContent.WriteString(fmt.Sprintf("\n[Code Execution]\n%s\n", execResult.String()))
				hasThinking = true
				continue
			}

			if text := part.Get("text"); text.Exists() {
				textStr := text.String()
				// ✅ Detect thinking patterns in text
				if detectThinkingInText(textStr) {
					reasoningContent.WriteString(textStr)
					hasThinking = true
				} else {
					messageContent.WriteString(textStr)
				}
			}
			// ✅ Enhanced function call handling
			if fnCall := part.Get("functionCall"); fnCall.Exists() {
				fnName := fnCall.Get("name").String()
				fnArgs := fnCall.Get("args")

				// Convert args to JSON string
				var argsJSON []byte
				if fnArgs.Exists() {
					if fnArgs.IsObject() || fnArgs.IsArray() {
						argsJSON, _ = json.Marshal(fnArgs.Value())
					} else {
						argsJSON = []byte(fnArgs.Raw)
					}
				} else {
					argsJSON = []byte("{}")
				}

				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   fmt.Sprintf("call_%s_%d", fnName, len(toolCalls)),
					"type": "function",
					"function": map[string]interface{}{
						"name":      fnName,
						"arguments": string(argsJSON),
					},
				})
			}

			// ✅ Handle function response (convert back to content)
			if fnResp := part.Get("functionResponse"); fnResp.Exists() {
				// Function responses are typically in tool messages, not assistant
				// Skip them in assistant message conversion
				continue
			}
		}

		message := map[string]interface{}{
			"role":    "assistant",
			"content": messageContent.String(),
		}

		// ✅ Add reasoning_content if thinking was detected
		if hasThinking && reasoningContent.Len() > 0 {
			message["reasoning_content"] = reasoningContent.String()
		}

		if len(toolCalls) > 0 {
			message["tool_calls"] = toolCalls
		}

		finishReason := "stop"
		if fr := candidate.Get("finishReason"); fr.Exists() {
			switch fr.String() {
			case "STOP":
				finishReason = "stop"
			case "MAX_TOKENS":
				finishReason = "length"
			case "SAFETY":
				finishReason = "content_filter"
			case "RECITATION":
				finishReason = "content_filter"
			default:
				finishReason = "stop"
			}
		}
		if len(toolCalls) > 0 {
			finishReason = "tool_calls"
		}

		choices = append(choices, map[string]interface{}{
			"index":         idx,
			"message":       message,
			"finish_reason": finishReason,
		})
	}

	// Extract usage metadata
	if usage := result.Get("usageMetadata"); usage.Exists() {
		totalPromptTokens = usage.Get("promptTokenCount").Int()
		totalCompletionTokens = usage.Get("candidatesTokenCount").Int()
		// Gemini doesn't separate reasoning tokens, approximate if needed
	}

	response := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": choices,
		"usage": map[string]interface{}{
			"prompt_tokens":     totalPromptTokens,
			"completion_tokens": totalCompletionTokens,
			"total_tokens":      totalPromptTokens + totalCompletionTokens,
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": reasoningTokens,
			},
		},
	}

	return json.Marshal(response)
}

// GeminiToOpenAIStream converts a streaming Gemini response to OpenAI SSE format.
func GeminiToOpenAIStream(ctx context.Context, model string, reader io.Reader) (io.Reader, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

		chunkIndex := 0
		var accumulatedText strings.Builder
		var accumulatedReasoning strings.Builder

		for scanner.Scan() {
			line := scanner.Bytes()

			// Skip empty lines
			if len(line) == 0 {
				continue
			}

			// Parse SSE format: "data: {...}"
			if !bytes.HasPrefix(line, []byte("data: ")) {
				continue
			}

			jsonData := bytes.TrimPrefix(line, []byte("data: "))
			if bytes.Equal(jsonData, []byte("[DONE]")) {
				// Send final [DONE] marker
				pw.Write([]byte("data: [DONE]\n\n"))
				return
			}

			result := gjson.ParseBytes(jsonData)

			// Check for errors
			if errMsg := result.Get("error"); errMsg.Exists() {
				// Send error as OpenAI format
				errorChunk := map[string]interface{}{
					"error": map[string]interface{}{
						"message": errMsg.Get("message").String(),
						"type":    "server_error",
					},
				}
				errorJSON, _ := json.Marshal(errorChunk)
				pw.Write([]byte("data: "))
				pw.Write(errorJSON)
				pw.Write([]byte("\n\n"))
				return
			}

			// Extract candidates
			candidates := result.Get("candidates")
			if !candidates.Exists() {
				continue
			}

			for _, candidate := range candidates.Array() {
				content := candidate.Get("content")
				parts := content.Get("parts").Array()

				var delta map[string]interface{}
				var finishReason *string

				if chunkIndex == 0 {
					// First chunk includes role
					delta = map[string]interface{}{
						"role": "assistant",
					}
				} else {
					delta = map[string]interface{}{}
				}

				for _, part := range parts {
					// ✅ Handle thinking/reasoning parts in streaming
					if thought := part.Get("thought"); thought.Exists() {
						thoughtContent := thought.String()
						accumulatedReasoning.WriteString(thoughtContent)
						delta["reasoning_content"] = thoughtContent
						continue
					}

					if text := part.Get("text"); text.Exists() {
						textContent := text.String()
						// ✅ Detect if this is thinking content
						if detectThinkingInText(textContent) {
							accumulatedReasoning.WriteString(textContent)
							delta["reasoning_content"] = textContent
						} else {
							accumulatedText.WriteString(textContent)
							delta["content"] = textContent
						}
					}
					// ✅ Enhanced streaming function calls
					if fnCall := part.Get("functionCall"); fnCall.Exists() {
						fnName := fnCall.Get("name").String()
						fnArgs := fnCall.Get("args")

						var argsJSON []byte
						if fnArgs.Exists() {
							argsJSON, _ = json.Marshal(fnArgs.Value())
						} else {
							argsJSON = []byte("{}")
						}

						delta["tool_calls"] = []map[string]interface{}{
							{
								"index": 0,
								"id":    fmt.Sprintf("call_%s_%d", fnName, chunkIndex),
								"type":  "function",
								"function": map[string]interface{}{
									"name":      fnName,
									"arguments": string(argsJSON),
								},
							},
						}
					}
				}

				// Check finish reason
				if fr := candidate.Get("finishReason"); fr.Exists() {
					switch fr.String() {
					case "STOP":
						reason := "stop"
						finishReason = &reason
					case "MAX_TOKENS":
						reason := "length"
						finishReason = &reason
					case "SAFETY", "RECITATION":
						reason := "content_filter"
						finishReason = &reason
					}
				}

				// Build OpenAI chunk
				chunk := map[string]interface{}{
					"id":      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
					"object":  "chat.completion.chunk",
					"created": time.Now().Unix(),
					"model":   model,
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"delta": delta,
						},
					},
				}

				if finishReason != nil {
					chunk["choices"].([]map[string]interface{})[0]["finish_reason"] = *finishReason
				} else {
					chunk["choices"].([]map[string]interface{})[0]["finish_reason"] = nil
				}

				chunkJSON, _ := json.Marshal(chunk)
				pw.Write([]byte("data: "))
				pw.Write(chunkJSON)
				pw.Write([]byte("\n\n"))

				chunkIndex++
			}
		}

		if err := scanner.Err(); err != nil {
			// Log error but don't fail the stream
		}

		// Send final [DONE]
		pw.Write([]byte("data: [DONE]\n\n"))
	}()

	return pr, nil
}

// ✅ detectThinkingInText detects if text contains thinking/reasoning patterns
func detectThinkingInText(text string) bool {
	// Check for common thinking markers
	thinkingMarkers := []string{
		"<think>",
		"</think>",
		"<thinking>",
		"</thinking>",
		"[THINKING]",
		"[/THINKING]",
		"Let me think",
		"Let me analyze",
		"Step by step",
	}

	lowerText := strings.ToLower(text)
	for _, marker := range thinkingMarkers {
		if strings.Contains(lowerText, strings.ToLower(marker)) {
			return true
		}
	}

	// Check if text starts with thinking indicators
	trimmed := strings.TrimSpace(lowerText)
	if strings.HasPrefix(trimmed, "thinking:") ||
		strings.HasPrefix(trimmed, "reasoning:") ||
		strings.HasPrefix(trimmed, "analysis:") {
		return true
	}

	return false
}
