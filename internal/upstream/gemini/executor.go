package gemini

import (
	"context"
	"encoding/json"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/oauth"
)

// Executor provides a small abstraction around Client to centralize
// gemini-cli specific transforms (headers are still done by Client).
type Executor struct {
	cfg  *config.Config
	base *Client
}

func NewExecutor(cfg *config.Config) *Executor {
	return &Executor{cfg: cfg, base: New(cfg)}
}

func (e *Executor) clientFor(cred *oauth.Credentials) *Client {
	if cred == nil {
		return e.base
	}
	return NewWithCredential(e.cfg, cred)
}

// preparePayload applies lightweight, CLI-aligned request fixes.
func (e *Executor) preparePayload(model string, raw []byte) []byte {
	// image hints and thinking safety (mirror client safeguards)
	out := fixGeminiCLIImageHints(model, raw)
	// If model disallows thinking, strip thinkingConfig at top-level too (defense in depth)
	if geminiModelDisallowsThinking(model) {
		var m map[string]any
		if json.Unmarshal(out, &m) == nil {
			if gc, ok := m["generationConfig"].(map[string]any); ok {
				delete(gc, "thinkingConfig")
				m["generationConfig"] = gc
				if b, err := json.Marshal(m); err == nil {
					out = b
				}
			}
		}
	}
	return out
}

func (e *Executor) Execute(ctx context.Context, cred *oauth.Credentials, model string, payload []byte) (*Response, error) {
	cli := e.clientFor(cred)
	p := e.preparePayload(model, payload)
	httpResp, err := cli.Generate(ctx, p)
	return &Response{HTTP: httpResp}, err
}

func (e *Executor) ExecuteStream(ctx context.Context, cred *oauth.Credentials, model string, payload []byte) (*Response, error) {
	cli := e.clientFor(cred)
	p := e.preparePayload(model, payload)
	httpResp, err := cli.Stream(ctx, p)
	return &Response{HTTP: httpResp}, err
}

func (e *Executor) CountTokens(ctx context.Context, cred *oauth.Credentials, model string, payload []byte) (*Response, error) {
	cli := e.clientFor(cred)
	p := e.preparePayload(model, payload)
	httpResp, err := cli.CountTokens(ctx, p)
	return &Response{HTTP: httpResp}, err
}

// Response is a thin wrapper to avoid leaking http in signatures everywhere.
type Response struct{ HTTP any }
