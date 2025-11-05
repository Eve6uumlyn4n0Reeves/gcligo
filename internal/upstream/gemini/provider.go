package gemini

import (
	"context"
	"strings"
	"sync"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/oauth"
	"gcli2api-go/internal/upstream"
)

// Provider 实现 upstream.Provider，用于 Gemini Code Assist 上游。
type Provider struct {
	cfg        *config.Config
	baseClient *Client
	cacheMu    sync.RWMutex
	cache      map[string]*Client
}

// NewProvider 创建 provider。
func NewProvider(cfg *config.Config) *Provider {
	return &Provider{
		cfg:        cfg,
		baseClient: New(cfg).WithCaller("upstream"),
		cache:      make(map[string]*Client),
	}
}

// BaseClient returns the shared client without credential binding.
func (p *Provider) BaseClient() *Client {
	return p.baseClient
}

func (p *Provider) Name() string { return "code_assist" }

func (p *Provider) SupportsModel(baseModel string) bool {
	if baseModel == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(baseModel), "gemini-")
}

func (p *Provider) Stream(ctx upstream.RequestContext) upstream.ProviderResponse {
	client := p.clientFor(ctx.Credential)
	if ctx.Ctx == nil {
		ctx.Ctx = context.Background()
	}
	reqCtx := upstream.WithHeaderOverrides(ctx.Ctx, ctx.HeaderOverrides)
	resp, err := client.Stream(reqCtx, ctx.Body)
	return upstream.ProviderResponse{Resp: resp, UsedModel: ctx.BaseModel, Err: err, Credential: ctx.Credential}
}

func (p *Provider) Generate(ctx upstream.RequestContext) upstream.ProviderResponse {
	client := p.clientFor(ctx.Credential)
	if ctx.Ctx == nil {
		ctx.Ctx = context.Background()
	}
	reqCtx := upstream.WithHeaderOverrides(ctx.Ctx, ctx.HeaderOverrides)
	resp, err := client.Generate(reqCtx, ctx.Body)
	return upstream.ProviderResponse{Resp: resp, UsedModel: ctx.BaseModel, Err: err, Credential: ctx.Credential}
}

func (p *Provider) ListModels(ctx upstream.RequestContext) upstream.ProviderListResponse {
	client := p.clientFor(ctx.Credential)
	project := strings.TrimSpace(ctx.ProjectID)
	if project == "" && ctx.Credential != nil && strings.TrimSpace(ctx.Credential.ProjectID) != "" {
		project = strings.TrimSpace(ctx.Credential.ProjectID)
	}
	if project == "" {
		project = strings.TrimSpace(p.cfg.GoogleProjID)
	}
	if ctx.Ctx == nil {
		ctx.Ctx = context.Background()
	}
	models, err := client.ListModels(ctx.Ctx, project)
	return upstream.ProviderListResponse{Models: models, Err: err, Credential: ctx.Credential}
}

func (p *Provider) clientFor(cred *credential.Credential) *Client {
	if cred == nil || cred.ID == "" {
		return p.baseClient
	}
	p.cacheMu.RLock()
	if c, ok := p.cache[cred.ID]; ok {
		p.cacheMu.RUnlock()
		return c
	}
	p.cacheMu.RUnlock()

	oc := &oauth.Credentials{
		AccessToken: cred.AccessToken,
		ProjectID:   cred.ProjectID,
	}
	c := NewWithCredential(p.cfg, oc).WithCaller("upstream")
	p.cacheMu.Lock()
	p.cache[cred.ID] = c
	p.cacheMu.Unlock()
	return c
}

// Invalidate removes a cached client for a credential id, forcing rebuild on next use.
func (p *Provider) Invalidate(credID string) {
	if credID == "" {
		return
	}
	p.cacheMu.Lock()
	delete(p.cache, credID)
	p.cacheMu.Unlock()
}
