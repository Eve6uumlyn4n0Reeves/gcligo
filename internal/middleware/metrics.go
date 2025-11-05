package middleware

import (
	"fmt"
	"time"

	"gcli2api-go/internal/monitoring"
	"github.com/gin-gonic/gin"
)

func statusClass(code int) string {
	if code <= 0 {
		return "error"
	}
	c := code / 100
	return fmt.Sprintf("%dxx", c)
}

// Metrics is an HTTP middleware to track per-route counters and latency histogram
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		monitoring.HTTPInFlight.Inc()
		c.Next()
		monitoring.HTTPInFlight.Dec()

		durSec := time.Since(start).Seconds()
		server, _ := c.Get("server_label")
		serverStr, _ := server.(string)
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		sc := statusClass(c.Writer.Status())

		monitoring.HTTPRequestsTotal.WithLabelValues(serverStr, c.Request.Method, path, sc).Inc()
		monitoring.HTTPRequestDuration.WithLabelValues(serverStr, c.Request.Method, path, sc).Observe(durSec)
	}
}

// SetRateLimitKeyGauge sets the current per-key limiter count.
func SetRateLimitKeyGauge(n int) {
	monitoring.RateLimitKeysGauge.Set(float64(n))
}

// RecordRateLimitSweep increments the sweep counter for TTL cache.
func RecordRateLimitSweep() {
	monitoring.RateLimitSweepsTotal.Inc()
}
