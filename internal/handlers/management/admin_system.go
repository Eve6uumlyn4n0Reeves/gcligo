package management

import (
	"net/http"
	"runtime"
	"time"

	"gcli2api-go/internal/stats"
	"github.com/gin-gonic/gin"
)

// GetSystemInfo returns system information
func (h *AdminAPIHandler) GetSystemInfo(c *gin.Context) {
	uptime := time.Since(h.startTime).Seconds()
	c.JSON(http.StatusOK, gin.H{
		"version":    "2.0.0",
		"go_version": runtime.Version(),
		"uptime":     uptime,
		"timestamp":  time.Now().Unix(),
	})
}

// GetHealth returns health status
func (h *AdminAPIHandler) GetHealth(c *gin.Context) {
	healthy := true
	checks := make(map[string]interface{})

	// Check storage
	if h.storage != nil {
		if err := h.storage.Health(c.Request.Context()); err != nil {
			healthy = false
			checks["storage"] = gin.H{"status": "unhealthy", "error": err.Error()}
		} else {
			checks["storage"] = gin.H{"status": "healthy"}
		}
	}

	// Check credentials
	creds := h.credMgr.GetAllCredentials()
	healthyCreds := 0
	autoBanned := 0
	maxFailureWeight := 0.0
	for _, cred := range creds {
		if cred.IsHealthy() {
			healthyCreds++
		}
		if cred.AutoBanned {
			autoBanned++
		}
		if cred.FailureWeight > maxFailureWeight {
			maxFailureWeight = cred.FailureWeight
		}
	}
	credStatus := gin.H{
		"total":              len(creds),
		"healthy":            healthyCreds,
		"auto_banned":        autoBanned,
		"max_failure_weight": maxFailureWeight,
	}
	if len(creds) > 0 && healthyCreds == 0 {
		healthy = false
		credStatus["status"] = "unhealthy"
	} else {
		credStatus["status"] = "ok"
	}
	checks["credentials"] = credStatus

	// Auto-probe scheduler insight
	autoProbe := gin.H{}
	if h.cfg != nil {
		autoProbe["enabled"] = h.cfg.AutoProbeEnabled
		autoProbe["model"] = h.cfg.AutoProbeModel
		autoProbe["timeout_sec"] = h.cfg.AutoProbeTimeoutSec
		autoProbe["next_schedule"] = h.nextAutoProbeTime(time.Now().UTC())
	}
	h.autoProbeMu.Lock()
	if !h.autoProbeLastRun.IsZero() {
		autoProbe["last_run"] = h.autoProbeLastRun
	}
	h.autoProbeMu.Unlock()
	h.probeHistoryMu.Lock()
	if len(h.probeHistory) > 0 {
		last := h.probeHistory[0]
		autoProbe["last_probe"] = gin.H{
			"timestamp": last.Timestamp,
			"source":    last.Source,
			"success":   last.Success,
			"total":     last.Total,
		}
	}
	h.probeHistoryMu.Unlock()
	checks["auto_probe"] = autoProbe

	// Upstream discovery snapshot
	if h.modelFinder != nil {
		if bases, expires, ok := h.modelFinder.Snapshot(); ok {
			entry := gin.H{
				"status":     "cached",
				"count":      len(bases),
				"expires_at": expires,
			}
			if len(bases) > 0 {
				sample := bases
				if len(sample) > 5 {
					sample = sample[:5]
				}
				entry["sample"] = sample
			}
			checks["upstream_discovery"] = entry
		} else {
			checks["upstream_discovery"] = gin.H{"status": "stale"}
		}
	}

	// Runtime info
	checks["runtime"] = gin.H{
		"uptime_sec": int(time.Since(h.startTime).Seconds()),
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"healthy": healthy,
		"checks":  checks,
	})
}

// GetMetrics returns detailed metrics
func (h *AdminAPIHandler) GetMetrics(c *gin.Context) {
	if h.metrics == nil {
		c.JSON(http.StatusOK, gin.H{"metrics": gin.H{}})
		return
	}
	snapshot := h.metrics.GetSnapshot()
	c.JSON(http.StatusOK, snapshot)
}

// GetUsage returns usage statistics
func (h *AdminAPIHandler) GetUsage(c *gin.Context) {
	if h.usageStats == nil {
		respondError(c, http.StatusNotImplemented, "usage tracking not configured")
		return
	}
	apiKey := c.Query("api_key")
	if apiKey != "" {
		usage, err := h.usageStats.GetUsage(c.Request.Context(), apiKey)
		if err != nil {
			if isNotSupported(err) {
				respondNotSupported(c)
				return
			}
			respondError(c, http.StatusNotFound, "API key not found")
			return
		}
		c.JSON(http.StatusOK, usage)
		return
	}
	allUsage, err := h.usageStats.GetAllUsage(c.Request.Context())
	if err != nil {
		if isNotSupported(err) {
			respondNotSupported(c)
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	apiKeys := make(map[string]*stats.UsageRecord)
	models := make(map[string]*stats.UsageRecord)
	var total *stats.UsageRecord

	for key, record := range allUsage {
		if kind, value, ok := stats.ClassifyAggregateKey(key); ok {
			switch kind {
			case stats.AggregateKindTotal:
				total = record
			case stats.AggregateKindModel:
				if value != "" {
					models[value] = record
				}
			}
			continue
		}
		apiKeys[key] = record
	}

	response := gin.H{"api_keys": apiKeys}
	aggregates := gin.H{}
	if total != nil {
		aggregates["total"] = total
	}
	if len(models) > 0 {
		aggregates["models"] = models
	}
	if len(aggregates) > 0 {
		response["aggregates"] = aggregates
	}
	c.JSON(http.StatusOK, response)
}
