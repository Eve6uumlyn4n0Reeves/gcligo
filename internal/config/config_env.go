package config

import (
	"os"
	"strings"
)

func (cm *ConfigManager) mergeEnvVars() {
	if cm.config == nil {
		cm.config = cm.defaultConfig()
	}

	if v := os.Getenv("OPENAI_PORT"); v != "" {
		if port, err := parsePort(v); err == nil {
			cm.config.OpenAIPort = port
		}
	}
	if v := os.Getenv("GEMINI_PORT"); v != "" {
		if port, err := parsePort(v); err == nil {
			cm.config.GeminiPort = port
		}
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cm.config.OpenAIKey = v
	}
	if v := os.Getenv("GEMINI_API_KEY"); v != "" {
		cm.config.GeminiKey = v
	}
	if v := os.Getenv("MANAGEMENT_KEY"); v != "" {
		cm.config.ManagementKey = v
	}
	if v := os.Getenv("MANAGEMENT_KEY_HASH"); v != "" {
		cm.config.ManagementKeyHash = v
	}
	if v := os.Getenv("MANAGEMENT_ALLOW_REMOTE"); v != "" {
		cm.config.ManagementAllowRemote = !(v == "false" || v == "0")
	}
	if v := os.Getenv("MANAGEMENT_REMOTE_TTL_HOURS"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.ManagementRemoteTTlHours = n
		}
	}
	if v := os.Getenv("MANAGEMENT_REMOTE_ALLOW_IPS"); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		cm.config.ManagementRemoteAllowIPs = out
	}
	if v := os.Getenv("BASE_PATH"); v != "" {
		cm.config.BasePath = normalizeBasePath(v)
	}
	if v := os.Getenv("CODE_ASSIST_ENDPOINT"); v != "" {
		cm.config.CodeAssistEndpoint = v
	}
	if v := os.Getenv("GOOGLE_BEARER_TOKEN"); v != "" {
		cm.config.GoogleBearerToken = v
	}
	if v := os.Getenv("GOOGLE_PROJECT_ID"); v != "" {
		cm.config.GoogleProjectID = v
	}
	if v := os.Getenv("STORAGE_BACKEND"); v != "" {
		cm.config.StorageBackend = strings.ToLower(v)
	}
	if v := os.Getenv("STORAGE_BASE_DIR"); v != "" {
		cm.config.StorageBaseDir = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cm.config.RedisAddr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cm.config.RedisPassword = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.RedisDB = n
		}
	}
	if v := os.Getenv("REDIS_PREFIX"); v != "" {
		cm.config.RedisPrefix = v
	}
	if v := os.Getenv("MONGODB_URI"); v != "" {
		cm.config.MongoDBURI = v
	}
	if v := os.Getenv("MONGODB_DATABASE"); v != "" {
		cm.config.MongoDatabase = v
	}
	if v := os.Getenv("POSTGRES_DSN"); v != "" {
		cm.config.PostgresDSN = v
	}
	if v := os.Getenv("AUTH_DIR"); v != "" {
		cm.config.AuthDir = v
	}
	if v := os.Getenv("PROXY_URL"); v != "" {
		cm.config.ProxyURL = v
	}
	if v := os.Getenv("OAUTH_CLIENT_ID"); v != "" {
		cm.config.OAuthClientID = v
	}
	if v := os.Getenv("OAUTH_CLIENT_SECRET"); v != "" {
		cm.config.OAuthClientSecret = v
	}
	if v := os.Getenv("OAUTH_REDIRECT_URL"); v != "" {
		cm.config.OAuthRedirectURL = v
	}
	if v := os.Getenv("DEBUG"); v == "true" || v == "1" {
		cm.config.Debug = true
	}
	if v := os.Getenv("LOG_FILE"); v != "" {
		cm.config.LogFile = v
	}
	if v := os.Getenv("REQUEST_LOG"); v == "true" || v == "1" {
		cm.config.RequestLog = true
	}
	if v := os.Getenv("DISABLED_MODELS"); v != "" {
		cm.config.DisabledModels = strings.Split(v, ",")
	}
	if v := os.Getenv("ANTI_TRUNCATION_MAX_ATTEMPTS"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.AntiTruncationMax = n
		}
	}
	if v := os.Getenv("ANTI_TRUNCATION_ENABLED"); v == "true" || v == "1" {
		cm.config.AntiTruncationEnabled = true
	}
	if v := os.Getenv("CALLS_PER_ROTATION"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.CallsPerRotation = n
		}
	}
	if v := os.Getenv("OPENAI_IMAGES_INCLUDE_MIME"); v == "true" || v == "1" {
		cm.config.OpenAIImagesIncludeMime = true
	}
	if v := os.Getenv("TOOL_ARGS_DELTA_CHUNK"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.ToolArgsDeltaChunk = n
		}
	}
	if v := os.Getenv("FAKE_STREAMING_ENABLED"); v == "true" || v == "1" {
		cm.config.FakeStreamingEnabled = true
	}
	if v := os.Getenv("FAKE_STREAMING_CHUNK_SIZE"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.FakeStreamingChunkSize = n
		}
	}
	if v := os.Getenv("FAKE_STREAMING_DELAY_MS"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.FakeStreamingDelayMs = n
		}
	}
	if v := os.Getenv("AUTO_IMAGE_PLACEHOLDER"); v == "false" || v == "0" {
		cm.config.AutoImagePlaceholder = false
	}
	if v := os.Getenv("SANITIZER_ENABLED"); v != "" {
		lower := strings.ToLower(strings.TrimSpace(v))
		cm.config.SanitizerEnabled = !(lower == "false" || lower == "0")
	}
	if v := os.Getenv("SANITIZER_PATTERNS"); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		cm.config.SanitizerPatterns = out
	}
	if v := os.Getenv("RATE_LIMIT_ENABLED"); v == "true" || v == "1" {
		cm.config.RateLimitEnabled = true
	}
	if v := os.Getenv("RATE_LIMIT_RPS"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.RateLimitRPS = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.RateLimitBurst = n
		}
	}
	if v := os.Getenv("USAGE_RESET_INTERVAL_HOURS"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.UsageResetIntervalHours = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("USAGE_RESET_TIMEZONE")); v != "" {
		cm.config.UsageResetTimezone = v
	}
	if v := os.Getenv("USAGE_RESET_HOUR_LOCAL"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.UsageResetHourLocal = n
		}
	}
	if v := os.Getenv("AUTO_BAN_ENABLED"); v != "" {
		cm.config.AutoBanEnabled = !(v == "false" || v == "0")
	}
	if v := os.Getenv("AUTO_BAN_429_THRESHOLD"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.AutoBan429Threshold = n
		}
	}
	if v := os.Getenv("AUTO_BAN_403_THRESHOLD"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.AutoBan403Threshold = n
		}
	}
	if v := os.Getenv("AUTO_BAN_401_THRESHOLD"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.AutoBan401Threshold = n
		}
	}
	if v := os.Getenv("AUTO_BAN_5XX_THRESHOLD"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.AutoBan5xxThreshold = n
		}
	}
	if v := os.Getenv("AUTO_BAN_CONSECUTIVE_FAILS"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.AutoBanConsecutiveFails = n
		}
	}
	if v := os.Getenv("AUTO_RECOVERY_ENABLED"); v != "" {
		cm.config.AutoRecoveryEnabled = !(v == "false" || v == "0")
	}
	if v := os.Getenv("AUTO_RECOVERY_INTERVAL_MIN"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.AutoRecoveryIntervalMin = n
		}
	}
	if v := os.Getenv("HEADER_PASSTHROUGH"); v == "true" || v == "1" {
		cm.config.HeaderPassThrough = true
	}
	if v := os.Getenv("WEB_ADMIN_ENABLED"); v != "" {
		cm.config.WebAdminEnabled = !(v == "false" || v == "0")
	}
	if v := os.Getenv("AUTO_LOAD_ENV_CREDS"); v != "" {
		cm.config.AutoLoadEnvCreds = (v == "true" || v == "1")
	}
	if v := os.Getenv("AUTOPROBE_DISABLE_THRESHOLD_PCT"); v != "" {
		if n, err := parseInt(v); err == nil {
			cm.config.AutoProbeDisableThresholdPct = n
		}
	}
}
