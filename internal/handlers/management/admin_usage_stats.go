package management

import (
	"net/http"
	"sort"

	"gcli2api-go/internal/usage"
	"github.com/gin-gonic/gin"
)

// usageTracker is set by SetUsageTracker
var usageTracker *usage.Tracker

// SetUsageTracker sets the global usage tracker for management API
func SetUsageTracker(tracker *usage.Tracker) {
	usageTracker = tracker
}

// GetUsageStats returns overall usage statistics
// GET /api/management/usage
func (h *AdminAPIHandler) GetUsageStats(c *gin.Context) {
	if !h.isAdminRequest(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if usageTracker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "usage tracker not available",
		})
		return
	}

	stats := usageTracker.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"total_requests": stats.TotalRequests,
		"success_count":  stats.SuccessCount,
		"failure_count":  stats.FailureCount,
		"total_tokens":   stats.TotalTokens,
		"credentials":    len(stats.Credentials),
		"daily_stats":    len(stats.DailyStats),
		"hourly_stats":   len(stats.HourlyStats),
		"apis":           len(stats.APIs),
	})
}

// GetCredentialUsageStats returns usage statistics for all credentials
// GET /api/management/usage/credentials
func (h *AdminAPIHandler) GetCredentialUsageStats(c *gin.Context) {
	if !h.isAdminRequest(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if usageTracker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "usage tracker not available",
		})
		return
	}

	stats := usageTracker.GetStats()
	
	// Convert to array and sort by total calls
	credentials := make([]*usage.CredentialUsage, 0, len(stats.Credentials))
	for _, cred := range stats.Credentials {
		credentials = append(credentials, cred)
	}
	
	sort.Slice(credentials, func(i, j int) bool {
		return credentials[i].TotalCalls > credentials[j].TotalCalls
	})

	c.JSON(http.StatusOK, gin.H{
		"credentials": credentials,
		"total":       len(credentials),
	})
}

// GetCredentialUsageDetail returns detailed usage statistics for a specific credential
// GET /api/management/usage/credentials/:id
func (h *AdminAPIHandler) GetCredentialUsageDetail(c *gin.Context) {
	if !h.isAdminRequest(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if usageTracker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "usage tracker not available",
		})
		return
	}

	credentialID := c.Param("id")
	if credentialID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "credential ID required"})
		return
	}

	credUsage := usageTracker.GetCredentialStats(credentialID)
	if credUsage == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		return
	}

	c.JSON(http.StatusOK, credUsage)
}

// GetDailyUsageStats returns daily usage statistics
// GET /api/management/usage/daily
func (h *AdminAPIHandler) GetDailyUsageStats(c *gin.Context) {
	if !h.isAdminRequest(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if usageTracker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "usage tracker not available",
		})
		return
	}

	stats := usageTracker.GetStats()
	
	// Convert to array and sort by date
	dailyStats := make([]*usage.DailyStats, 0, len(stats.DailyStats))
	for _, daily := range stats.DailyStats {
		dailyStats = append(dailyStats, daily)
	}
	
	sort.Slice(dailyStats, func(i, j int) bool {
		return dailyStats[i].Date > dailyStats[j].Date
	})

	c.JSON(http.StatusOK, gin.H{
		"daily_stats": dailyStats,
		"total":       len(dailyStats),
	})
}

// GetHourlyUsageStats returns hourly usage statistics
// GET /api/management/usage/hourly
func (h *AdminAPIHandler) GetHourlyUsageStats(c *gin.Context) {
	if !h.isAdminRequest(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if usageTracker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "usage tracker not available",
		})
		return
	}

	stats := usageTracker.GetStats()
	
	// Convert to array and sort by hour
	hourlyStats := make([]*usage.HourlyStats, 0, len(stats.HourlyStats))
	for _, hourly := range stats.HourlyStats {
		hourlyStats = append(hourlyStats, hourly)
	}
	
	sort.Slice(hourlyStats, func(i, j int) bool {
		return hourlyStats[i].Hour < hourlyStats[j].Hour
	})

	c.JSON(http.StatusOK, gin.H{
		"hourly_stats": hourlyStats,
		"total":        len(hourlyStats),
	})
}

// GetAPIUsageStats returns per-API usage statistics
// GET /api/management/usage/apis
func (h *AdminAPIHandler) GetAPIUsageStats(c *gin.Context) {
	if !h.isAdminRequest(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if usageTracker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "usage tracker not available",
		})
		return
	}

	stats := usageTracker.GetStats()
	
	// Convert to array
	apiStats := make([]*usage.APIStats, 0, len(stats.APIs))
	for _, api := range stats.APIs {
		apiStats = append(apiStats, api)
	}
	
	sort.Slice(apiStats, func(i, j int) bool {
		return apiStats[i].TotalRequests > apiStats[j].TotalRequests
	})

	c.JSON(http.StatusOK, gin.H{
		"apis":  apiStats,
		"total": len(apiStats),
	})
}

// GetModelUsageStats returns per-model usage statistics across all APIs
// GET /api/management/usage/models
func (h *AdminAPIHandler) GetModelUsageStats(c *gin.Context) {
	if !h.isAdminRequest(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if usageTracker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "usage tracker not available",
		})
		return
	}

	stats := usageTracker.GetStats()
	
	// Aggregate model stats across all APIs
	modelMap := make(map[string]*usage.ModelStats)
	for _, api := range stats.APIs {
		for modelName, modelStats := range api.Models {
			if existing, ok := modelMap[modelName]; ok {
				existing.Calls += modelStats.Calls
				existing.Tokens += modelStats.Tokens
				existing.InputTokens += modelStats.InputTokens
				existing.OutputTokens += modelStats.OutputTokens
				existing.ReasoningTokens += modelStats.ReasoningTokens
				existing.CachedTokens += modelStats.CachedTokens
				if modelStats.LastUsed.After(existing.LastUsed) {
					existing.LastUsed = modelStats.LastUsed
				}
			} else {
				modelMap[modelName] = &usage.ModelStats{
					ModelName:       modelStats.ModelName,
					Calls:           modelStats.Calls,
					Tokens:          modelStats.Tokens,
					InputTokens:     modelStats.InputTokens,
					OutputTokens:    modelStats.OutputTokens,
					ReasoningTokens: modelStats.ReasoningTokens,
					CachedTokens:    modelStats.CachedTokens,
					LastUsed:        modelStats.LastUsed,
				}
			}
		}
	}
	
	// Convert to array and sort by calls
	modelStats := make([]*usage.ModelStats, 0, len(modelMap))
	for _, model := range modelMap {
		modelStats = append(modelStats, model)
	}
	
	sort.Slice(modelStats, func(i, j int) bool {
		return modelStats[i].Calls > modelStats[j].Calls
	})

	c.JSON(http.StatusOK, gin.H{
		"models": modelStats,
		"total":  len(modelStats),
	})
}

