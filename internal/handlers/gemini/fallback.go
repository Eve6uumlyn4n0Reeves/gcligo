package gemini

import (
	"context"
	"encoding/json"
	"net/http"

	credpkg "gcli2api-go/internal/credential"
	"gcli2api-go/internal/models"
	upstream "gcli2api-go/internal/upstream"
)

// usedCredSafe dereferences an optional credential pointer.
func usedCredSafe(ptr **credpkg.Credential) *credpkg.Credential {
	if ptr != nil && *ptr != nil {
		return *ptr
	}
	return nil
}

// tryGenerateWithFallback iterates model fallback bases for non-stream requests.
func (h *Handler) tryGenerateWithFallback(ctx context.Context, client upstreamClient, usedCred **credpkg.Credential, baseModel string, projectID string, req map[string]any) (*http.Response, string, error) {
	bases := models.FallbackBases(baseModel)
	var lastErr error
	var lastResp *http.Response
	for _, attempt := range bases {
		do := func(cur *credpkg.Credential) (*http.Response, error) {
			effProject := projectID
			if cur != nil && cur.ProjectID != "" {
				effProject = cur.ProjectID
			}
			payload := map[string]any{"model": attempt, "project": effProject, "request": req}
			b, _ := json.Marshal(payload)
			return client.Generate(ctx, b)
		}
		resp, cred, err := upstream.TryWithRotation(ctx, h.credMgr, h.router, usedCredSafe(usedCred), upstream.RotationOptions{MaxRotations: 0, RotateOn5xx: h.cfg.RetryOn5xx}, do)
		if err == nil && resp != nil && resp.StatusCode < 400 {
			if usedCred != nil {
				*usedCred = cred
			}
			return resp, attempt, nil
		}
		lastResp = resp
		lastErr = err
	}
	return lastResp, baseModel, lastErr
}

// tryStreamWithFallback iterates model fallback bases for streaming requests.
func (h *Handler) tryStreamWithFallback(ctx context.Context, client upstreamClient, usedCred **credpkg.Credential, baseModel string, projectID string, req map[string]any) (*http.Response, string, error) {
	bases := models.FallbackBases(baseModel)
	var lastErr error
	var lastResp *http.Response
	for _, attempt := range bases {
		do := func(cur *credpkg.Credential) (*http.Response, error) {
			effProject := projectID
			if cur != nil && cur.ProjectID != "" {
				effProject = cur.ProjectID
			}
			payload := map[string]any{"model": attempt, "project": effProject, "request": req}
			b, _ := json.Marshal(payload)
			return client.Stream(ctx, b)
		}
		resp, cred, err := upstream.TryWithRotation(ctx, h.credMgr, h.router, usedCredSafe(usedCred), upstream.RotationOptions{MaxRotations: 0, RotateOn5xx: h.cfg.RetryOn5xx}, do)
		if err == nil && resp != nil && resp.StatusCode < 400 {
			if usedCred != nil {
				*usedCred = cred
			}
			return resp, attempt, nil
		}
		lastResp = resp
		lastErr = err
	}
	return lastResp, baseModel, lastErr
}
