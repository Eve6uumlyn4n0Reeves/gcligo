package streaming

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"gcli2api-go/internal/common"
)

// FakeStreamConfig holds configuration for fake streaming
type FakeStreamConfig struct {
	ChunkSize   int           // Characters per chunk
	ChunkDelay  time.Duration // Delay between chunks
	IncludeRole bool          // Include role in first chunk
	IncludeDone bool          // Include [DONE] marker at end
}

// DefaultFakeStreamConfig returns default configuration
func DefaultFakeStreamConfig() FakeStreamConfig {
	return FakeStreamConfig{
		ChunkSize:   20,
		ChunkDelay:  50 * time.Millisecond,
		IncludeRole: true,
		IncludeDone: true,
	}
}

// ConvertToFakeStream converts a complete response to SSE streaming format
func ConvertToFakeStream(ctx context.Context, completeResponse []byte, model string, cfg FakeStreamConfig) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		// Parse the complete response
		var response map[string]interface{}
		if err := json.Unmarshal(completeResponse, &response); err != nil {
			// If parsing fails, just return empty stream
			if cfg.IncludeDone {
				pw.Write([]byte("data: [DONE]\n\n"))
			}
			return
		}

		// Extract message content
		choices, ok := response["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			if cfg.IncludeDone {
				pw.Write([]byte("data: [DONE]\n\n"))
			}
			return
		}

		firstChoice, ok := choices[0].(map[string]interface{})
		if !ok {
			if cfg.IncludeDone {
				pw.Write([]byte("data: [DONE]\n\n"))
			}
			return
		}

		message, ok := firstChoice["message"].(map[string]interface{})
		if !ok {
			if cfg.IncludeDone {
				pw.Write([]byte("data: [DONE]\n\n"))
			}
			return
		}

		content, _ := message["content"].(string)
		finishReason, _ := firstChoice["finish_reason"].(string)

		// Split content into chunks
		chunks := splitIntoChunks(content, cfg.ChunkSize)

		for i, chunk := range chunks {
			select {
			case <-ctx.Done():
				return
			default:
			}

			delta := map[string]interface{}{}

			// First chunk includes role
			if i == 0 && cfg.IncludeRole {
				delta["role"] = "assistant"
			}

			if chunk != "" {
				delta["content"] = chunk
			}

			streamChunk := map[string]interface{}{
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

			// Last chunk includes finish_reason
			if i == len(chunks)-1 {
				streamChunk["choices"].([]map[string]interface{})[0]["finish_reason"] = finishReason
			} else {
				streamChunk["choices"].([]map[string]interface{})[0]["finish_reason"] = nil
			}

			chunkJSON, _ := json.Marshal(streamChunk)
			pw.Write([]byte("data: "))
			pw.Write(chunkJSON)
			pw.Write([]byte("\n\n"))

			// Add delay between chunks (except for the last one)
			if i < len(chunks)-1 && cfg.ChunkDelay > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(cfg.ChunkDelay):
				}
			}
		}

		// Send [DONE] marker
		if cfg.IncludeDone {
			pw.Write([]byte("data: [DONE]\n\n"))
		}
	}()

	return pr
}

// splitIntoChunks splits text into chunks of approximately the given size
func splitIntoChunks(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		chunkSize = 20
	}

	var chunks []string
	runes := []rune(text)

	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		chunks = append(chunks, chunk)
	}

	// Ensure at least one chunk
	if len(chunks) == 0 {
		chunks = append(chunks, "")
	}

	return chunks
}

// IsCompleteStream checks if a stream is complete (ended gracefully)
func IsCompleteStream(reader io.Reader) bool {
	scanner := bufio.NewScanner(reader)
	lastLine := ""

	for scanner.Scan() {
		lastLine = scanner.Text()
	}

	return common.HasDoneMarker(lastLine) ||
		strings.Contains(lastLine, "finish_reason")
}

// ExtractTextFromStream extracts all text content from a streaming response
func ExtractTextFromStream(reader io.Reader) (string, error) {
	var fullText strings.Builder
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

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
		if bytes.EqualFold(jsonData, []byte("[DONE]")) {
			break
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal(jsonData, &chunk); err != nil {
			continue
		}

		// Extract text from delta
		choices, ok := chunk["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			continue
		}

		firstChoice, ok := choices[0].(map[string]interface{})
		if !ok {
			continue
		}

		delta, ok := firstChoice["delta"].(map[string]interface{})
		if !ok {
			continue
		}

		if content, ok := delta["content"].(string); ok {
			fullText.WriteString(content)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return fullText.String(), nil
}
