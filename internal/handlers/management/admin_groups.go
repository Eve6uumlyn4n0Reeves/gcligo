package management

import (
	"encoding/json"
	"net/http"
	"strings"

	"gcli2api-go/internal/models"
	"github.com/gin-gonic/gin"
)

// ListGroups returns existing groups
func (h *AdminAPIHandler) ListGroups(c *gin.Context) { h.ListGroupsByChannel(withChannel(c, "openai")) }

func (h *AdminAPIHandler) ListGroupsByChannel(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusOK, gin.H{"groups": []any{}})
		return
	}
	v, err := h.storage.GetConfig(c.Request.Context(), groupKey(c.Param("channel")))
	if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		c.JSON(http.StatusOK, gin.H{"groups": []any{}})
		return
	}
	if v == nil {
		c.JSON(http.StatusOK, gin.H{"groups": []any{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": v})
}

// CreateGroup creates a new group
func (h *AdminAPIHandler) CreateGroup(c *gin.Context) {
	h.CreateGroupByChannel(withChannel(c, "openai"))
}

func (h *AdminAPIHandler) CreateGroupByChannel(c *gin.Context) {
	if h.storage == nil {
		respondError(c, http.StatusNotImplemented, "storage not configured")
		return
	}
	var g models.GroupEntry
	if err := c.ShouldBindJSON(&g); err != nil {
		respondError(c, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(g.Name) == "" {
		respondError(c, http.StatusBadRequest, "missing name")
		return
	}
	if strings.TrimSpace(g.ID) == "" {
		g.ID = g.Name
	}
	// load existing
	var groups []models.GroupEntry
	if v, err := h.storage.GetConfig(c.Request.Context(), groupKey(c.Param("channel"))); err == nil && v != nil {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &groups)
	} else if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
	}
	// add if not exists
	for _, ex := range groups {
		if ex.ID == g.ID {
			respondError(c, http.StatusConflict, "group exists")
			return
		}
	}
	groups = append(groups, g)
	if err := h.storage.SetConfig(c.Request.Context(), groupKey(c.Param("channel")), groups); err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "created", "id": g.ID})
}

// UpdateGroup updates an existing group by id
func (h *AdminAPIHandler) UpdateGroup(c *gin.Context) {
	h.UpdateGroupByChannel(withChannel(c, "openai"))
}

func (h *AdminAPIHandler) UpdateGroupByChannel(c *gin.Context) {
	if h.storage == nil {
		respondError(c, http.StatusNotImplemented, "storage not configured")
		return
	}
	id := c.Param("id")
	var g models.GroupEntry
	if err := c.ShouldBindJSON(&g); err != nil {
		respondError(c, http.StatusBadRequest, "invalid json")
		return
	}
	var groups []models.GroupEntry
	if v, err := h.storage.GetConfig(c.Request.Context(), groupKey(c.Param("channel"))); err == nil && v != nil {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &groups)
	} else if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
	}
	updated := false
	for i := range groups {
		if groups[i].ID == id {
			if strings.TrimSpace(g.ID) == "" {
				g.ID = id
			}
			groups[i] = g
			updated = true
			break
		}
	}
	if !updated {
		respondError(c, http.StatusNotFound, "group not found")
		return
	}
	if err := h.storage.SetConfig(c.Request.Context(), groupKey(c.Param("channel")), groups); err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated", "id": g.ID})
}

// DeleteGroup deletes a group by id
func (h *AdminAPIHandler) DeleteGroup(c *gin.Context) {
	h.DeleteGroupByChannel(withChannel(c, "openai"))
}

func (h *AdminAPIHandler) DeleteGroupByChannel(c *gin.Context) {
	if h.storage == nil {
		respondError(c, http.StatusNotImplemented, "storage not configured")
		return
	}
	id := c.Param("id")
	var groups []models.GroupEntry
	if v, err := h.storage.GetConfig(c.Request.Context(), groupKey(c.Param("channel"))); err == nil && v != nil {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &groups)
	} else if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
	}
	out := make([]models.GroupEntry, 0, len(groups))
	removed := false
	for _, g := range groups {
		if g.ID == id {
			removed = true
			continue
		}
		out = append(out, g)
	}
	if !removed {
		respondError(c, http.StatusNotFound, "group not found")
		return
	}
	if err := h.storage.SetConfig(c.Request.Context(), groupKey(c.Param("channel")), out); err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted", "id": id})
}
