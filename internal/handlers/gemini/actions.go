package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	common "gcli2api-go/internal/handlers/common"
	upstream "gcli2api-go/internal/upstream"
	up "gcli2api-go/internal/upstream/gemini"
)

// LoadCodeAssist proxies Gemini loadCodeAssist action.
func (h *Handler) LoadCodeAssist(c *gin.Context) {
	var request map[string]any
	if err := c.ShouldBindJSON(&request); err != nil {
		common.AbortWithError(c, http.StatusBadRequest, "invalid_request", "invalid json")
		return
	}
	if _, ok := request["metadata"].(map[string]any); !ok {
		request["metadata"] = map[string]any{"ideType": "IDE_UNSPECIFIED", "platform": "PLATFORM_UNSPECIFIED", "pluginType": "GEMINI"}
	}
	b, _ := json.Marshal(request)
	ctx, cancel := context.WithTimeout(up.WithHeaderOverrides(c.Request.Context(), c.Request.Header), 60*time.Second)
	defer cancel()
	client, usedCred := h.getUpstreamClient(ctx)
	if usedCred != nil && usedCred.ProjectID != "" {
		if _, exists := request["cloudaicompanionProject"]; !exists {
			request["cloudaicompanionProject"] = usedCred.ProjectID
			b, _ = json.Marshal(request)
		}
	}
	resp, err := client.Action(ctx, "loadCodeAssist", b)
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
		}
		common.AbortWithUpstreamError(c, resp.StatusCode, "upstream_error", "", by)
		return
	}
	if usedCred != nil {
		h.credMgr.MarkSuccess(usedCred.ID)
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), by)
}

// OnboardUser proxies Gemini onboardUser action.
func (h *Handler) OnboardUser(c *gin.Context) {
	var request map[string]any
	if err := c.ShouldBindJSON(&request); err != nil {
		common.AbortWithError(c, http.StatusBadRequest, "invalid_request", "invalid json")
		return
	}
	b, _ := json.Marshal(request)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	client, usedCred := h.getUpstreamClient(ctx)
	resp, err := client.Action(ctx, "onboardUser", b)
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
		}
		common.AbortWithUpstreamError(c, resp.StatusCode, "upstream_error", "", by)
		return
	}
	if usedCred != nil {
		h.credMgr.MarkSuccess(usedCred.ID)
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), by)
}
