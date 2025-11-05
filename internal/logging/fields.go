package logging

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"time"
)

// WithReq builds a log entry enriched with common HTTP request fields.
// Fields:
// - request_id: X-Request-ID or generated in middleware
// - method, path, ip
// Any extras passed in will be merged (extras take precedence on key conflicts).
func WithReq(c *gin.Context, extras log.Fields) *log.Entry {
	if c == nil {
		return log.WithFields(extras)
	}
	path := c.FullPath()
	if path == "" && c.Request != nil && c.Request.URL != nil {
		path = c.Request.URL.Path
	}
	rid, _ := c.Get("request_id")
	fields := log.Fields{
		"request_id": rid,
		"method":     c.Request.Method,
		"path":       path,
		"ip":         c.ClientIP(),
	}
	for k, v := range extras {
		fields[k] = v
	}
	return log.WithFields(fields)
}

// DurationMS converts a duration to integer milliseconds for logging.
func DurationMS(d time.Duration) int64 { return d.Milliseconds() }
