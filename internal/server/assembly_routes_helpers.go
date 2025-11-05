package server

import (
	"strings"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/logging"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func buildAssemblyAudit(c *gin.Context, cfg *config.Config) AssemblyAudit {
	reason := strings.TrimSpace(c.GetHeader("X-Change-Reason"))
	if reason == "" {
		reason = strings.TrimSpace(c.Query("reason"))
	}
	actorLabel, actorID := resolveAuditActor(c, cfg)
	return AssemblyAudit{
		ActorLabel: actorLabel,
		ActorID:    actorID,
		Reason:     reason,
	}
}

func resolveAuditActor(c *gin.Context, cfg *config.Config) (string, string) {
	if c == nil {
		return "unknown", ""
	}
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token := strings.TrimSpace(auth[7:])
		if token != "" {
			if config.CheckManagementKey(cfg, token) {
				return "management_key", maskToken(token)
			}
			return "bearer", maskToken(token)
		}
	}
	if cookie, err := c.Cookie("mgmt_session"); err == nil {
		token := strings.TrimSpace(cookie)
		if token != "" {
			return "mgmt_session", maskToken(token)
		}
	}
	return "unknown", ""
}

func maskToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 6 {
		return token
	}
	return token[:3] + "..." + token[len(token)-3:]
}

func logAssemblyEvent(c *gin.Context, fields log.Fields) *log.Entry {
	return logging.WithReq(c, fields)
}
