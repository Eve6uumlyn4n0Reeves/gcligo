package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	apperrors "gcli2api-go/internal/errors"
	"gcli2api-go/internal/models"
	"github.com/gin-gonic/gin"
)

// ParsedRequest represents a parsed and validated OpenAI-style request
type ParsedRequest struct {
	Raw       map[string]any
	Model     string
	BaseModel string
	Stream    bool
	RawJSON   []byte
}

// ValidationError represents a request validation error
type ValidationError struct {
	Status  int
	Type    string
	Message string
	apiErr  *apperrors.APIError
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "validation error"
	}
	return e.Message
}

func (e *ValidationError) APIError() *apperrors.APIError {
	if e == nil {
		return nil
	}
	if e.apiErr != nil {
		return e.apiErr
	}
	status := e.Status
	if status == 0 {
		status = http.StatusBadRequest
	}
	typ := strings.TrimSpace(e.Type)
	if typ == "" {
		typ = "invalid_request_error"
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = "invalid request"
	}
	e.apiErr = apperrors.New(status, typ, typ, msg)
	return e.apiErr
}

// ParseOpenAIRequest parses and validates an OpenAI-style request
// It automatically sets model and base_model in the gin context
func ParseOpenAIRequest(c *gin.Context, defaultModel string) (*ParsedRequest, error) {
	var raw map[string]any
	if err := c.ShouldBindJSON(&raw); err != nil {
		return nil, newValidationError("invalid json: " + err.Error())
	}

	// Extract model
	model, _ := raw["model"].(string)
	if model == "" {
		model = defaultModel
	}

	// Extract stream flag
	stream, _ := raw["stream"].(bool)

	// Get base model
	baseModel := models.BaseFromFeature(model)

	// Set context values
	c.Set("model", model)
	c.Set("base_model", baseModel)

	// Marshal back to JSON for downstream processing
	rawJSON, _ := json.Marshal(raw)

	return &ParsedRequest{
		Raw:       raw,
		Model:     model,
		BaseModel: baseModel,
		Stream:    stream,
		RawJSON:   rawJSON,
	}, nil
}

// ParseOpenAIChatRequest parses and validates an OpenAI chat completion request
// with additional chat-specific validation
func ParseOpenAIChatRequest(c *gin.Context, defaultModel string) (*ParsedRequest, error) {
	req, err := ParseOpenAIRequest(c, defaultModel)
	if err != nil {
		return nil, err
	}

	// Validate messages field exists
	messages, ok := req.Raw["messages"]
	if !ok {
		return nil, newValidationError("missing required field: messages")
	}

	// Validate messages is an array
	if _, ok := messages.([]any); !ok {
		return nil, newValidationError("messages must be an array")
	}

	return req, nil
}

// ParseGeminiRequest parses a Gemini-style request
func ParseGeminiRequest(c *gin.Context, defaultModel string) (*ParsedRequest, error) {
	var raw map[string]any
	if err := c.ShouldBindJSON(&raw); err != nil {
		return nil, newValidationError("invalid json: " + err.Error())
	}

	// For Gemini requests, model might be in different location
	model, _ := raw["model"].(string)
	if model == "" {
		model = defaultModel
	}

	baseModel := models.BaseFromFeature(model)

	c.Set("model", model)
	c.Set("base_model", baseModel)

	rawJSON, _ := json.Marshal(raw)

	return &ParsedRequest{
		Raw:       raw,
		Model:     model,
		BaseModel: baseModel,
		Stream:    false, // Gemini requests don't have stream flag in same way
		RawJSON:   rawJSON,
	}, nil
}

// AbortWithValidationError is a helper to abort with a validation error
func AbortWithValidationError(c *gin.Context, err error) {
	if ve, ok := err.(*ValidationError); ok {
		AbortWithAPIError(c, ve.APIError())
	} else {
		AbortWithError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
	}
}

// ValidateAndNormalizeMessages validates and normalizes the messages array
func ValidateAndNormalizeMessages(messages []any) error {
	if len(messages) == 0 {
		return newValidationError("messages array cannot be empty")
	}

	for i, msg := range messages {
		msgMap, ok := msg.(map[string]any)
		if !ok {
			return newValidationError(fmt.Sprintf("message at index %d is not an object", i))
		}

		role, ok := msgMap["role"].(string)
		if !ok || role == "" {
			return newValidationError(fmt.Sprintf("message at index %d missing required field: role", i))
		}

		// Validate role is one of the allowed values
		switch role {
		case "system", "user", "assistant", "tool", "function":
			// Valid roles
		default:
			return newValidationError(fmt.Sprintf("message at index %d has invalid role: %s", i, role))
		}

		// Check for content field (required for most roles except tool/function)
		if role != "tool" && role != "function" {
			if _, hasContent := msgMap["content"]; !hasContent {
				if _, hasToolCalls := msgMap["tool_calls"]; !hasToolCalls {
					return newValidationError(fmt.Sprintf("message at index %d missing content or tool_calls", i))
				}
			}
		}
	}

	return nil
}

func newValidationError(message string) *ValidationError {
	return &ValidationError{
		Status:  http.StatusBadRequest,
		Type:    "invalid_request_error",
		Message: message,
	}
}
