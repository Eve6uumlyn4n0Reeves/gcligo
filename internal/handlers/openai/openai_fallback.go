package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gcli2api-go/internal/credential"
	logx "gcli2api-go/internal/logging"
	"gcli2api-go/internal/models"
	upstream "gcli2api-go/internal/upstream"
	"github.com/sirupsen/logrus"
)

func (h *Handler) logUpstreamEvent(level logrus.Level, msg string, base, attempt string, cred *credential.Credential, status int, err error) {
	fields := logrus.Fields{
		"component":     "openai_handler",
		"base_model":    base,
		"attempt_model": attempt,
		"status":        status,
		"upstream":      "gemini",
		"fallback":      (attempt != "" && base != "" && attempt != base),
	}
	if cred != nil {
		fields["cred_id"] = cred.ID
	}
	fields["error_kind"] = logx.ErrorKind(status, err != nil)
	entry := logrus.WithFields(fields)
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Log(level, msg)
}

// tryStreamWithFallback attempts streaming with model fallback and optional credential rotation on 429.
func (h *Handler) tryStreamWithFallback(ctx context.Context, usedCred **credential.Credential, baseModel string, projectID string, gemReq map[string]any) (*http.Response, string, error) {
	bases := models.FallbackBases(baseModel)
	var lastErr error
	var lastResp *http.Response
	headerOverrides := upstream.HeaderOverrides(ctx)
	for _, attempt := range bases {
		provider := h.providers.ProviderFor(models.BaseFromFeature(attempt))
		if provider == nil {
			lastErr = fmt.Errorf("no upstream provider available for %s", attempt)
			continue
		}
		do := func(cur *credential.Credential) (*http.Response, error) {
			currentProject := strings.TrimSpace(projectID)
			if cur != nil && strings.TrimSpace(cur.ProjectID) != "" {
				currentProject = strings.TrimSpace(cur.ProjectID)
			}
			if currentProject == "" {
				currentProject = strings.TrimSpace(h.cfg.GoogleProjID)
			}
			payload := map[string]any{"model": attempt, "project": currentProject, "request": gemReq}
			body, _ := json.Marshal(payload)
			reqCtx := upstream.RequestContext{Ctx: ctx, Credential: cur, BaseModel: attempt, ProjectID: currentProject, Body: body, HeaderOverrides: headerOverrides}
			res := provider.Stream(reqCtx)
			return res.Resp, res.Err
		}
		resp, cred, err := upstream.TryWithRotation(ctx, h.credMgr, h.router, nil, upstream.RotationOptions{MaxRotations: 0, RotateOn5xx: true}, do)
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		h.logUpstreamEvent(logrus.DebugLevel, "stream attempt upstream", baseModel, attempt, cred, status, err)
		if err == nil && resp != nil && resp.StatusCode < 400 {
			h.logUpstreamEvent(logrus.InfoLevel, "stream upstream success", baseModel, attempt, cred, resp.StatusCode, nil)
			if usedCred != nil {
				*usedCred = cred
			}
			return resp, attempt, nil
		}
		if resp != nil {
			lastResp = resp
		}
		lastErr = err
		h.logUpstreamEvent(logrus.WarnLevel, "stream upstream failed", baseModel, attempt, cred, status, err)
	}
	return lastResp, baseModel, lastErr
}

// tryGenerateWithFallback attempts non-stream call with model fallback and credential rotation on 429.
func (h *Handler) tryGenerateWithFallback(ctx context.Context, usedCred **credential.Credential, baseModel string, projectID string, gemReq map[string]any) (*http.Response, string, error) {
	bases := models.FallbackBases(baseModel)
	var lastErr error
	var lastResp *http.Response
	headerOverrides := upstream.HeaderOverrides(ctx)
	for _, attempt := range bases {
		provider := h.providers.ProviderFor(models.BaseFromFeature(attempt))
		if provider == nil {
			lastErr = fmt.Errorf("no upstream provider available for %s", attempt)
			continue
		}
		do := func(cur *credential.Credential) (*http.Response, error) {
			currentProject := strings.TrimSpace(projectID)
			if cur != nil && strings.TrimSpace(cur.ProjectID) != "" {
				currentProject = strings.TrimSpace(cur.ProjectID)
			}
			if currentProject == "" {
				currentProject = strings.TrimSpace(h.cfg.GoogleProjID)
			}
			payload := map[string]any{"model": attempt, "project": currentProject, "request": gemReq}
			body, _ := json.Marshal(payload)
			reqCtx := upstream.RequestContext{Ctx: ctx, Credential: cur, BaseModel: attempt, ProjectID: currentProject, Body: body, HeaderOverrides: headerOverrides}
			res := provider.Generate(reqCtx)
			return res.Resp, res.Err
		}
		resp, cred, err := upstream.TryWithRotation(ctx, h.credMgr, h.router, nil, upstream.RotationOptions{MaxRotations: 0, RotateOn5xx: true}, do)
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		h.logUpstreamEvent(logrus.DebugLevel, "generate upstream attempt", baseModel, attempt, cred, status, err)
		if err == nil && resp != nil && resp.StatusCode < 400 {
			h.logUpstreamEvent(logrus.InfoLevel, "generate upstream success", baseModel, attempt, cred, resp.StatusCode, nil)
			if usedCred != nil {
				*usedCred = cred
			}
			return resp, attempt, nil
		}
		if resp != nil {
			lastResp = resp
		}
		lastErr = err
		h.logUpstreamEvent(logrus.WarnLevel, "generate upstream failed", baseModel, attempt, cred, status, err)
	}
	return lastResp, baseModel, lastErr
}
