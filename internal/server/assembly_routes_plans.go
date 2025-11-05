package server

import (
	"net/http"
	"strings"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/handlers/management"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func registerAssemblyPlanRoutes(mg *gin.RouterGroup, cfg *config.Config, deps Dependencies) {
	mg.GET("/assembly/plans", func(c *gin.Context) {
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		plans, err := svc.ListPlans(c.Request.Context())
		if err != nil {
			if management.IsNotSupported(err) {
				management.RespondNotSupported(c)
				return
			}
			respondError(c, http.StatusInternalServerError, err.Error(), nil)
			return
		}
		items := make([]gin.H, 0, len(plans))
		for _, p := range plans {
			items = append(items, gin.H(p))
		}
		c.JSON(http.StatusOK, gin.H{"plans": items})
	})

	mg.GET("/assembly/plans/:name", func(c *gin.Context) {
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		v, err := svc.GetPlan(c.Request.Context(), c.Param("name"))
		if err != nil {
			if management.IsNotSupported(err) {
				management.RespondNotSupported(c)
				return
			}
			respondError(c, http.StatusNotFound, "plan not found", nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"plan": v})
	})

	mg.POST("/assembly/plans", func(c *gin.Context) {
		var req assemblySavePlanRequest
		if !bindJSON(c, &req) {
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			respondError(c, http.StatusBadRequest, "name is required", nil)
			return
		}
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		plan, err := svc.SavePlan(c.Request.Context(), req.Name, req.Include)
		if err != nil {
			if management.IsNotSupported(err) {
				management.RespondNotSupported(c)
				return
			}
			respondError(c, http.StatusInternalServerError, err.Error(), nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "plan saved", "plan": plan})
	})

	mg.PUT("/assembly/plans/:name/apply", func(c *gin.Context) {
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		audit := buildAssemblyAudit(c, cfg)
		name := c.Param("name")
		if err := svc.ApplyPlan(c.Request.Context(), name); err != nil {
			svc.RecordOperation("plan_apply", "error", audit)
			if management.IsNotSupported(err) {
				management.RespondNotSupported(c)
				return
			}
			logAssemblyEvent(c, log.Fields{
				"component": "assembly",
				"action":    "plan_apply",
				"plan":      name,
				"actor":     audit.ActorLabel,
				"actor_id":  audit.ActorID,
				"reason":    audit.Reason,
				"status":    "error",
			}).WithError(err).Warn("apply assembly plan failed")
			respondError(c, http.StatusInternalServerError, err.Error(), nil)
			return
		}
		svc.RecordOperation("plan_apply", "success", audit)
		logAssemblyEvent(c, log.Fields{
			"component": "assembly",
			"action":    "plan_apply",
			"plan":      name,
			"actor":     audit.ActorLabel,
			"actor_id":  audit.ActorID,
			"reason":    audit.Reason,
			"status":    "success",
		}).Info("assembly plan applied")
		c.JSON(http.StatusOK, gin.H{"message": "applied"})
	})

	mg.PUT("/assembly/plans/:name/rollback", func(c *gin.Context) {
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		audit := buildAssemblyAudit(c, cfg)
		name := c.Param("name")
		if err := svc.RollbackPlan(c.Request.Context(), name); err != nil {
			svc.RecordOperation("plan_rollback", "error", audit)
			if management.IsNotSupported(err) {
				management.RespondNotSupported(c)
				return
			}
			logAssemblyEvent(c, log.Fields{
				"component": "assembly",
				"action":    "plan_rollback",
				"plan":      name,
				"actor":     audit.ActorLabel,
				"actor_id":  audit.ActorID,
				"reason":    audit.Reason,
				"status":    "error",
			}).WithError(err).Warn("rollback assembly plan failed")
			respondError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		svc.RecordOperation("plan_rollback", "success", audit)
		logAssemblyEvent(c, log.Fields{
			"component": "assembly",
			"action":    "plan_rollback",
			"plan":      name,
			"actor":     audit.ActorLabel,
			"actor_id":  audit.ActorID,
			"reason":    audit.Reason,
			"status":    "success",
		}).Info("assembly plan rolled back")
		c.JSON(http.StatusOK, gin.H{"message": "rolled back"})
	})

	mg.DELETE("/assembly/plans/:name", func(c *gin.Context) {
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		if err := svc.DeletePlan(c.Request.Context(), c.Param("name")); err != nil {
			if management.IsNotSupported(err) {
				management.RespondNotSupported(c)
				return
			}
			respondError(c, http.StatusNotFound, "plan not found", nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	})
}
