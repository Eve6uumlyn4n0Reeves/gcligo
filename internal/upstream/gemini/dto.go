package gemini

// DTO (Data Transfer Objects) for Gemini Code Assist API
// This file defines the contract between our client and the upstream API

// CodeAssistRequest represents the request payload for Code Assist API
type CodeAssistRequest struct {
	Model   string                 `json:"model"`
	Project string                 `json:"project"`
	Request CodeAssistRequestInner `json:"request"`
}

// CodeAssistRequestInner represents the inner request structure
type CodeAssistRequestInner struct {
	Contents         []Content          `json:"contents,omitempty"`
	GenerationConfig *GenerationConfig  `json:"generationConfig,omitempty"`
	SystemInstruction *SystemInstruction `json:"systemInstruction,omitempty"`
	SafetySettings   []SafetySetting    `json:"safetySettings,omitempty"`
	Tools            []Tool             `json:"tools,omitempty"`
	ToolConfig       *ToolConfig        `json:"toolConfig,omitempty"`
}

// Content represents a content item in the request
type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts,omitempty"`
}

// Part represents a part of content (text, inline data, etc.)
type Part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *InlineData `json:"inlineData,omitempty"`
	FileData   *FileData   `json:"fileData,omitempty"`
}

// InlineData represents inline binary data
type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

// FileData represents file reference
type FileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

// GenerationConfig represents generation configuration
type GenerationConfig struct {
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"topP,omitempty"`
	TopK             *int            `json:"topK,omitempty"`
	MaxOutputTokens  *int            `json:"maxOutputTokens,omitempty"`
	StopSequences    []string        `json:"stopSequences,omitempty"`
	ResponseMimeType string          `json:"responseMimeType,omitempty"`
	ResponseSchema   interface{}     `json:"responseSchema,omitempty"`
	ThinkingConfig   *ThinkingConfig `json:"thinkingConfig,omitempty"`
}

// ThinkingConfig represents thinking configuration
type ThinkingConfig struct {
	ThinkingBudget *int `json:"thinkingBudget,omitempty"`
}

// SystemInstruction represents system instruction
type SystemInstruction struct {
	Parts []Part `json:"parts,omitempty"`
}

// SafetySetting represents a safety setting
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// Tool represents a tool definition
type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations,omitempty"`
	CodeExecution        *CodeExecution        `json:"codeExecution,omitempty"`
}

// FunctionDeclaration represents a function declaration
type FunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// CodeExecution represents code execution tool
type CodeExecution struct{}

// ToolConfig represents tool configuration
type ToolConfig struct {
	FunctionCallingConfig *FunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

// FunctionCallingConfig represents function calling configuration
type FunctionCallingConfig struct {
	Mode          string   `json:"mode,omitempty"`
	AllowedFunctions []string `json:"allowedFunctionNames,omitempty"`
}

// CodeAssistResponse represents the response from Code Assist API
type CodeAssistResponse struct {
	Response         *ResponseInner    `json:"response,omitempty"`
	Candidates       []Candidate       `json:"candidates,omitempty"`
	PromptFeedback   *PromptFeedback   `json:"promptFeedback,omitempty"`
	UsageMetadata    *UsageMetadata    `json:"usageMetadata,omitempty"`
	ModelVersion     string            `json:"modelVersion,omitempty"`
}

// ResponseInner represents the inner response structure
type ResponseInner struct {
	Candidates     []Candidate     `json:"candidates,omitempty"`
	PromptFeedback *PromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *UsageMetadata  `json:"usageMetadata,omitempty"`
}

// Candidate represents a candidate response
type Candidate struct {
	Content       *Content      `json:"content,omitempty"`
	FinishReason  string        `json:"finishReason,omitempty"`
	Index         int           `json:"index,omitempty"`
	SafetyRatings []SafetyRating `json:"safetyRatings,omitempty"`
	CitationMetadata *CitationMetadata `json:"citationMetadata,omitempty"`
	TokenCount    int           `json:"tokenCount,omitempty"`
	GroundingAttributions []GroundingAttribution `json:"groundingAttributions,omitempty"`
	AvgLogprobs   *float64      `json:"avgLogprobs,omitempty"`
}

// SafetyRating represents a safety rating
type SafetyRating struct {
	Category    string  `json:"category"`
	Probability string  `json:"probability"`
	Blocked     bool    `json:"blocked,omitempty"`
}

// CitationMetadata represents citation metadata
type CitationMetadata struct {
	CitationSources []CitationSource `json:"citationSources,omitempty"`
}

// CitationSource represents a citation source
type CitationSource struct {
	StartIndex int    `json:"startIndex,omitempty"`
	EndIndex   int    `json:"endIndex,omitempty"`
	URI        string `json:"uri,omitempty"`
	License    string `json:"license,omitempty"`
}

// GroundingAttribution represents grounding attribution
type GroundingAttribution struct {
	SourceID      *SourceID      `json:"sourceId,omitempty"`
	Content       *Content       `json:"content,omitempty"`
}

// SourceID represents a source identifier
type SourceID struct {
	GroundingPassageID string `json:"groundingPassageId,omitempty"`
	SemanticRetrieverChunk *SemanticRetrieverChunk `json:"semanticRetrieverChunk,omitempty"`
}

// SemanticRetrieverChunk represents semantic retriever chunk
type SemanticRetrieverChunk struct {
	Source string `json:"source,omitempty"`
	Chunk  string `json:"chunk,omitempty"`
}

// PromptFeedback represents prompt feedback
type PromptFeedback struct {
	BlockReason   string         `json:"blockReason,omitempty"`
	SafetyRatings []SafetyRating `json:"safetyRatings,omitempty"`
}

// UsageMetadata represents usage metadata
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
}

// CountTokensRequest represents the request for counting tokens
type CountTokensRequest struct {
	Model    string                 `json:"model"`
	Project  string                 `json:"project"`
	Request  CountTokensRequestInner `json:"request"`
}

// CountTokensRequestInner represents the inner request for counting tokens
type CountTokensRequestInner struct {
	Contents []Content `json:"contents,omitempty"`
}

// CountTokensResponse represents the response for counting tokens
type CountTokensResponse struct {
	TotalTokens int `json:"totalTokens,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error *ErrorDetail `json:"error,omitempty"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
	Details []ErrorDetailItem `json:"details,omitempty"`
}

// ErrorDetailItem represents an error detail item
type ErrorDetailItem struct {
	Type     string `json:"@type,omitempty"`
	Reason   string `json:"reason,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

