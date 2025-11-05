package openai

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"gcli2api-go/internal/models"
)

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
	}
	return 0
}

func (h *Handler) recordUsage(c *gin.Context, model string, success bool, usage map[string]any, promptTokens, completionTokens int64) {
	if h.usageStats == nil {
		return
	}
	if usage != nil {
		if v, ok := usage["prompt_tokens"]; ok {
			promptTokens = toInt64(v)
		}
		if v, ok := usage["completion_tokens"]; ok {
			completionTokens = toInt64(v)
		}
	}
	apiKey := "anonymous"
	if v, ok := c.Get("api_key"); ok {
		if s, ok := v.(string); ok && s != "" {
			apiKey = s
		}
	}
	baseModel := strings.TrimSpace(model)
	if baseModel != "" {
		if base := models.BaseFromFeature(baseModel); base != "" {
			baseModel = base
		}
	}
	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := h.usageStats.RecordRequest(ctx, apiKey, baseModel, success, promptTokens, completionTokens); err != nil {
		log.WithError(err).Debug("record usage failed")
	}
}

func toJSONString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}
