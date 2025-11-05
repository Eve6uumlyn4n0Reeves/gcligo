package middleware

import (
	"time"

	"gcli2api-go/internal/logging"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// RequestLogger logs HTTP requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		// Attempt to include caller API key if middleware set it
		apiKeyVal, _ := c.Get("api_key")
		modelVal, _ := c.Get("model")
		baseVal, _ := c.Get("base_model")
		extras := log.Fields{
			"status":     status,
			"latency_ms": logging.DurationMS(latency),
			"user_agent": c.Request.UserAgent(),
			"method":     method,
			"path":       path,
			"api_key":    apiKeyVal,
			"model":      modelVal,
			"base":       baseVal,
		}
		logging.WithReq(c, extras).Info("http_request")
	}
}
