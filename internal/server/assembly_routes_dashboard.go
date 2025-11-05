package server

import (
	"net/http"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/models"
	"github.com/gin-gonic/gin"
)

func registerAssemblyDashboardRoutes(mg *gin.RouterGroup, cfg *config.Config, deps Dependencies) {
	mg.GET("/assembly/dashboard", func(c *gin.Context) {
		routesMeta := buildRoutesJSON(cfg, deps.Storage)
		oaModels := models.ExposedModelIDsByChannel(cfg, deps.Storage, "openai")
		gmModels := models.ExposedModelIDsByChannel(cfg, deps.Storage, "gemini")
		creds := deps.CredentialManager.GetAllCredentials()
		totalCreds, healthyCreds, disabledCreds, autobanCreds := 0, 0, 0, 0
		var totalReq, totalSucc int64
		recent := make([]gin.H, 0)
		for _, cr := range creds {
			if cr == nil {
				continue
			}
			totalCreds++
			if cr.Disabled {
				disabledCreds++
			}
			if cr.AutoBanned {
				autobanCreds++
			}
			if cr.IsHealthy() {
				healthyCreds++
			}
			totalReq += cr.TotalRequests
			totalSucc += cr.SuccessCount
			if len(recent) < 50 {
				lastSuccess := int64(0)
				if !cr.LastSuccess.IsZero() {
					lastSuccess = cr.LastSuccess.Unix()
				}
				lastFailure := int64(0)
				if !cr.LastFailure.IsZero() {
					lastFailure = cr.LastFailure.Unix()
				}
				recent = append(recent, gin.H{
					"id":           cr.ID,
					"disabled":     cr.Disabled,
					"auto_banned":  cr.AutoBanned,
					"last_success": lastSuccess,
					"last_failure": lastFailure,
				})
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"overview": gin.H{
				"endpoints":              routesMeta,
				"models_total":           len(append(oaModels, gmModels...)),
				"credentials_total":      totalCreds,
				"credentials_healthy":    healthyCreds,
				"credentials_disabled":   disabledCreds,
				"credentials_autobanned": autobanCreds,
				"success_rate": func() float64 {
					if totalReq == 0 {
						return 0
					}
					return float64(totalSucc) / float64(totalReq)
				}(),
			},
			"models":      gin.H{"openai": gin.H{"ids": oaModels}, "gemini": gin.H{"ids": gmModels}},
			"credentials": gin.H{"recent": recent},
		})
	})
}

func registerAssemblyResourceRoutes(mg *gin.RouterGroup, cfg *config.Config, deps Dependencies) {
	mg.GET("/assembly/overview", func(c *gin.Context) {
		routesMeta := buildRoutesJSON(cfg, deps.Storage)
		openaiModels := models.ExposedModelIDsByChannel(cfg, deps.Storage, "openai")
		c.JSON(http.StatusOK, gin.H{
			"routes":              routesMeta,
			"openai_models_count": len(openaiModels),
		})
	})

	mg.GET("/assembly/routes-meta", func(c *gin.Context) {
		c.JSON(http.StatusOK, buildRoutesJSON(cfg, deps.Storage))
	})

	mg.GET("/assembly/models", func(c *gin.Context) {
		openaiModels := models.ExposedModelIDsByChannel(cfg, deps.Storage, "openai")
		geminiModels := models.ExposedModelIDsByChannel(cfg, deps.Storage, "gemini")
		c.JSON(http.StatusOK, gin.H{"openai": openaiModels, "gemini": geminiModels})
	})

	mg.GET("/assembly/credentials", func(c *gin.Context) {
		creds := deps.CredentialManager.GetAllCredentials()
		total, healthy, disabled, autoban := 0, 0, 0, 0
		recent := make([]gin.H, 0)
		for _, cr := range creds {
			if cr == nil {
				continue
			}
			total++
			if cr.Disabled {
				disabled++
			}
			if cr.AutoBanned {
				autoban++
			}
			if cr.IsHealthy() {
				healthy++
			}
			if len(recent) < 30 {
				lastSuccess := int64(0)
				if !cr.LastSuccess.IsZero() {
					lastSuccess = cr.LastSuccess.Unix()
				}
				lastFailure := int64(0)
				if !cr.LastFailure.IsZero() {
					lastFailure = cr.LastFailure.Unix()
				}
				recent = append(recent, gin.H{
					"id":           cr.ID,
					"disabled":     cr.Disabled,
					"auto_banned":  cr.AutoBanned,
					"last_success": lastSuccess,
					"last_failure": lastFailure,
				})
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"total":      total,
			"healthy":    healthy,
			"disabled":   disabled,
			"autobanned": autoban,
			"recent":     recent,
		})
	})

	mg.GET("/assembly/routing", func(c *gin.Context) {
		st := deps.RoutingStrategy
		if st == nil {
			respondError(c, http.StatusNotImplemented, "routing strategy unavailable", nil)
			return
		}
		stickyCount, cooldowns := st.Snapshot()
		c.JSON(http.StatusOK, gin.H{"sticky": stickyCount, "cooldowns": cooldowns})
	})

	mg.GET("/assembly/usage", func(c *gin.Context) {
		usageOverview := gin.H{}
		if deps.UsageStats != nil {
			if all, err := deps.UsageStats.GetAllUsage(c.Request.Context()); err == nil {
				usageOverview = gin.H{"keys": len(all)}
			}
		}
		c.JSON(http.StatusOK, gin.H{"overview": usageOverview})
	})
}
