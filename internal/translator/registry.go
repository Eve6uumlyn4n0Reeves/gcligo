package translator

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// Registry manages translation functions between different API formats.
type Registry struct {
	mu        sync.RWMutex
	requests  map[Format]map[Format]RequestTransform
	responses map[Format]map[Format]ResponseTransform
	streams   map[Format]map[Format]StreamTransform
}

// NewRegistry constructs an empty translator registry.
func NewRegistry() *Registry {
	return &Registry{
		requests:  make(map[Format]map[Format]RequestTransform),
		responses: make(map[Format]map[Format]ResponseTransform),
		streams:   make(map[Format]map[Format]StreamTransform),
	}
}

var defaultRegistry = NewRegistry()

// Default returns the default global registry.
func Default() *Registry {
	return defaultRegistry
}

// Register stores request/response transforms between two formats.
func (r *Registry) Register(from, to Format, cfg TranslatorConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.requests[from]; !ok {
		r.requests[from] = make(map[Format]RequestTransform)
	}
	if cfg.RequestTransform != nil {
		r.requests[from][to] = cfg.RequestTransform
	}

	if _, ok := r.responses[from]; !ok {
		r.responses[from] = make(map[Format]ResponseTransform)
	}
	if cfg.ResponseTransform != nil {
		r.responses[from][to] = cfg.ResponseTransform
	}

	if _, ok := r.streams[from]; !ok {
		r.streams[from] = make(map[Format]StreamTransform)
	}
	if cfg.StreamTransform != nil {
		r.streams[from][to] = cfg.StreamTransform
	}
}

// TranslateRequest converts a request payload between formats.
// Returns the original payload if no translator is registered.
func (r *Registry) TranslateRequest(from, to Format, model string, rawJSON []byte, stream bool) []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if byTarget, ok := r.requests[from]; ok {
		if fn, exists := byTarget[to]; exists && fn != nil {
			return fn(model, rawJSON, stream)
		}
	}
	return rawJSON
}

// TranslateResponse converts a non-streaming response between formats.
func (r *Registry) TranslateResponse(ctx context.Context, from, to Format, model string, responseBody []byte) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if byTarget, ok := r.responses[from]; ok {
		if fn, exists := byTarget[to]; exists && fn != nil {
			return fn(ctx, model, responseBody)
		}
	}
	return responseBody, nil
}

// TranslateStream converts a streaming response between formats.
func (r *Registry) TranslateStream(ctx context.Context, from, to Format, model string, reader io.Reader) (io.Reader, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if byTarget, ok := r.streams[from]; ok {
		if fn, exists := byTarget[to]; exists && fn != nil {
			return fn(ctx, model, reader)
		}
	}
	return reader, nil
}

// HasResponseTransformer checks if a response translator exists.
func (r *Registry) HasResponseTransformer(from, to Format) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if byTarget, ok := r.responses[from]; ok {
		if _, exists := byTarget[to]; exists {
			return true
		}
	}
	return false
}

// HasStreamTransformer checks if a stream translator exists.
func (r *Registry) HasStreamTransformer(from, to Format) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if byTarget, ok := r.streams[from]; ok {
		if _, exists := byTarget[to]; exists {
			return true
		}
	}
	return false
}

// Register is a convenience function for registering with the default registry.
func Register(from, to Format, cfg TranslatorConfig) {
	defaultRegistry.Register(from, to, cfg)
}

// TranslateRequest uses the default registry.
func TranslateRequest(from, to Format, model string, rawJSON []byte, stream bool) []byte {
	return defaultRegistry.TranslateRequest(from, to, model, rawJSON, stream)
}

// TranslateResponse uses the default registry.
func TranslateResponse(ctx context.Context, from, to Format, model string, responseBody []byte) ([]byte, error) {
	return defaultRegistry.TranslateResponse(ctx, from, to, model, responseBody)
}

// TranslateStream uses the default registry.
func TranslateStream(ctx context.Context, from, to Format, model string, reader io.Reader) (io.Reader, error) {
	return defaultRegistry.TranslateStream(ctx, from, to, model, reader)
}

// FromString converts a string to Format.
func FromString(s string) Format {
	switch s {
	case "openai":
		return FormatOpenAI
	case "gemini":
		return FormatGemini
	default:
		return FormatGeneric
	}
}

// String returns the string representation of a Format.
func (f Format) String() string {
	return string(f)
}

// ErrNoTranslator is returned when no translator is found.
type ErrNoTranslator struct {
	From Format
	To   Format
}

func (e *ErrNoTranslator) Error() string {
	return fmt.Sprintf("no translator found from %s to %s", e.From, e.To)
}
