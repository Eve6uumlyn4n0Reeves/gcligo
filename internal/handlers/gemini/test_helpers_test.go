package gemini

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"gcli2api-go/internal/config"
)

type stubUpstream struct {
	generateFunc    func(context.Context, []byte) (*http.Response, error)
	streamFunc      func(context.Context, []byte) (*http.Response, error)
	countTokensFunc func(context.Context, []byte) (*http.Response, error)
	actionFunc      func(context.Context, string, []byte) (*http.Response, error)
}

func (s *stubUpstream) Generate(ctx context.Context, payload []byte) (*http.Response, error) {
	if s.generateFunc != nil {
		return s.generateFunc(ctx, payload)
	}
	return newHTTPResponse(http.StatusOK, []byte(`{}`)), nil
}

func (s *stubUpstream) Stream(ctx context.Context, payload []byte) (*http.Response, error) {
	if s.streamFunc != nil {
		return s.streamFunc(ctx, payload)
	}
	return newHTTPResponse(http.StatusOK, []byte(`{}`)), nil
}

func (s *stubUpstream) CountTokens(ctx context.Context, payload []byte) (*http.Response, error) {
	if s.countTokensFunc != nil {
		return s.countTokensFunc(ctx, payload)
	}
	return newHTTPResponse(http.StatusOK, []byte(`{"response":{"totalTokens":0}}`)), nil
}

func (s *stubUpstream) Action(ctx context.Context, action string, payload []byte) (*http.Response, error) {
	if s.actionFunc != nil {
		return s.actionFunc(ctx, action, payload)
	}
	return newHTTPResponse(http.StatusOK, []byte(`{}`)), nil
}

func newHTTPResponse(status int, body []byte) *http.Response {
	if status == 0 {
		status = http.StatusOK
	}
	if body == nil {
		body = []byte("{}")
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
}

func newHandlerForTests(cfg *config.Config, client upstreamClient) *Handler {
	if cfg == nil {
		cfg = &config.Config{}
	}
	if client == nil {
		client = &stubUpstream{}
	}
	return &Handler{
		cfg:         cfg,
		cl:          client,
		clientCache: make(map[string]upstreamClient),
	}
}
