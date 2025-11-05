package openai

import (
	"context"

	"gcli2api-go/internal/credential"
	hcommon "gcli2api-go/internal/handlers/common"
	"gcli2api-go/internal/oauth"
	upstream "gcli2api-go/internal/upstream"
)

func (h *Handler) invalidateClientCache(credID string) {
	if credID == "" {
		return
	}
	h.cacheMu.Lock()
	delete(h.clientCache, credID)
	h.cacheMu.Unlock()
}

func (h *Handler) invalidateProviderCache(credID string) {
	if credID == "" || h.providers == nil {
		return
	}
	for _, p := range h.providers.Providers() {
		p.Invalidate(credID)
	}
}

func (h *Handler) shouldRefreshAhead(c *credential.Credential) bool {
	return hcommon.ShouldRefreshAhead(h.cfg, c)
}

func (h *Handler) acquireCredential(ctx context.Context) (*credential.Credential, error) {
	if h.credMgr == nil {
		return nil, nil
	}
	cred, err := h.credMgr.GetCredential()
	if err != nil {
		return nil, err
	}
	return cred, nil
}

func (h *Handler) getUpstreamClient(ctx context.Context) (geminiClient, *credential.Credential) {
	cred, err := h.acquireCredential(ctx)
	if err != nil || cred == nil {
		return h.baseClient, nil
	}
	if h.router != nil {
		if picked := h.router.Pick(ctx, upstream.HeaderOverrides(ctx)); picked != nil {
			return h.getClientFor(picked), picked
		}
	}
	cred = h.router.PrepareCredential(ctx, cred)
	return h.getClientFor(cred), cred
}

func (h *Handler) getClientFor(cred *credential.Credential) geminiClient {
	if cred == nil || cred.ID == "" {
		return h.baseClient
	}
	h.cacheMu.RLock()
	if c, ok := h.clientCache[cred.ID]; ok && c != nil {
		h.cacheMu.RUnlock()
		return c
	}
	h.cacheMu.RUnlock()
	oc := &oauth.Credentials{AccessToken: cred.AccessToken, ProjectID: cred.ProjectID}
	_ = oc
	client := hcommon.UpstreamClientFor(h.cfg, cred, "openai")
	h.cacheMu.Lock()
	h.clientCache[cred.ID] = client
	h.cacheMu.Unlock()
	return client
}
