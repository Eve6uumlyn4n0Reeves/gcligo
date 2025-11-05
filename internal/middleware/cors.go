package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS provides Cross-Origin Resource Sharing support
// Note: Management API routes (/api/management) deliberately skip CORS headers
// to avoid broadening cross-origin surface for admin endpoints.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// Skip CORS for admin/management APIs (served same-origin by design)
		if strings.Contains(path, "/api/management") {
			c.Next()
			return
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		// Credentials are not required for bearer-token style API calls
		// Avoid enabling credentials with wildcard origin
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "false")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, x-goog-api-key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
