package errors

// ErrorFormat represents the target error format.
type ErrorFormat string

const (
	FormatOpenAI ErrorFormat = "openai"
	FormatGemini ErrorFormat = "gemini"
)

// APIError represents a standardized error across upstream providers.
type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
	Type       string
	Details    map[string]interface{}
}

// OpenAIError mirrors OpenAI's error envelope.
type OpenAIError struct {
	Error struct {
		Message string                 `json:"message"`
		Type    string                 `json:"type"`
		Code    string                 `json:"code,omitempty"`
		Param   string                 `json:"param,omitempty"`
		Details map[string]interface{} `json:"details,omitempty"`
	} `json:"error"`
}

// GeminiError mirrors Gemini Code Assist's error structure.
type GeminiError struct {
	Error struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Status  string                 `json:"status"`
		Details map[string]interface{} `json:"details,omitempty"`
	} `json:"error"`
}
