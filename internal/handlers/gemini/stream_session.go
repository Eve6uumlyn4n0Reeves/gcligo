package gemini

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	credpkg "gcli2api-go/internal/credential"
	common "gcli2api-go/internal/handlers/common"
	mw "gcli2api-go/internal/middleware"
	"gcli2api-go/internal/models"
	upstream "gcli2api-go/internal/upstream"
	up "gcli2api-go/internal/upstream/gemini"
)

type streamSession struct {
	handler      *Handler
	ctx          context.Context
	ginCtx       *gin.Context
	model        string
	baseModel    string
	decoratedReq map[string]any
	usedCred     *credpkg.Credential
	client       upstreamClient
	effProject   string
	payloadBytes []byte
	useAnti      bool
	path         string
}

func newStreamSession(h *Handler, c *gin.Context) (*streamSession, bool) {
	model := c.Param("model")

	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		common.AbortWithError(c, http.StatusBadRequest, "invalid_request", "invalid json")
		return nil, true
	}

	decorated := h.applyRequestDecorators(model, body)
	baseModel := models.BaseFromFeature(model)

	ctx0 := c.Request.Context()
	overrideCtx := up.WithHeaderOverrides(ctx0, c.Request.Header)

	var usedCred *credpkg.Credential
	if h.router != nil {
		if cred, info := h.router.PickWithInfo(overrideCtx, upstream.HeaderOverrides(ctx0)); cred != nil {
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
		client, usedCred = h.getUpstreamClient(ctx0)
	}

	effProject := h.cfg.GoogleProjID
	if usedCred != nil && usedCred.ProjectID != "" {
		effProject = usedCred.ProjectID
	}

	payload := map[string]any{
		"model":   baseModel,
		"project": effProject,
		"request": decorated,
	}
	payloadBytes, _ := json.Marshal(payload)

	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}

	session := &streamSession{
		handler:      h,
		ctx:          overrideCtx,
		ginCtx:       c,
		model:        model,
		baseModel:    baseModel,
		decoratedReq: decorated,
		usedCred:     usedCred,
		client:       client,
		effProject:   effProject,
		payloadBytes: payloadBytes,
		useAnti:      models.IsAntiTruncation(model) || h.cfg.AntiTruncationEnabled,
		path:         path,
	}

	return session, false
}

func (s *streamSession) execute() {
	if models.IsFakeStreaming(s.model) {
		s.streamFake()
		return
	}

	resp, usedModel, err := s.handler.tryStreamWithFallback(s.ctx, s.client, &s.usedCred, s.baseModel, s.effProject, s.decoratedReq)
	if err != nil {
		if s.usedCred != nil {
			s.handler.credMgr.MarkFailure(s.usedCred.ID, "upstream_error", 0)
		}
		common.AbortWithError(s.ginCtx, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	if resp == nil {
		if s.usedCred != nil {
			s.handler.credMgr.MarkFailure(s.usedCred.ID, "upstream_error", 0)
		}
		common.AbortWithError(s.ginCtx, http.StatusBadGateway, "upstream_error", "empty response")
		return
	}
	if resp.StatusCode >= 400 {
		body, _ := upstream.ReadAll(resp)
		if s.usedCred != nil {
			s.handler.credMgr.MarkFailure(s.usedCred.ID, "upstream_error", resp.StatusCode)
		}
		common.AbortWithUpstreamError(s.ginCtx, http.StatusBadGateway, "upstream_error", "", body)
		return
	}
	defer resp.Body.Close()

	s.prepareSSEHeaders()

	if usedModel != "" && usedModel != s.baseModel {
		mw.RecordFallback("gemini", s.path, s.baseModel, usedModel)
	}

	reader := s.wrapResponseBody(resp.Body)
	stats := s.pumpStream(reader)

	mw.RecordSSELines("gemini", s.path, stats.sseCount)
	mw.RecordToolCalls("gemini", s.path, stats.toolCount)
	mw.RecordSSEClose("gemini", s.path, "ended")

	if s.usedCred != nil {
		s.handler.credMgr.MarkSuccess(s.usedCred.ID)
		if s.handler.router != nil {
			s.handler.router.OnResult(s.usedCred.ID, http.StatusOK)
		}
	}
}

func (s *streamSession) markFailure(reason string, status int) {
	if s.usedCred != nil {
		s.handler.credMgr.MarkFailure(s.usedCred.ID, reason, status)
		if s.handler.router != nil {
			s.handler.router.OnResult(s.usedCred.ID, status)
		}
	}
}
