package streaming

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func TestDefaultFakeStreamConfig(t *testing.T) {
	cfg := DefaultFakeStreamConfig()

	if cfg.ChunkSize != 20 {
		t.Errorf("Expected ChunkSize 20, got %d", cfg.ChunkSize)
	}
	if cfg.ChunkDelay != 50*time.Millisecond {
		t.Errorf("Expected ChunkDelay 50ms, got %v", cfg.ChunkDelay)
	}
	if !cfg.IncludeRole {
		t.Error("Expected IncludeRole to be true")
	}
	if !cfg.IncludeDone {
		t.Error("Expected IncludeDone to be true")
	}
}

func TestSplitIntoChunks(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		chunkSize int
		expected  int
	}{
		{"Empty text", "", 10, 1},
		{"Short text", "Hello", 10, 1},
		{"Exact chunk", "1234567890", 10, 1},
		{"Multiple chunks", "Hello, World! This is a test.", 10, 3},
		{"Unicode text", "你好世界", 2, 2},
		{"Zero chunk size", "Hello", 0, 1},
		{"Negative chunk size", "Hello", -5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := splitIntoChunks(tt.text, tt.chunkSize)
			if len(chunks) != tt.expected {
				t.Errorf("Expected %d chunks, got %d", tt.expected, len(chunks))
			}

			// Verify chunks reconstruct original text
			reconstructed := strings.Join(chunks, "")
			if reconstructed != tt.text {
				t.Errorf("Reconstructed text doesn't match. Expected %q, got %q", tt.text, reconstructed)
			}
		})
	}
}

func TestConvertToFakeStream(t *testing.T) {
	tests := []struct {
		name     string
		response string
		model    string
		cfg      FakeStreamConfig
		validate func(t *testing.T, output string)
	}{
		{
			name: "Valid response",
			response: `{
				"choices": [{
					"message": {"content": "Hello, world!"},
					"finish_reason": "stop"
				}]
			}`,
			model: "test-model",
			cfg:   DefaultFakeStreamConfig(),
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "data: ") {
					t.Error("Output should contain SSE data prefix")
				}
				if !strings.Contains(output, "Hello, world!") {
					t.Error("Output should contain original content")
				}
				if !strings.Contains(output, "[DONE]") {
					t.Error("Output should contain [DONE] marker")
				}
				if !strings.Contains(output, "test-model") {
					t.Error("Output should contain model name")
				}
			},
		},
		{
			name:     "Invalid JSON",
			response: `{invalid json}`,
			model:    "test-model",
			cfg:      DefaultFakeStreamConfig(),
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "[DONE]") {
					t.Error("Should still include [DONE] marker for invalid JSON")
				}
			},
		},
		{
			name:     "Empty choices",
			response: `{"choices": []}`,
			model:    "test-model",
			cfg:      DefaultFakeStreamConfig(),
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "[DONE]") {
					t.Error("Should include [DONE] marker for empty choices")
				}
			},
		},
		{
			name: "Without role",
			response: `{
				"choices": [{
					"message": {"content": "Test"},
					"finish_reason": "stop"
				}]
			}`,
			model: "test-model",
			cfg: FakeStreamConfig{
				ChunkSize:   10,
				ChunkDelay:  0,
				IncludeRole: false,
				IncludeDone: true,
			},
			validate: func(t *testing.T, output string) {
				if strings.Contains(output, `"role"`) {
					t.Error("Should not include role when IncludeRole is false")
				}
			},
		},
		{
			name: "Without DONE marker",
			response: `{
				"choices": [{
					"message": {"content": "Test"},
					"finish_reason": "stop"
				}]
			}`,
			model: "test-model",
			cfg: FakeStreamConfig{
				ChunkSize:   10,
				ChunkDelay:  0,
				IncludeRole: true,
				IncludeDone: false,
			},
			validate: func(t *testing.T, output string) {
				if strings.Contains(output, "[DONE]") {
					t.Error("Should not include [DONE] when IncludeDone is false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			reader := ConvertToFakeStream(ctx, []byte(tt.response), tt.model, tt.cfg)

			output, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("Failed to read stream: %v", err)
			}

			tt.validate(t, string(output))
		})
	}
}

func TestConvertToFakeStreamCancellation(t *testing.T) {
	response := `{
		"choices": [{
			"message": {"content": "This is a very long message that will be split into many chunks"},
			"finish_reason": "stop"
		}]
	}`

	ctx, cancel := context.WithCancel(context.Background())
	cfg := FakeStreamConfig{
		ChunkSize:   5,
		ChunkDelay:  100 * time.Millisecond,
		IncludeRole: true,
		IncludeDone: true,
	}

	reader := ConvertToFakeStream(ctx, []byte(response), "test-model", cfg)

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read stream: %v", err)
	}

	// Should have partial output
	if len(output) == 0 {
		t.Error("Expected some output before cancellation")
	}
}

func TestExtractTextFromStream(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name: "Valid stream",
			input: `data: {"choices":[{"delta":{"content":"Hello"}}]}

data: {"choices":[{"delta":{"content":", world!"}}]}

data: [DONE]

`,
			expected: "Hello, world!",
			wantErr:  false,
		},
		{
			name: "Stream with role",
			input: `data: {"choices":[{"delta":{"role":"assistant"}}]}

data: {"choices":[{"delta":{"content":"Test"}}]}

data: [DONE]

`,
			expected: "Test",
			wantErr:  false,
		},
		{
			name:     "Empty stream",
			input:    `data: [DONE]`,
			expected: "",
			wantErr:  false,
		},
		{
			name: "Stream with invalid JSON",
			input: `data: {invalid}

data: {"choices":[{"delta":{"content":"Valid"}}]}

data: [DONE]

`,
			expected: "Valid",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := ExtractTextFromStream(reader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTextFromStream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != tt.expected {
				t.Errorf("ExtractTextFromStream() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsCompleteStream(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Stream with DONE marker",
			input:    "data: {}\ndata: [DONE]",
			expected: true,
		},
		{
			name:     "Stream with finish_reason",
			input:    `data: {"choices":[{"finish_reason":"stop"}]}`,
			expected: true,
		},
		{
			name:     "Incomplete stream",
			input:    "data: {}",
			expected: false,
		},
		{
			name:     "Empty stream",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result := IsCompleteStream(reader)

			if result != tt.expected {
				t.Errorf("IsCompleteStream() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertToFakeStreamChunking(t *testing.T) {
	response := `{
		"choices": [{
			"message": {"content": "1234567890"},
			"finish_reason": "stop"
		}]
	}`

	ctx := context.Background()
	cfg := FakeStreamConfig{
		ChunkSize:   3,
		ChunkDelay:  0,
		IncludeRole: true,
		IncludeDone: true,
	}

	reader := ConvertToFakeStream(ctx, []byte(response), "test-model", cfg)
	output, _ := io.ReadAll(reader)

	// Parse all chunks
	lines := strings.Split(string(output), "\n")
	chunkCount := 0
	var allContent strings.Builder

	for _, line := range lines {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data: ")
		if jsonData == "[DONE]" {
			break
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
			continue
		}

		choices := chunk["choices"].([]interface{})
		delta := choices[0].(map[string]interface{})["delta"].(map[string]interface{})

		if content, ok := delta["content"].(string); ok {
			allContent.WriteString(content)
			chunkCount++
		}
	}

	// Should have 4 chunks: "123", "456", "789", "0"
	if chunkCount != 4 {
		t.Errorf("Expected 4 chunks, got %d", chunkCount)
	}

	if allContent.String() != "1234567890" {
		t.Errorf("Reconstructed content = %q, want %q", allContent.String(), "1234567890")
	}
}
