package server

import (
	"net/http"
	"strings"

	"gcli2api-go/internal/config"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// ManagementAuthLevel represents the authentication level for management endpoints
type ManagementAuthLevel int

const (
	// AuthLevelNone means no authentication required (should never be used)
	AuthLevelNone ManagementAuthLevel = iota
	// AuthLevelReadOnly means read-only access (GET, HEAD, OPTIONS)
	AuthLevelReadOnly
	// AuthLevelAdmin means full admin access (all methods)
	AuthLevelAdmin
)

// ManagementAuthConfig holds the configuration for management authentication
type ManagementAuthConfig struct {
	AdminKey      string
	AdminKeyHash  string
	ReadOnlyKey   string
	AllowReadOnly bool
}

// NewManagementAuthConfig creates a new management auth config from the main config
func NewManagementAuthConfig(cfg *config.Config) *ManagementAuthConfig {
	return &ManagementAuthConfig{
		AdminKey:      cfg.Security.ManagementKey,
		AdminKeyHash:  cfg.Security.ManagementKeyHash,
		ReadOnlyKey:   cfg.Security.ManagementReadOnlyKey,
		AllowReadOnly: cfg.Security.ManagementReadOnly,
	}
}

// ValidateToken validates a token and returns the authentication level
func (mac *ManagementAuthConfig) ValidateToken(token string) ManagementAuthLevel {
	if token == "" {
		return AuthLevelNone
	}

	// Check admin key first
	if mac.AdminKey != "" && token == mac.AdminKey {
		return AuthLevelAdmin
	}

	// Check admin key hash
	if mac.AdminKeyHash != "" {
		if err := config.CheckManagementKey(token, mac.AdminKey, mac.AdminKeyHash); err == nil {
			return AuthLevelAdmin
		}
	}

	// Check read-only key
	if mac.AllowReadOnly && mac.ReadOnlyKey != "" && token == mac.ReadOnlyKey {
		return AuthLevelReadOnly
	}

	return AuthLevelNone
}

// ExtractToken extracts the authentication token from the request
func ExtractToken(c *gin.Context) string {
	// Try Authorization header (Bearer token)
	if auth := c.GetHeader("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}

	// Try x-api-key header
	if apiKey := c.GetHeader("x-api-key"); apiKey != "" {
		return apiKey
	}

	// Try x-goog-api-key header
	if googKey := c.GetHeader("x-goog-api-key"); googKey != "" {
		return googKey
	}

	// Try query parameter
	if queryKey := c.Query("key"); queryKey != "" {
		return queryKey
	}

	return ""
}

// ManagementAuthMiddleware creates a middleware that enforces management authentication
func ManagementAuthMiddleware(authConfig *ManagementAuthConfig, requiredLevel ManagementAuthLevel) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ExtractToken(c)
		level := authConfig.ValidateToken(token)

		// Check if the level is sufficient
		if level < requiredLevel {
			log.WithFields(log.Fields{
				"path":          c.Request.URL.Path,
				"method":        c.Request.Method,
				"required":      requiredLevel,
				"actual":        level,
				"remote_addr":   c.ClientIP(),
			}).Warn("Management authentication failed: insufficient privileges")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized: insufficient privileges for this operation",
			})
			c.Abort()
			return
		}

		// Store the auth level in context for later use
		c.Set("auth_level", level)
		c.Next()
	}
}

// RequireReadOnly creates a middleware that requires at least read-only access
func RequireReadOnly(authConfig *ManagementAuthConfig) gin.HandlerFunc {
	return ManagementAuthMiddleware(authConfig, AuthLevelReadOnly)
}

// RequireAdmin creates a middleware that requires admin access
func RequireAdmin(authConfig *ManagementAuthConfig) gin.HandlerFunc {
	return ManagementAuthMiddleware(authConfig, AuthLevelAdmin)
}

// GetAuthLevel retrieves the authentication level from the context
func GetAuthLevel(c *gin.Context) ManagementAuthLevel {
	if level, exists := c.Get("auth_level"); exists {
		if authLevel, ok := level.(ManagementAuthLevel); ok {
			return authLevel
		}
	}
	return AuthLevelNone
}

// IsAdmin checks if the current request has admin privileges
func IsAdmin(c *gin.Context) bool {
	return GetAuthLevel(c) == AuthLevelAdmin
}

// IsReadOnly checks if the current request has at least read-only privileges
func IsReadOnly(c *gin.Context) bool {
	level := GetAuthLevel(c)
	return level == AuthLevelReadOnly || level == AuthLevelAdmin
}

