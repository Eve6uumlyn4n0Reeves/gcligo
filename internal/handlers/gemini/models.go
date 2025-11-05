package gemini

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gcli2api-go/internal/models"
)

// Models returns list of models with richer metadata (Gemini native style).
func (h *Handler) Models(c *gin.Context) {
	items := make([]any, 0)
	for _, m := range models.ExposedModelIDsByChannel(h.cfg, h.store, "gemini") {
		items = append(items, gin.H{
			"name":                       "models/" + m,
			"baseModelId":                m,
			"version":                    "001",
			"displayName":                m,
			"description":                "Gemini model: " + m,
			"inputTokenLimit":            1048576,
			"outputTokenLimit":           8192,
			"supportedGenerationMethods": []string{"generateContent", "streamGenerateContent", "countTokens"},
		})
	}
	resp := gin.H{"models": items}
	c.JSON(http.StatusOK, resp)
}

// ModelInfo returns metadata for a single model.
func (h *Handler) ModelInfo(c *gin.Context) {
	model := c.Param("model")
	c.JSON(http.StatusOK, gin.H{
		"name":                       "models/" + model,
		"version":                    "001",
		"displayName":                model,
		"description":                "Gemini model: " + model,
		"inputTokenLimit":            1048576,
		"outputTokenLimit":           8192,
		"supportedGenerationMethods": []string{"generateContent", "streamGenerateContent", "countTokens"},
	})
}

// ListModels delegates to Models for backward compatibility.
func (h *Handler) ListModels(c *gin.Context) { h.Models(c) }

// GetModel delegates to ModelInfo.
func (h *Handler) GetModel(c *gin.Context) { h.ModelInfo(c) }
