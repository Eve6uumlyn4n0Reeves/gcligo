package server

import (
	"net/http"
	"strings"

	"gcli2api-go/internal/config"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func registerAssemblyRoutingStateRoutes(mg *gin.RouterGroup, cfg *config.Config, deps Dependencies) {
	mg.POST("/routing/persist", func(c *gin.Context) {
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		audit := buildAssemblyAudit(c, cfg)
		n, err := svc.SaveRoutingState(c.Request.Context())
		if err != nil {
			svc.RecordOperation("routing_persist", "error", audit)
			respondError(c, http.StatusNotImplemented, err.Error(), nil)
			return
		}
		svc.RecordOperation("routing_persist", "success", audit)
		logAssemblyEvent(c, log.Fields{
			"component": "assembly",
			"action":    "routing_persist",
			"count":     n,
			"actor":     audit.ActorLabel,
			"actor_id":  audit.ActorID,
			"reason":    audit.Reason,
			"status":    "success",
		}).Info("routing state persisted")
		c.JSON(http.StatusOK, gin.H{"message": "persisted", "count": n})
	})

	mg.POST("/routing/restore", func(c *gin.Context) {
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		audit := buildAssemblyAudit(c, cfg)
		n, err := svc.RestoreRoutingState(c.Request.Context())
		if err != nil {
			svc.RecordOperation("routing_restore", "error", audit)
			respondError(c, http.StatusNotImplemented, err.Error(), nil)
			return
		}
		svc.RecordOperation("routing_restore", "success", audit)
		logAssemblyEvent(c, log.Fields{
			"component": "assembly",
			"action":    "routing_restore",
			"applied":   n,
			"actor":     audit.ActorLabel,
			"actor_id":  audit.ActorID,
			"reason":    audit.Reason,
			"status":    "success",
		}).Info("routing state restored")
		c.JSON(http.StatusOK, gin.H{"message": "restored", "applied": n})
	})

	mg.POST("/assembly/cooldowns/clear", func(c *gin.Context) {
		st := deps.RoutingStrategy
		if st == nil {
			respondError(c, http.StatusNotImplemented, "routing strategy unavailable", nil)
			return
		}
		var req assemblyCooldownClearRequest
		if !bindJSON(c, &req) {
			return
		}
		audit := buildAssemblyAudit(c, cfg)
		_, current := st.Snapshot()
		cleared := make([]string, 0)
		skipped := make([]string, 0)
		if req.All {
			for _, cd := range current {
				if st.ClearCooldown(cd.CredID) {
					cleared = append(cleared, cd.CredID)
				}
			}
		} else {
			if len(req.Credentials) == 0 {
				respondError(c, http.StatusBadRequest, "credentials or all required", nil)
				return
			}
			seen := make(map[string]struct{})
			for _, id := range req.Credentials {
				trimmed := strings.TrimSpace(id)
				if trimmed == "" {
					continue
				}
				if _, ok := seen[trimmed]; ok {
					continue
				}
				seen[trimmed] = struct{}{}
				if st.ClearCooldown(trimmed) {
					cleared = append(cleared, trimmed)
				} else {
					skipped = append(skipped, trimmed)
				}
			}
		}
		_, remaining := st.Snapshot()
		status := "success"
		if len(cleared) == 0 && !req.All && len(req.Credentials) > 0 {
			status = "noop"
		}
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		svc.RecordOperation("routing_clear_cooldowns", status, audit)
		logAssemblyEvent(c, log.Fields{
			"component":    "assembly",
			"action":       "routing_clear_cooldowns",
			"cleared":      len(cleared),
			"skipped":      len(skipped),
			"actor":        audit.ActorLabel,
			"actor_id":     audit.ActorID,
			"reason":       audit.Reason,
			"status":       status,
			"total_before": len(current),
			"total_after":  len(remaining),
		}).Info("routing cooldowns cleared")
		c.JSON(http.StatusOK, gin.H{
			"cleared":      cleared,
			"skipped":      skipped,
			"total_before": len(current),
			"total_after":  len(remaining),
		})
	})
}
