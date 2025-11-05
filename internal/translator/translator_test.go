package translator

import (
	"context"
	"encoding/json"
	"testing"

	"gcli2api-go/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIToGeminiRequest(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKeys []string
	}{
		{
			name: "simple chat request",
			input: `{
				"model": "gemini-2.5-pro",
				"messages": [
					{"role": "user", "content": "Hello"}
				]
			}`,
			wantKeys: []string{"contents", "generationConfig"},
		},
		{
			name: "request with thinking mode",
			input: `{
				"model": "gemini-2.5-pro",
				"messages": [
					{"role": "user", "content": "Solve this problem"}
				],
				"reasoning_effort": "high"
			}`,
			wantKeys: []string{"contents", "generationConfig"},
		},
		{
			name: "request with tools",
			input: `{
				"model": "gemini-2.5-pro",
				"messages": [
					{"role": "user", "content": "Call a function"}
				],
				"tools": [
					{
						"type": "function",
						"function": {
							"name": "get_weather",
							"description": "Get weather info",
							"parameters": {
								"type": "object",
								"properties": {
									"location": {"type": "string"}
								}
							}
						}
					}
				]
			}`,
			wantKeys: []string{"contents", "generationConfig", "tools"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := OpenAIToGeminiRequest("gemini-2.5-pro", []byte(tt.input), false)

			var parsed map[string]interface{}
			err := json.Unmarshal(result, &parsed)
			require.NoError(t, err)

			for _, key := range tt.wantKeys {
				assert.Contains(t, parsed, key, "Expected key %s in result", key)
			}
		})
	}
}

func TestGeminiToOpenAIResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "simple response",
			input: `{
				"candidates": [
					{
						"content": {
							"parts": [
								{"text": "Hello! How can I help you?"}
							],
							"role": "model"
						},
						"finishReason": "STOP"
					}
				],
				"usageMetadata": {
					"promptTokenCount": 10,
					"candidatesTokenCount": 20
				}
			}`,
			wantErr: false,
		},
		{
			name: "response with tool calls",
			input: `{
				"candidates": [
					{
						"content": {
							"parts": [
								{
									"functionCall": {
										"name": "get_weather",
										"args": {"location": "Tokyo"}
									}
								}
							],
							"role": "model"
						},
						"finishReason": "STOP"
					}
				]
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GeminiToOpenAIResponse(context.Background(), "gemini-2.5-pro", []byte(tt.input))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				var parsed map[string]interface{}
				err := json.Unmarshal(result, &parsed)
				require.NoError(t, err)

				assert.Contains(t, parsed, "choices")
				assert.Contains(t, parsed, "model")
			}
		})
	}
}

func TestThinkingConfigConversion(t *testing.T) {
	tests := []struct {
		name            string
		reasoningEffort string
		expectBudget    int
	}{
		{"none", "none", 0},
		{"auto", "auto", -1},
		{"low", "low", 1024},
		{"medium", "medium", 8192},
		{"high", "high", 24576},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]interface{}{
				"model": "gemini-2.5-pro",
				"messages": []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "test",
					},
				},
				"reasoning_effort": tt.reasoningEffort,
			}

			inputJSON, _ := json.Marshal(input)
			result := OpenAIToGeminiRequest("gemini-2.5-pro", inputJSON, false)

			var parsed map[string]interface{}
			json.Unmarshal(result, &parsed)

			genConfig, ok := parsed["generationConfig"].(map[string]interface{})
			require.True(t, ok, "generationConfig should exist")

			if tt.expectBudget != 0 {
				thinkingConfig, ok := genConfig["thinkingConfig"].(map[string]interface{})
				require.True(t, ok, "thinkingConfig should exist")

				budget := int(thinkingConfig["thinkingBudget"].(float64))
				assert.Equal(t, tt.expectBudget, budget)
			}
		})
	}
}

func TestMergeConsecutiveMessages(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"role":  "user",
			"parts": []interface{}{map[string]interface{}{"text": "Part 1"}},
		},
		map[string]interface{}{
			"role":  "user",
			"parts": []interface{}{map[string]interface{}{"text": "Part 2"}},
		},
		map[string]interface{}{
			"role":  "model",
			"parts": []interface{}{map[string]interface{}{"text": "Response"}},
		},
	}

	result := mergeConsecutiveMessages(input)

	// Should merge the two user messages
	assert.Equal(t, 2, len(result))

	firstMsg := result[0].(map[string]interface{})
	assert.Equal(t, "user", firstMsg["role"])

	parts := firstMsg["parts"].([]interface{})
	assert.Equal(t, 2, len(parts), "Should have merged 2 parts")
}

func TestDetectThinkingInText(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"<think>Let me think</think>", true},
		{"[THINKING] Analyzing the problem", true},
		{"Let me think about this", true},
		{"This is a normal response", false},
		{"Thinking: First, we need to...", true},
		{"Just a regular answer", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := detectThinkingInText(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkOpenAIToGeminiRequest(b *testing.B) {
	input := []byte(`{
		"model": "gemini-2.5-pro",
		"messages": [
			{"role": "user", "content": "Hello, how are you?"}
		],
		"temperature": 0.7,
		"max_tokens": 100
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		OpenAIToGeminiRequest("gemini-2.5-pro", input, false)
	}
}

func TestOpenAIResponsesToGeminiRequest(t *testing.T) {
	input := `{
        "model": "gemini-2.5-pro",
        "instructions": "Follow system",
        "input": [
            {"type":"input_text","text":"describe the image"},
            {"type":"image_url","image_url":{"url":"data:image/png;base64,AAAA"}}
        ],
        "tools": [{"type":"function","function":{"name":"f","description":"d","parameters":{"type":"object"}}}],
        "temperature": 0.2,
        "max_output_tokens": 256
    }`
	out := OpenAIResponsesToGeminiRequest("gemini-2.5-pro", []byte(input), false)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(out, &obj))
	assert.NotNil(t, obj["contents"])
	assert.NotNil(t, obj["generationConfig"])
	gc := obj["generationConfig"].(map[string]any)
	assert.Equal(t, float64(constants.DefaultTopK), gc["topK"])
}

func TestOpenAICompletionsToGeminiRequest(t *testing.T) {
	input := `{"prompt":"Hello world","temperature":0.5,"max_tokens":64}`
	out := OpenAICompletionsToGeminiRequest("gemini-2.5-pro", []byte(input), false)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(out, &obj))
	assert.NotNil(t, obj["contents"])
	assert.NotNil(t, obj["generationConfig"])
	gc := obj["generationConfig"].(map[string]any)
	assert.Equal(t, float64(constants.DefaultTopK), gc["topK"])
}

func TestOpenAIToGeminiRequest_AdditionalParams(t *testing.T) {
	input := map[string]any{
		"model":             "gemini-2.5-pro",
		"messages":          []any{map[string]any{"role": "user", "content": "hi"}},
		"stop":              []any{"END", "STOP"},
		"frequency_penalty": 0.25,
		"presence_penalty":  0.5,
		"n":                 2,
		"seed":              42,
	}
	b, _ := json.Marshal(input)
	out := OpenAIToGeminiRequest("gemini-2.5-pro", b, false)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(out, &obj))
	gc, ok := obj["generationConfig"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(0.25), gc["frequencyPenalty"])
	assert.Equal(t, float64(0.5), gc["presencePenalty"])
	assert.Equal(t, float64(2), gc["candidateCount"])
	assert.Equal(t, float64(42), gc["seed"])
	assert.Equal(t, float64(constants.DefaultTopK), gc["topK"])
	// stop sequences
	ss, _ := gc["stopSequences"].([]any)
	require.Len(t, ss, 2)
}

func TestOpenAIResponsesToGeminiRequest_AdditionalParams(t *testing.T) {
	input := map[string]any{
		"input":             "hi",
		"stop":              "END",
		"frequency_penalty": 0.1,
		"presence_penalty":  0.2,
		"n":                 3,
		"seed":              7,
	}
	b, _ := json.Marshal(input)
	out := OpenAIResponsesToGeminiRequest("gemini-2.5-pro", b, false)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(out, &obj))
	gc, ok := obj["generationConfig"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(0.1), gc["frequencyPenalty"])
	assert.Equal(t, float64(0.2), gc["presencePenalty"])
	assert.Equal(t, float64(3), gc["candidateCount"])
	assert.Equal(t, float64(7), gc["seed"])
	assert.Equal(t, float64(constants.DefaultTopK), gc["topK"])
	// stop as single string becomes array of one
	ss, _ := gc["stopSequences"].([]any)
	require.Len(t, ss, 1)
}

func TestTopKAndMaxTokensClamped(t *testing.T) {
	input := map[string]any{
		"model":      "gemini-2.5-pro",
		"messages":   []any{map[string]any{"role": "user", "content": "hi"}},
		"top_k":      128,
		"max_tokens": 999999,
	}
	payload, _ := json.Marshal(input)
	out := OpenAIToGeminiRequest("gemini-2.5-pro", payload, false)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(out, &obj))
	gc := obj["generationConfig"].(map[string]any)
	assert.Equal(t, float64(constants.MaxTopK), gc["topK"])
	assert.Equal(t, float64(constants.MaxOutputTokens), gc["maxOutputTokens"])

	respInput := map[string]any{
		"input":             "hello",
		"top_k":             -5,
		"max_output_tokens": 888888,
	}
	respPayload, _ := json.Marshal(respInput)
	respOut := OpenAIResponsesToGeminiRequest("gemini-2.5-pro", respPayload, false)
	var respObj map[string]any
	require.NoError(t, json.Unmarshal(respOut, &respObj))
	respGc := respObj["generationConfig"].(map[string]any)
	assert.Equal(t, float64(constants.DefaultTopK), respGc["topK"])
	assert.Equal(t, float64(constants.MaxOutputTokens), respGc["maxOutputTokens"])
}
