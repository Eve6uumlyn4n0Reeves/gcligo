package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	common "gcli2api-go/internal/handlers/common"
	"gcli2api-go/internal/models"
	upstream "gcli2api-go/internal/upstream"
	up "gcli2api-go/internal/upstream/gemini"
)

// CountTokens proxies token count requests to upstream Gemini.
func (h *Handler) CountTokens(c *gin.Context) {
	model := c.Param("model")
	var request map[string]any
	if err := c.ShouldBindJSON(&request); err != nil {
		common.AbortWithError(c, http.StatusBadRequest, "invalid_request", "invalid json")
		return
	}
	client, usedCred := h.getUpstreamClient(c.Request.Context())
	effProject := h.cfg.GoogleProjID
	if usedCred != nil && usedCred.ProjectID != "" {
		effProject = usedCred.ProjectID
	}
	payload := map[string]any{"model": models.BaseFromFeature(model), "project": effProject, "request": request}
	b, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(up.WithHeaderOverrides(c.Request.Context(), c.Request.Header), 60*time.Second)
	defer cancel()
	resp, err := client.CountTokens(ctx, b)
	if err != nil {
		if usedCred != nil {
			h.credMgr.MarkFailure(usedCred.ID, "upstream_error", 0)
			if h.router != nil {
				h.router.OnResult(usedCred.ID, 0)
			}
		}
		common.AbortWithError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	by, err := upstream.ReadAll(resp)
	if err != nil {
		if usedCred != nil {
			h.credMgr.MarkFailure(usedCred.ID, "read_error", 0)
			if h.router != nil {
				h.router.OnResult(usedCred.ID, 0)
			}
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
	var obj map[string]any
	if json.Unmarshal(by, &obj) == nil {
		if r, ok := obj["response"]; ok {
			if usedCred != nil {
				h.credMgr.MarkSuccess(usedCred.ID)
				if h.router != nil {
					h.router.OnResult(usedCred.ID, 200)
				}
			}
			c.JSON(http.StatusOK, r)
			return
		}
	}
	if usedCred != nil {
		h.credMgr.MarkSuccess(usedCred.ID)
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), by)
}
