package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation error [%s=%s]: %s", e.Field, e.Value, e.Message)
}

// ValidationResult holds the results of configuration validation
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationError
	Valid    bool
}

// AddError adds a validation error
func (r *ValidationResult) AddError(field, value, message string) {
	r.Errors = append(r.Errors, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
	r.Valid = false
}

// AddWarning adds a validation warning
func (r *ValidationResult) AddWarning(field, value, message string) {
	r.Warnings = append(r.Warnings, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

// Validate validates the configuration and returns validation results
func (c *Config) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate ports
	if err := validatePort(c.OpenAIPort); err != nil {
		result.AddError("openai_port", c.OpenAIPort, err.Error())
	}
	if c.GeminiPort != "" && c.GeminiPort != "0" {
		if err := validatePort(c.GeminiPort); err != nil {
			result.AddError("gemini_port", c.GeminiPort, err.Error())
		}
	}

	// Enforce single upstream provider
	up := strings.ToLower(strings.TrimSpace(c.UpstreamProvider))
	if up != "" && up != "gemini" && up != "code_assist" {
		result.AddError("upstream_provider", c.UpstreamProvider, "only 'gemini' is supported")
	}

	// Validate storage backend
	validBackends := []string{"file", "redis", "mongodb", "postgres", "git"}
	if !contains(validBackends, c.StorageBackend) {
		result.AddError("storage_backend", c.StorageBackend,
			fmt.Sprintf("must be one of: %s", strings.Join(validBackends, ", ")))
	}

	// Validate storage backend specific configuration
	switch c.StorageBackend {
	case "redis":
		if c.RedisAddr == "" {
			result.AddError("redis_addr", c.RedisAddr, "required when using redis backend")
		}
	case "mongodb":
		if c.MongoURI == "" {
			result.AddError("mongodb_uri", c.MongoURI, "required when using mongodb backend")
		}
	case "postgres":
		if c.PostgresDSN == "" {
			result.AddError("postgres_dsn", c.PostgresDSN, "required when using postgres backend")
		}
	case "file":
		if c.StorageBaseDir == "" {
			result.AddWarning("storage_base_dir", c.StorageBaseDir, "using default directory")
		}
	case "git":
		if c.GitRemoteURL == "" {
			result.AddError("git_remote_url", c.GitRemoteURL, "required when using git backend")
		}
		if c.GitBranch == "" {
			result.AddWarning("git_branch", c.GitBranch, "branch not specified, defaulting to 'main'")
		}
		if strings.TrimSpace(c.GitAuthorName) == "" || strings.TrimSpace(c.GitAuthorEmail) == "" {
			result.AddWarning("git_author", fmt.Sprintf("%s <%s>", c.GitAuthorName, c.GitAuthorEmail), "git author name/email recommended for commit history")
		}
		if c.StorageBaseDir == "" {
			result.AddWarning("storage_base_dir", c.StorageBaseDir, "using default clone directory for git backend")
		}
	}

	// Validate auth directory
	if c.AuthDir == "" {
		result.AddError("auth_dir", c.AuthDir, "authentication directory is required")
	}

	// Validate proxy URL if set
	if c.ProxyURL != "" {
		if _, err := url.Parse(c.ProxyURL); err != nil {
			result.AddError("proxy_url", c.ProxyURL, "invalid proxy URL format")
		}
	}

	// Validate OAuth configuration
	if c.OAuthClientID != "" && c.OAuthClientSecret == "" {
		result.AddError("oauth_client_secret", c.OAuthClientSecret,
			"oauth_client_secret required when oauth_client_id is set")
	}
	if c.OAuthClientSecret != "" && c.OAuthClientID == "" {
		result.AddError("oauth_client_id", c.OAuthClientID,
			"oauth_client_id required when oauth_client_secret is set")
	}

	// Validate retry configuration
	if c.RetryMax < 0 || c.RetryMax > 10 {
		result.AddWarning("retry_max", strconv.Itoa(c.RetryMax),
			"retry_max should be between 0 and 10")
	}
	if c.RetryIntervalSec < 0 || c.RetryIntervalSec > 60 {
		result.AddWarning("retry_interval_sec", strconv.Itoa(c.RetryIntervalSec),
			"retry_interval_sec should be between 0 and 60")
	}

	// Validate timeouts
	if c.DialTimeoutSec < 1 || c.DialTimeoutSec > 300 {
		result.AddWarning("dial_timeout_sec", strconv.Itoa(c.DialTimeoutSec),
			"dial_timeout_sec should be between 1 and 300")
	}
	if c.ResponseHeaderTimeoutSec < 1 || c.ResponseHeaderTimeoutSec > 600 {
		result.AddWarning("response_header_timeout_sec", strconv.Itoa(c.ResponseHeaderTimeoutSec),
			"response_header_timeout_sec should be between 1 and 600")
	}

	// Validate rate limiting
	if c.RateLimitEnabled {
		if c.RateLimitRPS <= 0 {
			result.AddError("rate_limit_rps", strconv.Itoa(c.RateLimitRPS),
				"must be positive when rate limiting is enabled")
		}
		if c.RateLimitBurst <= 0 {
			result.AddError("rate_limit_burst", strconv.Itoa(c.RateLimitBurst),
				"must be positive when rate limiting is enabled")
		}
	}

	// Validate auto-ban thresholds
	if c.AutoBan429Threshold <= 0 {
		result.AddWarning("auto_ban_429_threshold", strconv.Itoa(c.AutoBan429Threshold),
			"should be positive")
	}
	if c.AutoBan403Threshold <= 0 {
		result.AddWarning("auto_ban_403_threshold", strconv.Itoa(c.AutoBan403Threshold),
			"should be positive")
	}

	// Validate management keys & remote management hardening
	rp := strings.ToLower(strings.TrimSpace(c.RunProfile))
	prodLike := (rp == "prod" || rp == "production")
	if c.ManagementKey == "" && c.ManagementKeyHash == "" {
		if prodLike || c.ManagementAllowRemote {
			result.AddError("management_key", "", "management key (plain or hash) required in production or when remote management is enabled")
		} else {
			result.AddWarning("management_key", "", "no management key set, management API will be disabled")
		}
	}
	if c.ManagementAllowRemote {
		if len(c.ManagementRemoteAllowIPs) == 0 {
			result.AddError("management_remote_allow_ips", "", "ip whitelist required when remote management is enabled")
		}
		if c.ManagementRemoteTTlHours <= 0 {
			result.AddWarning("management_remote_ttl_hours", strconv.Itoa(c.ManagementRemoteTTlHours), "ttl hours should be positive when remote management is enabled")
		}
	}

	// Validate upstream endpoint
	if c.CodeAssist != "" {
		if _, err := url.Parse(c.CodeAssist); err != nil {
			result.AddError("code_assist_endpoint", c.CodeAssist, "invalid URL format")
		}
	}

	return result
}

// validatePort validates a port string
func validatePort(port string) error {
	if port == "" {
		return fmt.Errorf("port cannot be empty")
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number: %v", err)
	}

	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", portNum)
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ValidateAndExpandPaths validates and expands file paths in configuration
func (c *Config) ValidateAndExpandPaths() error {
	var err error

	// Expand auth directory
	if c.AuthDir != "" {
		c.AuthDir, err = expandPath(c.AuthDir)
		if err != nil {
			return fmt.Errorf("invalid auth_dir path: %v", err)
		}
	}

	// Expand storage base directory
	if c.StorageBaseDir != "" {
		c.StorageBaseDir, err = expandPath(c.StorageBaseDir)
		if err != nil {
			return fmt.Errorf("invalid storage_base_dir path: %v", err)
		}
	}

	// Expand log file destination
	if c.LogFile != "" {
		c.LogFile, err = expandPath(c.LogFile)
		if err != nil {
			return fmt.Errorf("invalid log_file path: %v", err)
		}
	}

	return nil
}

// expandPath expands ~ and environment variables in file paths
func expandPath(path string) (string, error) {
	if path == "" {
		return path, nil
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot get home directory: %v", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot convert to absolute path: %v", err)
	}

	return absPath, nil
}
