package middleware

import (
	"net/http"
	"strings"

	apperrors "gcli2api-go/internal/errors"
	"gcli2api-go/internal/httpformat"
	"github.com/gin-gonic/gin"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	// RequiredKey is the expected API key (if empty, auth is disabled)
	RequiredKey string
	// AllowMultipleSources enables checking multiple header/query locations
	AllowMultipleSources bool
	// CustomValidator is an optional function for custom validation logic
	CustomValidator func(key string) bool
	// AcceptCookieName, if non-empty, allows extracting the token from a cookie
	// with this name (e.g., for same-origin WebSocket/Admin UI session tokens).
	// This is evaluated only when Authorization header is empty.
	AcceptCookieName string
}

// UnifiedAuth provides flexible authentication middleware that supports:
// - Authorization: Bearer <token>
// - x-goog-api-key: <token>
// - x-api-key: <token>
// - Query parameter: ?key=<token>
func UnifiedAuth(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth if no key is configured
		if cfg.RequiredKey == "" && cfg.CustomValidator == nil {
			c.Next()
			return
		}

		var providedKey string

		// Try Authorization header (Bearer token)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				providedKey = strings.TrimSpace(authHeader[7:])
			} else {
				providedKey = authHeader
			}
		}

		// Try cookie (session token for admin UI) when configured
		if providedKey == "" && strings.TrimSpace(cfg.AcceptCookieName) != "" {
			if v, err := c.Cookie(strings.TrimSpace(cfg.AcceptCookieName)); err == nil && v != "" {
				providedKey = v
			}
		}

		// Try x-goog-api-key header (Gemini style)
		if providedKey == "" || cfg.AllowMultipleSources {
			if key := c.GetHeader("x-goog-api-key"); key != "" {
				providedKey = key
			}
		}

		// Try x-api-key header (Claude/Anthropic style)
		if providedKey == "" || cfg.AllowMultipleSources {
			if key := c.GetHeader("x-api-key"); key != "" {
				providedKey = key
			}
		}

		// Try query parameter
		if providedKey == "" || cfg.AllowMultipleSources {
			if key := c.Query("key"); key != "" {
				providedKey = key
			}
		}

		// Validate the key
		if providedKey == "" {
			respondUnauthorized(c, "API key not provided")
			return
		}

		// Use custom validator if provided
		if cfg.CustomValidator != nil {
			if !cfg.CustomValidator(providedKey) {
				respondUnauthorized(c, "Invalid API key")
				return
			}
			c.Set("api_key", providedKey)
			c.Next()
			return
		}

		// Standard validation
		if cfg.RequiredKey != "" && providedKey != cfg.RequiredKey {
			respondUnauthorized(c, "Invalid API key")
			return
		}

		c.Set("api_key", providedKey)
		c.Next()
	}
}

func respondUnauthorized(c *gin.Context, message string) {
	err := apperrors.New(
		http.StatusUnauthorized,
		"invalid_api_key",
		"invalid_request_error",
		message,
	)
	format := httpformat.DetectFromContext(c)
	payload, marshalErr := err.ToJSON(format)
	if marshalErr != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": err.Message,
				"type":    err.Type,
				"code":    err.Code,
			},
		})
		c.Abort()
		return
	}
	c.Data(http.StatusUnauthorized, "application/json", payload)
	c.Abort()
}

// MultiKeyAuth validates against a list of allowed keys
func MultiKeyAuth(allowedKeys []string) gin.HandlerFunc {
	keySet := make(map[string]bool)
	for _, k := range allowedKeys {
		if k != "" {
			keySet[k] = true
		}
	}

	if len(keySet) == 0 {
		// No keys configured, allow all
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return UnifiedAuth(AuthConfig{
		AllowMultipleSources: true,
		CustomValidator: func(key string) bool {
			return keySet[key]
		},
	})
}
