package gemini

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeAssistRequestContract verifies the request DTO structure
func TestCodeAssistRequestContract(t *testing.T) {
	t.Run("basic request serialization", func(t *testing.T) {
		req := CodeAssistRequest{
			Model:   "gemini-2.5-pro",
			Project: "test-project",
			Request: CodeAssistRequestInner{
				Contents: []Content{
					{
						Role: "user",
						Parts: []Part{
							{Text: "Hello, world!"},
						},
					},
				},
			},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded CodeAssistRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "gemini-2.5-pro", decoded.Model)
		assert.Equal(t, "test-project", decoded.Project)
		assert.Len(t, decoded.Request.Contents, 1)
		assert.Equal(t, "user", decoded.Request.Contents[0].Role)
		assert.Equal(t, "Hello, world!", decoded.Request.Contents[0].Parts[0].Text)
	})

	t.Run("request with generation config", func(t *testing.T) {
		temp := 0.7
		maxTokens := 1024
		req := CodeAssistRequest{
			Model:   "gemini-2.5-flash",
			Project: "test-project",
			Request: CodeAssistRequestInner{
				Contents: []Content{
					{Role: "user", Parts: []Part{{Text: "Test"}}},
				},
				GenerationConfig: &GenerationConfig{
					Temperature:     &temp,
					MaxOutputTokens: &maxTokens,
				},
			},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded CodeAssistRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.NotNil(t, decoded.Request.GenerationConfig)
		assert.Equal(t, 0.7, *decoded.Request.GenerationConfig.Temperature)
		assert.Equal(t, 1024, *decoded.Request.GenerationConfig.MaxOutputTokens)
	})

	t.Run("request with thinking config", func(t *testing.T) {
		budget := 5000
		req := CodeAssistRequest{
			Model:   "gemini-2.5-pro",
			Project: "test-project",
			Request: CodeAssistRequestInner{
				Contents: []Content{
					{Role: "user", Parts: []Part{{Text: "Complex problem"}}},
				},
				GenerationConfig: &GenerationConfig{
					ThinkingConfig: &ThinkingConfig{
						ThinkingBudget: &budget,
					},
				},
			},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded CodeAssistRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.NotNil(t, decoded.Request.GenerationConfig.ThinkingConfig)
		assert.Equal(t, 5000, *decoded.Request.GenerationConfig.ThinkingConfig.ThinkingBudget)
	})
}

// TestCodeAssistResponseContract verifies the response DTO structure
func TestCodeAssistResponseContract(t *testing.T) {
	t.Run("basic response deserialization", func(t *testing.T) {
		responseJSON := `{
			"response": {
				"candidates": [
					{
						"content": {
							"parts": [
								{"text": "Hello! How can I help you?"}
							],
							"role": "model"
						},
						"finishReason": "STOP",
						"index": 0
					}
				],
				"usageMetadata": {
					"promptTokenCount": 5,
					"candidatesTokenCount": 10,
					"totalTokenCount": 15
				}
			}
		}`

		var resp CodeAssistResponse
		err := json.Unmarshal([]byte(responseJSON), &resp)
		require.NoError(t, err)

		assert.NotNil(t, resp.Response)
		assert.Len(t, resp.Response.Candidates, 1)
		assert.Equal(t, "STOP", resp.Response.Candidates[0].FinishReason)
		assert.Equal(t, "Hello! How can I help you?", resp.Response.Candidates[0].Content.Parts[0].Text)
		assert.NotNil(t, resp.Response.UsageMetadata)
		assert.Equal(t, 5, resp.Response.UsageMetadata.PromptTokenCount)
		assert.Equal(t, 10, resp.Response.UsageMetadata.CandidatesTokenCount)
		assert.Equal(t, 15, resp.Response.UsageMetadata.TotalTokenCount)
	})

	t.Run("response with safety ratings", func(t *testing.T) {
		responseJSON := `{
			"candidates": [
				{
					"content": {
						"parts": [{"text": "Safe content"}],
						"role": "model"
					},
					"safetyRatings": [
						{
							"category": "HARM_CATEGORY_HATE_SPEECH",
							"probability": "NEGLIGIBLE"
						}
					],
					"finishReason": "STOP"
				}
			]
		}`

		var resp CodeAssistResponse
		err := json.Unmarshal([]byte(responseJSON), &resp)
		require.NoError(t, err)

		assert.Len(t, resp.Candidates, 1)
		assert.Len(t, resp.Candidates[0].SafetyRatings, 1)
		assert.Equal(t, "HARM_CATEGORY_HATE_SPEECH", resp.Candidates[0].SafetyRatings[0].Category)
		assert.Equal(t, "NEGLIGIBLE", resp.Candidates[0].SafetyRatings[0].Probability)
	})

	t.Run("error response deserialization", func(t *testing.T) {
		errorJSON := `{
			"error": {
				"code": 404,
				"message": "Model not found",
				"status": "NOT_FOUND"
			}
		}`

		var errResp ErrorResponse
		err := json.Unmarshal([]byte(errorJSON), &errResp)
		require.NoError(t, err)

		assert.NotNil(t, errResp.Error)
		assert.Equal(t, 404, errResp.Error.Code)
		assert.Equal(t, "Model not found", errResp.Error.Message)
		assert.Equal(t, "NOT_FOUND", errResp.Error.Status)
	})
}

// TestCountTokensContract verifies the count tokens DTO structure
func TestCountTokensContract(t *testing.T) {
	t.Run("count tokens request", func(t *testing.T) {
		req := CountTokensRequest{
			Model:   "gemini-2.5-pro",
			Project: "test-project",
			Request: CountTokensRequestInner{
				Contents: []Content{
					{Role: "user", Parts: []Part{{Text: "Count these tokens"}}},
				},
			},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded CountTokensRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "gemini-2.5-pro", decoded.Model)
		assert.Len(t, decoded.Request.Contents, 1)
	})

	t.Run("count tokens response", func(t *testing.T) {
		responseJSON := `{"totalTokens": 42}`

		var resp CountTokensResponse
		err := json.Unmarshal([]byte(responseJSON), &resp)
		require.NoError(t, err)

		assert.Equal(t, 42, resp.TotalTokens)
	})
}

// TestPathConstants verifies path construction
func TestPathConstants(t *testing.T) {
	t.Run("standard paths", func(t *testing.T) {
		assert.Equal(t, "/v1internal:generate", PathGenerate)
		assert.Equal(t, "/v1internal:streamGenerate", PathStreamGenerate)
		assert.Equal(t, "/v1internal:countTokens", PathCountTokens)
		assert.Equal(t, "/v1internal/models", PathListModels)
	})

	t.Run("build action path", func(t *testing.T) {
		assert.Equal(t, "/v1internal:countTokens", BuildActionPath("countTokens"))
		assert.Equal(t, "/v1internal:customAction", BuildActionPath("customAction"))
		assert.Equal(t, PathGenerate, BuildActionPath(""))
	})

	t.Run("build model path", func(t *testing.T) {
		assert.Equal(t, "/v1internal/models/gemini-2.5-pro", BuildModelPath("gemini-2.5-pro"))
		assert.Equal(t, PathListModels, BuildModelPath(""))
	})
}

// TestDTOFieldPresence ensures critical fields are present
func TestDTOFieldPresence(t *testing.T) {
	t.Run("request has required fields", func(t *testing.T) {
		req := CodeAssistRequest{}
		data, _ := json.Marshal(req)
		
		var m map[string]interface{}
		json.Unmarshal(data, &m)
		
		// Empty request should still have the structure
		assert.Contains(t, m, "model")
		assert.Contains(t, m, "project")
		assert.Contains(t, m, "request")
	})

	t.Run("response handles optional fields", func(t *testing.T) {
		// Minimal response
		minimalJSON := `{}`
		var resp CodeAssistResponse
		err := json.Unmarshal([]byte(minimalJSON), &resp)
		require.NoError(t, err)
		
		// Should not panic with nil fields
		assert.Nil(t, resp.Response)
		assert.Nil(t, resp.Candidates)
	})
}

// 兼容性：未知字段与缺失字段
func TestUnknownAndMissingFieldsCompatibility(t *testing.T) {
    t.Run("unknown fields are ignored", func(t *testing.T) {
        raw := `{
            "model": "gemini-2.5-pro",
            "project": "p",
            "request": {
                "contents": [{"role":"user","parts":[{"text":"hello"}]}],
                "generationConfig": {"temperature": 0.2, "unknown_nested": 1}
            },
            "unknown_top_level": true
        }`
        var req CodeAssistRequest
        err := json.Unmarshal([]byte(raw), &req)
        require.NoError(t, err)
        require.NotNil(t, req.Request.GenerationConfig)
        assert.Equal(t, "gemini-2.5-pro", req.Model)
    })

    t.Run("missing optional fields result in zero values", func(t *testing.T) {
        raw := `{"model":"m","project":"p","request":{}}`
        var req CodeAssistRequest
        err := json.Unmarshal([]byte(raw), &req)
        require.NoError(t, err)
        assert.Equal(t, "m", req.Model)
        assert.Equal(t, "p", req.Project)
        assert.Len(t, req.Request.Contents, 0)
        assert.Nil(t, req.Request.GenerationConfig)
    })
}
