package server

import (
	"net/http"

	"gcli2api-go/internal/config"
	"github.com/gin-gonic/gin"
)

func registerAssemblyDryRunRoutes(mg *gin.RouterGroup, cfg *config.Config, deps Dependencies) {
	mg.POST("/assembly/dry-run", func(c *gin.Context) {
		var req assemblyDryRunRequest
		if !bindJSON(c, &req) {
			return
		}
		if len(req.Plan) == 0 {
			respondError(c, http.StatusBadRequest, "plan required", nil)
			return
		}
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		sanitized := sanitizePlanPayload(req.Plan)
		diff, err := svc.DiffPlan(c.Request.Context(), sanitized)
		if err != nil {
			respondError(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"diff": diff, "plan": sanitized})
	})

	mg.GET("/assembly/plans/:name/dry-run/apply", func(c *gin.Context) {
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		diff, err := svc.DiffApply(c.Request.Context(), c.Param("name"))
		if err != nil {
			respondError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"diff": diff})
	})

	mg.GET("/assembly/plans/:name/dry-run/rollback", func(c *gin.Context) {
		svc := NewAssemblyService(cfg, deps.Storage, deps.EnhancedMetrics, deps.RoutingStrategy)
		diff, err := svc.DiffRollback(c.Request.Context(), c.Param("name"))
		if err != nil {
			respondError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"diff": diff})
	})
}
