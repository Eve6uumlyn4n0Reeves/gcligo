package openai

import (
	"net/http"
	"strings"
	"time"

	common "gcli2api-go/internal/handlers/common"
	"gcli2api-go/internal/models"
	"github.com/gin-gonic/gin"
)

// GET /v1/models
func (h *Handler) ListModels(c *gin.Context) {
	items := make([]any, 0)

	// Check if model variants are enabled (default: true)
	enableVariants := true
	if h.cfg != nil && h.cfg.DisableModelVariants {
		enableVariants = false
	}

	// Prefer dynamic registry entries; fallback to base ids when empty
	entries := models.ActiveEntriesByChannel(h.cfg, h.store, "openai")
	if len(entries) > 0 {
		// disabled filter already applied by ActiveEntriesByChannel; also respect global DisabledModels if set
		disabled := map[string]struct{}{}
		for _, d := range h.cfg.DisabledModels {
			if d != "" {
				disabled[d] = struct{}{}
			}
		}

		// If variants are enabled, generate all variants for each base model
		if enableVariants {
			// Collect base models from entries
			baseModels := make([]string, 0)
			for _, e := range entries {
				if _, off := disabled[e.ID]; off {
					continue
				}
				baseModels = append(baseModels, e.ID)
			}

			// Generate all variants
			allVariants := models.GenerateVariantsForModels(baseModels)
			for _, variant := range allVariants {
				if _, off := disabled[variant]; off {
					continue
				}
				base := strings.ToLower(models.BaseFromFeature(variant))
				modalities := []string{"text"}
				if cap, ok := models.GetCapability(h.store, variant); ok && len(cap.Modalities) > 0 {
					modalities = cap.Modalities
				} else if strings.Contains(base, "flash-image") {
					modalities = []string{"image", "text"}
				}
				caps := gin.H{"completion": true, "chat": true, "images": (contains(modalities, "image"))}
				items = append(items, gin.H{
					"id": variant, "object": "model", "owned_by": "gcli2api-go",
					"created": time.Now().Unix(), "modalities": modalities,
					"description": "Gemini model with feature variants", "context_length": 1048576,
					"capabilities": caps,
				})
			}
		} else {
			// Original behavior: only list base models
			for _, e := range entries {
				if _, off := disabled[e.ID]; off {
					continue
				}
				base := strings.ToLower(models.BaseFromFeature(e.ID))
				modalities := []string{"text"}
				if cap, ok := models.GetCapability(h.store, e.ID); ok && len(cap.Modalities) > 0 {
					modalities = cap.Modalities
				} else if e.Image || strings.Contains(base, "flash-image") {
					modalities = []string{"image", "text"}
				}
				caps := gin.H{"completion": true, "chat": true, "images": (contains(modalities, "image"))}
				items = append(items, gin.H{
					"id": e.ID, "object": "model", "owned_by": "gcli2api-go",
					"created": time.Now().Unix(), "modalities": modalities,
					"description": "Gemini base model", "context_length": 1048576,
					"capabilities": caps,
				})
			}
		}
	} else {
		ids := h.cfg.PreferredBaseModels
		if len(ids) == 0 {
			ids = models.DefaultBaseModels()
		}
		disabled := map[string]struct{}{}
		for _, d := range h.cfg.DisabledModels {
			if d != "" {
				disabled[d] = struct{}{}
			}
		}

		// If variants are enabled, generate all variants
		if enableVariants {
			allVariants := models.GenerateVariantsForModels(ids)
			for _, variant := range allVariants {
				if _, off := disabled[variant]; off {
					continue
				}
				modalities := []string{"text"}
				if cap, ok := models.GetCapability(h.store, variant); ok && len(cap.Modalities) > 0 {
					modalities = cap.Modalities
				} else {
					base := strings.ToLower(models.BaseFromFeature(variant))
					if strings.Contains(base, "flash-image") {
						modalities = []string{"image", "text"}
					}
				}
				items = append(items, gin.H{
					"id": variant, "object": "model", "owned_by": "gcli2api-go",
					"created": time.Now().Unix(), "modalities": modalities,
					"description": "Gemini model with feature variants", "context_length": 1048576,
					"capabilities": gin.H{"completion": true, "chat": true, "images": contains(modalities, "image")},
				})
			}
		} else {
			// Original behavior: only list base models
			for _, id := range ids {
				if _, off := disabled[id]; off {
					continue
				}
				modalities := []string{"text"}
				if cap, ok := models.GetCapability(h.store, id); ok && len(cap.Modalities) > 0 {
					modalities = cap.Modalities
				} else {
					base := strings.ToLower(models.BaseFromFeature(id))
					if strings.Contains(base, "flash-image") {
						modalities = []string{"image", "text"}
					}
				}
				items = append(items, gin.H{
					"id": id, "object": "model", "owned_by": "gcli2api-go",
					"created": time.Now().Unix(), "modalities": modalities,
					"description": "Gemini base model", "context_length": 1048576,
					"capabilities": gin.H{"completion": true, "chat": true, "images": contains(modalities, "image")},
				})
			}
		}
	}
	// nano-banana 不再对外暴露为模型；作为别名在请求时解析并映射到 Gemini 模型
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": items})
}

// GET /v1/models/:id
func (h *Handler) GetModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.AbortWithError(c, http.StatusBadRequest, "invalid_request_error", "missing id")
		return
	}
	base := strings.ToLower(models.BaseFromFeature(id))
	modalities := []string{"text"}
	if cap, ok := models.GetCapability(h.store, id); ok && len(cap.Modalities) > 0 {
		modalities = cap.Modalities
	} else if strings.Contains(base, "flash-image") {
		modalities = []string{"image", "text"}
	}
	c.JSON(http.StatusOK, gin.H{
		"id": id, "object": "model", "owned_by": "gcli2api-go", "created": time.Now().Unix(),
		"modalities": modalities, "description": "Gemini base model", "context_length": 1048576,
		"capabilities": gin.H{"completion": true, "chat": true, "images": contains(modalities, "image")},
	})
}

// mapFinishReason maps Gemini finish reason to OpenAI style
func mapFinishReason(fr string) string {
	switch fr {
	case "MAX_TOKENS":
		return "length"
	case "STOP", "STOPPED":
		return "stop"
	case "SAFETY", "BLOCKLIST", "PROHIBITED_CONTENT", "RECITATION":
		return "content_filter"
	default:
		return "stop"
	}
}

func contains(arr []string, s string) bool {
	for _, v := range arr {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
