package management

import (
	"encoding/json"
	"net/http"
	"strings"

	"gcli2api-go/internal/models"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Model template endpoints
type modelTemplate struct {
	Base          string `json:"base"`
	Thinking      string `json:"thinking"`
	FakeStreaming bool   `json:"fake_streaming"`
	AntiTrunc     bool   `json:"anti_truncation"`
	Search        bool   `json:"search"`
	Image         bool   `json:"image"`
	Stream        bool   `json:"stream"`
	Group         string `json:"group"`
	Enabled       bool   `json:"enabled"`
}

func (h *AdminAPIHandler) GetModelTemplateByChannel(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusOK, gin.H{"template": gin.H{}})
		return
	}
	v, err := h.storage.GetConfig(c.Request.Context(), templateKey(c.Param("channel")))
	if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		c.JSON(http.StatusOK, gin.H{"template": gin.H{}})
		return
	}
	if v == nil {
		c.JSON(http.StatusOK, gin.H{"template": gin.H{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"template": v})
}

func (h *AdminAPIHandler) UpdateModelTemplateByChannel(c *gin.Context) {
	if h.storage == nil {
		respondError(c, http.StatusNotImplemented, "storage not configured")
		return
	}
	var tpl modelTemplate
	if err := c.ShouldBindJSON(&tpl); err != nil {
		respondError(c, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.storage.SetConfig(c.Request.Context(), templateKey(c.Param("channel")), tpl); err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.audit(c, "model.template.update", nil)
	c.JSON(http.StatusOK, gin.H{"message": "template updated"})
}

// Bulk enable/disable endpoints
func (h *AdminAPIHandler) BulkEnableByChannel(c *gin.Context)  { h.bulkToggle(c, true) }
func (h *AdminAPIHandler) BulkDisableByChannel(c *gin.Context) { h.bulkToggle(c, false) }

func (h *AdminAPIHandler) bulkToggle(c *gin.Context, on bool) {
	if h.storage == nil {
		respondError(c, http.StatusNotImplemented, "storage not configured")
		return
	}
	group := strings.TrimSpace(c.Query("group"))
	key := channelKey(c.Param("channel"))
	var items []models.RegistryEntry
	if v, err := h.storage.GetConfig(c.Request.Context(), key); err == nil && v != nil {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &items)
	} else if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
	}
	changed := 0
	for i := range items {
		if group == "" || strings.EqualFold(items[i].Group, group) {
			if items[i].Enabled != on {
				items[i].Enabled = on
				changed++
			}
		}
	}
	if err := h.storage.SetConfig(c.Request.Context(), key, items); err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.audit(c, "model.bulk_toggle", log.Fields{"enabled": on, "changed": changed, "group": group})
	c.JSON(http.StatusOK, gin.H{"message": "toggled", "enabled": on, "changed": changed})
}
