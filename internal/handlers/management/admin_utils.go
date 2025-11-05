package management

import (
	"errors"
	"net/http"
	"strings"

	"gcli2api-go/internal/storage"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Storage capability helpers
func isNotSupported(err error) bool {
	if err == nil {
		return false
	}
	var ns *storage.ErrNotSupported
	return errors.As(err, &ns)
}

func respondNotSupported(c *gin.Context) {
	respondError(c, http.StatusNotImplemented, "storage backend does not support this operation")
}

// respondError 统一管理端错误响应格式，兼容 OpenAI/Gemini 解析字段
func respondError(c *gin.Context, status int, message string, extra ...gin.H) {
	if status <= 0 {
		status = http.StatusInternalServerError
	}

	message = strings.TrimSpace(message)
	if message == "" {
		message = "error"
	}

	code := strings.ToLower(strings.ReplaceAll(http.StatusText(status), " ", "_"))
	if code == "" {
		code = "unknown_error"
	}

	errType := "management_error"
	statusLabel := strings.ToUpper(code)
	var details map[string]any

	if len(extra) > 0 && extra[0] != nil {
		clone := make(map[string]any, len(extra[0]))
		for k, v := range extra[0] {
			clone[k] = v
		}
		if v, ok := clone["code"].(string); ok && strings.TrimSpace(v) != "" {
			code = strings.TrimSpace(v)
			delete(clone, "code")
		}
		if v, ok := clone["type"].(string); ok && strings.TrimSpace(v) != "" {
			errType = strings.TrimSpace(v)
			delete(clone, "type")
		}
		if v, ok := clone["status"].(string); ok && strings.TrimSpace(v) != "" {
			statusLabel = strings.TrimSpace(v)
			delete(clone, "status")
		}
		if len(clone) > 0 {
			details = clone
		}
	}

	payload := gin.H{
		"message":   message,
		"type":      errType,
		"code":      code,
		"status":    statusLabel,
		"http_code": status,
	}
	if details != nil {
		payload["details"] = details
	}

	c.JSON(status, gin.H{"error": payload})
}

// Audit helper
func (h *AdminAPIHandler) audit(c *gin.Context, action string, fields log.Fields) {
	if fields == nil {
		fields = log.Fields{}
	}
	fields["component"] = "audit"
	fields["action"] = action
	if _, ok := fields["channel"]; !ok {
		if channel := strings.TrimSpace(c.Param("channel")); channel != "" {
			fields["channel"] = channel
		}
	}
	fields["remote_ip"] = c.ClientIP()
	if ua := c.Request.UserAgent(); ua != "" {
		fields["user_agent"] = ua
	}
	log.WithFields(fields).Info("management audit")
}

// Channel and template keys
func channelKey(ch string) string {
	ch = strings.ToLower(strings.TrimSpace(ch))
	if ch == "gemini" {
		return "model_registry_gemini"
	}
	return "model_registry_openai"
}

func groupKey(ch string) string {
	ch = strings.ToLower(strings.TrimSpace(ch))
	if ch == "gemini" {
		return "model_groups_gemini"
	}
	return "model_groups_openai"
}

func withChannel(c *gin.Context, ch string) *gin.Context {
	c.Params = append(c.Params, gin.Param{Key: "channel", Value: ch})
	return c
}

func templateKey(ch string) string {
	ch = strings.ToLower(strings.TrimSpace(ch))
	if ch == "gemini" {
		return "model_template_gemini"
	}
	return "model_template_openai"
}

// Misc helpers
func cloneResultMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func detectModelFamily(base string) string {
	lower := strings.ToLower(base)
	switch {
	case strings.Contains(lower, "pro"):
		return "pro"
	case strings.Contains(lower, "flash"):
		return "flash"
	case strings.Contains(lower, "nano"):
		return "nano"
	default:
		return "unknown"
	}
}

// Exported wrappers to share helpers with legacy server routes.
func IsNotSupported(err error) bool {
	return isNotSupported(err)
}

func RespondNotSupported(c *gin.Context) {
	respondNotSupported(c)
}
