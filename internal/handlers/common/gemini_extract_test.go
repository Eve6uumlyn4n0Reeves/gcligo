package common

import (
	"testing"
)

func TestMapFinishReason(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"STOP", "STOP", "stop"},
		{"MAX_TOKENS", "MAX_TOKENS", "length"},
		{"SAFETY", "SAFETY", "content_filter"},
		{"RECITATION", "RECITATION", "content_filter"},
		{"OTHER", "OTHER", "stop"},
		{"Unknown", "UNKNOWN_REASON", "UNKNOWN_REASON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapFinishReason(tt.input)
			if result != tt.expected {
				t.Errorf("mapFinishReason(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractFromResponse(t *testing.T) {
	tests := []struct {
		name           string
		input          map[string]any
		expectedText   string
		expectedImages int
		expectedFCs    int
		expectedReason string
	}{
		{
			name: "Simple text response",
			input: map[string]any{
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{
								map[string]any{"text": "Hello, world!"},
							},
						},
						"finishReason": "STOP",
					},
				},
			},
			expectedText:   "Hello, world!",
			expectedImages: 0,
			expectedFCs:    0,
			expectedReason: "stop",
		},
		{
			name: "Response with image",
			input: map[string]any{
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{
								map[string]any{
									"inlineData": map[string]any{
										"mimeType": "image/png",
										"data":     "base64data",
									},
								},
							},
						},
					},
				},
			},
			expectedText:   "",
			expectedImages: 1,
			expectedFCs:    0,
			expectedReason: "",
		},
		{
			name: "Response with function call",
			input: map[string]any{
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{
								map[string]any{
									"functionCall": map[string]any{
										"name": "get_weather",
										"args": map[string]any{
											"location": "San Francisco",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedText:   "",
			expectedImages: 0,
			expectedFCs:    1,
			expectedReason: "",
		},
		{
			name: "Wrapped response",
			input: map[string]any{
				"response": map[string]any{
					"candidates": []any{
						map[string]any{
							"content": map[string]any{
								"parts": []any{
									map[string]any{"text": "Wrapped text"},
								},
							},
						},
					},
				},
			},
			expectedText:   "Wrapped text",
			expectedImages: 0,
			expectedFCs:    0,
			expectedReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, _ := ExtractFromResponse(tt.input)

			if parsed.Text != tt.expectedText {
				t.Errorf("Text = %q, want %q", parsed.Text, tt.expectedText)
			}
			if len(parsed.Images) != tt.expectedImages {
				t.Errorf("Images count = %d, want %d", len(parsed.Images), tt.expectedImages)
			}
			if len(parsed.FunctionCalls) != tt.expectedFCs {
				t.Errorf("FunctionCalls count = %d, want %d", len(parsed.FunctionCalls), tt.expectedFCs)
			}
			if parsed.FinishReason != tt.expectedReason {
				t.Errorf("FinishReason = %q, want %q", parsed.FinishReason, tt.expectedReason)
			}
		})
	}
}

func TestStreamDeltaExtractor(t *testing.T) {
	extractor := NewStreamDeltaExtractor("test-model")

	t.Run("Extract text delta", func(t *testing.T) {
		event := &SSEEvent{
			Data: map[string]any{
				"response": map[string]any{
					"candidates": []any{
						map[string]any{
							"content": map[string]any{
								"parts": []any{
									map[string]any{"text": "Hello"},
								},
							},
						},
					},
				},
			},
		}

		chunks := extractor.ExtractDelta(event)
		if len(chunks) != 1 {
			t.Fatalf("Expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].Type != "delta_content" {
			t.Errorf("Expected type 'delta_content', got %q", chunks[0].Type)
		}
	})

	t.Run("Extract image delta", func(t *testing.T) {
		event := &SSEEvent{
			Data: map[string]any{
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{
								map[string]any{
									"inlineData": map[string]any{
										"mimeType": "image/jpeg",
										"data":     "base64imagedata",
									},
								},
							},
						},
					},
				},
			},
		}

		chunks := extractor.ExtractDelta(event)
		if len(chunks) != 1 {
			t.Fatalf("Expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].Type != "delta_image" {
			t.Errorf("Expected type 'delta_image', got %q", chunks[0].Type)
		}
	})

	t.Run("Extract tool call delta", func(t *testing.T) {
		event := &SSEEvent{
			Data: map[string]any{
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{
								map[string]any{
									"functionCall": map[string]any{
										"name": "search",
										"args": map[string]any{"query": "test"},
									},
								},
							},
						},
					},
				},
			},
		}

		chunks := extractor.ExtractDelta(event)
		if len(chunks) != 1 {
			t.Fatalf("Expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].Type != "tool_call" {
			t.Errorf("Expected type 'tool_call', got %q", chunks[0].Type)
		}
	})

	t.Run("Nil event", func(t *testing.T) {
		chunks := extractor.ExtractDelta(nil)
		if chunks != nil {
			t.Errorf("Expected nil chunks for nil event, got %v", chunks)
		}
	})
}
