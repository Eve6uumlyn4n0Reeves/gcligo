package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	credpkg "gcli2api-go/internal/credential"
	feat "gcli2api-go/internal/features"
	common "gcli2api-go/internal/handlers/common"
	mw "gcli2api-go/internal/middleware"
	"gcli2api-go/internal/models"
	upstream "gcli2api-go/internal/upstream"
	up "gcli2api-go/internal/upstream/gemini"
)

// GenerateContent handles non-stream Gemini generate requests.
func (h *Handler) GenerateContent(c *gin.Context) {
	model := c.Param("model")
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		common.AbortWithError(c, http.StatusBadRequest, "invalid_request", "invalid json")
		return
	}
	base := models.BaseFromFeature(model)
	req := h.applyRequestDecorators(model, body)
	baseCtx := c.Request.Context()
	baseCtx = up.WithHeaderOverrides(baseCtx, c.Request.Header)
	ctx, cancel := context.WithTimeout(baseCtx, 180*time.Second)
	defer cancel()
	// Use strategy pick to allow debug headers.
	var usedCred *credpkg.Credential
	if h.router != nil {
		if cred, info := h.router.PickWithInfo(ctx, upstream.HeaderOverrides(ctx)); cred != nil {
			usedCred = cred
			if h.cfg.RoutingDebugHeaders {
				if info != nil {
					c.Writer.Header().Set("X-Routing-Credential", info.CredID)
					c.Writer.Header().Set("X-Routing-Reason", info.Reason)
					if info.StickySource != "" {
						c.Writer.Header().Set("X-Routing-Sticky-Source", info.StickySource)
					}
				} else {
					c.Writer.Header().Set("X-Routing-Credential", cred.ID)
				}
			}
		}
	}
	client := h.cl
	if usedCred != nil {
		client = h.getClientFor(usedCred)
	} else {
		client, usedCred = h.getUpstreamClient(ctx)
	}
	effProject := h.cfg.GoogleProjID
	if usedCred != nil && usedCred.ProjectID != "" {
		effProject = usedCred.ProjectID
	}
	resp, usedModel, err := h.tryGenerateWithFallback(ctx, client, &usedCred, base, effProject, req)
	if err != nil {
		common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	by, err := upstream.ReadAll(resp)
	if err != nil {
		if usedCred != nil {
			h.credMgr.MarkFailure(usedCred.ID, "read_error", 0)
		}
		common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	if resp.StatusCode >= 400 {
		if usedCred != nil {
			h.credMgr.MarkFailure(usedCred.ID, "upstream_error", resp.StatusCode)
			if h.router != nil {
				h.router.OnResult(usedCred.ID, resp.StatusCode)
			}
		}
		common.AbortWithUpstreamError(c, resp.StatusCode, "upstream_error", "", by)
		return
	}
	// upstream may return {response: {...}} or direct.
	var obj map[string]any
	_ = json.Unmarshal(by, &obj)
	if usedModel != "" && usedModel != base {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		mw.RecordFallback("gemini", path, base, usedModel)
	}
	// Non-stream anti-truncation continuation for first candidate (append text).
	if models.IsAntiTruncation(model) || h.cfg.AntiTruncationEnabled {
		parsed, _ := common.ExtractFromResponse(obj)
		sh := feat.NewStreamHandler(feat.AntiTruncationConfig{MaxAttempts: h.cfg.AntiTruncationMax, Enabled: true})
		contFn := func(ctx context.Context) (string, error) {
			p2 := map[string]any{"model": base, "project": effProject, "request": req}
			b2, _ := json.Marshal(p2)
			r2, err := client.Generate(ctx, b2)
			if err != nil {
				return "", err
			}
			by2, err := upstream.ReadAll(r2)
			if err != nil {
				return "", err
			}
			var o map[string]any
			if json.Unmarshal(by2, &o) != nil {
				return "", nil
			}
			p, _ := common.ExtractFromResponse(o)
			mw.RecordAntiTruncAttempt("gemini", c.FullPath(), 1)
			return p.Text, nil
		}
		if full, err := sh.DetectAndHandle(c.Request.Context(), parsed.Text, contFn); err == nil && full != "" && full != parsed.Text {
			respMap, ok := obj["response"].(map[string]any)
			if !ok {
				respMap = obj
			}
			if cands, ok := respMap["candidates"].([]any); ok && len(cands) > 0 {
				if cand, ok := cands[0].(map[string]any); ok {
					content, _ := cand["content"].(map[string]any)
					if content == nil {
						content = map[string]any{}
					}
					content["parts"] = []any{map[string]any{"text": full}}
					cand["content"] = content
					cands[0] = cand
					respMap["candidates"] = cands
					obj["response"] = respMap
				}
			}
		}
	}
	if usedCred != nil {
		h.credMgr.MarkSuccess(usedCred.ID)
		if h.router != nil {
			h.router.OnResult(usedCred.ID, 200)
		}
	}
	if r, ok := obj["response"]; ok {
		c.JSON(http.StatusOK, r)
		return
	}
	// passthrough
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), by)
}
