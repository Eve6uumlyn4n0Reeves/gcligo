package translator

import (
	"context"
	"io"
)

// Format represents an API format (openai, gemini, etc.)
type Format string

const (
	FormatOpenAI  Format = "openai"
	FormatGemini  Format = "gemini"
	FormatGeneric Format = "generic"
)

// RequestTransform converts a request from one format to another.
// Returns the transformed request body as bytes.
type RequestTransform func(model string, rawJSON []byte, stream bool) []byte

// ResponseTransform converts a non-streaming response from one format to another.
type ResponseTransform func(ctx context.Context, model string, responseBody []byte) ([]byte, error)

// StreamTransform converts streaming response chunks from one format to another.
// It reads from the input reader and returns a new reader with transformed chunks.
type StreamTransform func(ctx context.Context, model string, reader io.Reader) (io.Reader, error)

// TranslatorConfig holds configuration for request/response translation
type TranslatorConfig struct {
	RequestTransform  RequestTransform
	ResponseTransform ResponseTransform
	StreamTransform   StreamTransform
}
