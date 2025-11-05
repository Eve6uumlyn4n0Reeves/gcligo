package server

import (
	"gcli2api-go/internal/config"
	"github.com/gin-gonic/gin"
)

// registerAssemblyRoutes mounts the external routing assembly endpoints under the given management group.
func registerAssemblyRoutes(mg *gin.RouterGroup, cfg *config.Config, deps Dependencies) {
	registerAssemblyDashboardRoutes(mg, cfg, deps)
	registerAssemblyResourceRoutes(mg, cfg, deps)
	registerAssemblyPlanRoutes(mg, cfg, deps)
	registerAssemblyRoutingStateRoutes(mg, cfg, deps)
	registerAssemblyDryRunRoutes(mg, cfg, deps)
}
