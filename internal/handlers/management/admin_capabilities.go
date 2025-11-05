package management

import (
	"net/http"
	"strings"

	"gcli2api-go/internal/models"
	"github.com/gin-gonic/gin"
)

// GET /models/capabilities
func (h *AdminAPIHandler) GetModelCapabilities(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusOK, gin.H{"capabilities": gin.H{}})
		return
	}
	v, err := h.storage.GetConfig(c.Request.Context(), "model_capabilities")
	if err != nil || v == nil {
		c.JSON(http.StatusOK, gin.H{"capabilities": gin.H{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"capabilities": v})
}

// PUT /models/capabilities  { id/base -> Capability }
func (h *AdminAPIHandler) UpsertModelCapabilities(c *gin.Context) {
	if !h.isAdminRequest(c) {
		respondError(c, http.StatusForbidden, "admin required")
		return
	}
	var body map[string]models.Capability
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid json")
		return
	}
	// 仅允许写入功能字段；source/updated_at 由服务器填充
	norm := make(map[string]models.Capability, len(body))
	for k, v := range body {
		norm[strings.ToLower(strings.TrimSpace(k))] = models.Capability{
			Modalities:    v.Modalities,
			ContextLength: v.ContextLength,
			Images:        v.Images,
			Thinking:      v.Thinking,
		}
	}
	if err := models.UpsertCapabilitiesWithSource(h.storage, norm, "manual"); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// POST /models/capabilities/seed-defaults
// 从内置基表推导一批默认能力（modalities/context_length/thinking），标记来源 upstream
func (h *AdminAPIHandler) SeedModelCapabilities(c *gin.Context) {
	if !h.isAdminRequest(c) {
		respondError(c, http.StatusForbidden, "admin required")
		return
	}
	caps := models.DefaultCapabilities()
	if len(caps) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "no defaults"})
		return
	}
	if err := models.UpsertCapabilitiesWithSource(h.storage, caps, "upstream"); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "seeded", "count": len(caps)})
}
