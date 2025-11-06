package gemini

// API path constants for Gemini Code Assist
// These constants define the contract with the upstream API endpoints

const (
	// PathGenerate is the endpoint for non-streaming generation
	PathGenerate = "/v1internal:generate"
	
	// PathStreamGenerate is the endpoint for streaming generation
	PathStreamGenerate = "/v1internal:streamGenerate"
	
	// PathCountTokens is the endpoint for counting tokens
	PathCountTokens = "/v1internal:countTokens"
	
	// PathListModels is the endpoint for listing available models
	PathListModels = "/v1internal/models"
	
	// PathGetModel is the endpoint for getting model details
	// Use with model name appended, e.g., /v1internal/models/gemini-2.5-pro
	PathGetModel = "/v1internal/models"
)

// BuildActionPath constructs the path for a custom action
// Example: BuildActionPath("countTokens") -> "/v1internal:countTokens"
func BuildActionPath(action string) string {
	if action == "" {
		return PathGenerate
	}
	return "/v1internal:" + action
}

// BuildModelPath constructs the path for model-specific operations
// Example: BuildModelPath("gemini-2.5-pro") -> "/v1internal/models/gemini-2.5-pro"
func BuildModelPath(modelName string) string {
	if modelName == "" {
		return PathListModels
	}
	return PathListModels + "/" + modelName
}

