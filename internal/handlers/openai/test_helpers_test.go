package openai

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	hcommon "gcli2api-go/internal/handlers/common"
	upstream "gcli2api-go/internal/upstream"
)

type stubGeminiClient struct {
	generateFunc    func(context.Context, []byte) (*http.Response, error)
	streamFunc      func(context.Context, []byte) (*http.Response, error)
	countTokensFunc func(context.Context, []byte) (*http.Response, error)
	actionFunc      func(context.Context, string, []byte) (*http.Response, error)
}

func (s *stubGeminiClient) Generate(ctx context.Context, payload []byte) (*http.Response, error) {
	if s.generateFunc != nil {
		return s.generateFunc(ctx, payload)
	}
	return nil, errors.New("generate not implemented")
}

func (s *stubGeminiClient) Stream(ctx context.Context, payload []byte) (*http.Response, error) {
	if s.streamFunc != nil {
		return s.streamFunc(ctx, payload)
	}
	return nil, errors.New("stream not implemented")
}

func (s *stubGeminiClient) CountTokens(ctx context.Context, payload []byte) (*http.Response, error) {
	if s.countTokensFunc != nil {
		return s.countTokensFunc(ctx, payload)
	}
	return nil, errors.New("countTokens not implemented")
}

func (s *stubGeminiClient) Action(ctx context.Context, action string, payload []byte) (*http.Response, error) {
	if s.actionFunc != nil {
		return s.actionFunc(ctx, action, payload)
	}
	return nil, errors.New("action not implemented")
}

type stubProvider struct {
	name           string
	generateFunc   func(upstream.RequestContext) upstream.ProviderResponse
	streamFunc     func(upstream.RequestContext) upstream.ProviderResponse
	listModelsFunc func(upstream.RequestContext) upstream.ProviderListResponse
}

func (s *stubProvider) Name() string {
	if s.name != "" {
		return s.name
	}
	return "stub"
}

func (s *stubProvider) SupportsModel(string) bool { return true }

func (s *stubProvider) Stream(ctx upstream.RequestContext) upstream.ProviderResponse {
	if s.streamFunc != nil {
		return s.streamFunc(ctx)
	}
	return upstream.ProviderResponse{}
}

func (s *stubProvider) Generate(ctx upstream.RequestContext) upstream.ProviderResponse {
	if s.generateFunc != nil {
		return s.generateFunc(ctx)
	}
	return upstream.ProviderResponse{}
}

func (s *stubProvider) ListModels(ctx upstream.RequestContext) upstream.ProviderListResponse {
	if s.listModelsFunc != nil {
		return s.listModelsFunc(ctx)
	}
	return upstream.ProviderListResponse{}
}

func (s *stubProvider) Invalidate(string) {}

func newHandlerForTests(cfg *config.Config, provider upstream.Provider, client geminiClient) *Handler {
	if cfg == nil {
		cfg = &config.Config{}
	}
	if client == nil {
		client = &stubGeminiClient{
			generateFunc: func(_ context.Context, _ []byte) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"response":{"candidates":[]}}`))),
					Header:     make(http.Header),
				}, nil
			},
		}
	}
	h := &Handler{
		cfg:         cfg,
		baseClient:  client,
		clientCache: make(map[string]geminiClient),
	}
	if provider != nil {
		h.providers = upstream.NewManager(provider)
	} else {
		h.providers = upstream.NewManager(&stubProvider{
			generateFunc: func(rc upstream.RequestContext) upstream.ProviderResponse {
				return upstream.ProviderResponse{
					Resp: &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte(`{"response":{"candidates":[]}}`))),
						Header:     make(http.Header),
					},
					UsedModel: rc.BaseModel,
				}
			},
		})
	}
	return h
}

func captureCredentialSuccess(h *Handler) *credential.Credential {
	if h == nil || h.credMgr == nil {
		return nil
	}
	if cred, _ := h.credMgr.GetCredential(); cred != nil {
		h.credMgr.MarkSuccess(cred.ID)
		return cred
	}
	return nil
}

func newStubResponse(body []byte, status int) *http.Response {
	if body == nil {
		body = []byte("{}")
	}
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
}

func stubProviderResponse(body []byte, status int, usedModel string, err error) upstream.ProviderResponse {
	resp := newStubResponse(body, status)
	return upstream.ProviderResponse{Resp: resp, Err: err, UsedModel: usedModel}
}

func stubUpstreamError(err error) upstream.ProviderResponse {
	return upstream.ProviderResponse{Resp: nil, Err: err}
}

func markCredFailure(h *Handler, cred *credential.Credential, status int) {
	if cred == nil {
		return
	}
	hcommon.MarkCredentialFailure(h.credMgr, h.router, cred, "test", status)
}
