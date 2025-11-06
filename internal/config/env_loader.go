package config

import "strings"

// loadFromEnv loads configuration from environment variables only.
func loadFromEnv() *Config {
	defaults := GetDefaults()
	cfg := baseConfigFromEnv(defaults)

	applyRetryEnvVars(cfg)
	applyTimeoutEnvVars(cfg)
	applyUsageEnvVars(cfg)
	applyAutoBanEnvVars(cfg)
	applyAutoProbeEnvVars(cfg)
	applyConcurrencyEnvVars(cfg)
	applyRoutingEnvVars(cfg)
	applyListEnvVars(cfg)
	applyRateLimitEnvVars(cfg)
	applyManagementEnvVars(cfg)
	applyMiscEnvVars(cfg)

	cfg = applyRunProfile(cfg)

	// 同步顶级字段到子结构体
	cfg.SyncToDomains()

	return cfg
}

func baseConfigFromEnv(defaults DefaultValues) *Config {
	return &Config{
		OpenAIPort:      getenv("OPENAI_PORT", defaults.OpenAIPort),
		GeminiPort:      getenv("GEMINI_PORT", defaults.GeminiPort),
		OpenAIKey:       getenv("OPENAI_API_KEY", ""),
		GeminiKey:       getenv("GEMINI_API_KEY", ""),
		WebAdminEnabled: getenvBool("WEB_ADMIN_ENABLED", defaults.WebAdminEnabled),
		BasePath:        normalizeBasePath(getenv("BASE_PATH", defaults.BasePath)),
		CodeAssist:      getenv("CODE_ASSIST_ENDPOINT", defaults.CodeAssistEndpoint),
		GoogleToken:     getenv("GOOGLE_BEARER_TOKEN", ""),
		GoogleProjID:    getenv("GOOGLE_PROJECT_ID", ""),

		StorageBackend: strings.ToLower(getenv("STORAGE_BACKEND", defaults.StorageBackend)),
		StorageBaseDir: getenv("STORAGE_BASE_DIR", defaults.StorageBaseDir),
		RedisAddr:      getenv("REDIS_ADDR", defaults.RedisAddr),
		RedisPassword:  getenv("REDIS_PASSWORD", defaults.RedisPassword),
		RedisDB:        defaults.RedisDB,
		RedisPrefix:    getenv("REDIS_PREFIX", defaults.RedisPrefix),
		MongoURI:       getenv("MONGODB_URI", ""),
		MongoDatabase:  getenv("MONGODB_DATABASE", defaults.MongoDatabase),
		PostgresDSN:    getenv("POSTGRES_DSN", ""),
		GitRemoteURL:   getenv("GIT_REMOTE_URL", ""),
		GitBranch:      getenv("GIT_BRANCH", defaults.GitBranch),
		GitUsername:    getenv("GIT_USERNAME", ""),
		GitPassword:    getenv("GIT_PASSWORD", ""),
		GitAuthorName:  getenv("GIT_AUTHOR_NAME", defaults.GitAuthorName),
		GitAuthorEmail: getenv("GIT_AUTHOR_EMAIL", defaults.GitAuthorEmail),

		RetryEnabled:        getenvBool("RETRY_429_ENABLED", defaults.RetryEnabled),
		RetryMax:            defaults.RetryMax,
		RetryIntervalSec:    defaults.RetryIntervalSec,
		RetryMaxIntervalSec: defaults.RetryMaxIntervalSec,
		RetryOn5xx:          getenvBool("RETRY_5XX_ENABLED", defaults.RetryOn5xx),
		RetryOnNetworkError: getenvBool("RETRY_NETWORK_ERROR_ENABLED", defaults.RetryOnNetworkError),

		AntiTruncationMax:     defaults.AntiTruncationMax,
		AntiTruncationEnabled: getenvBool("ANTI_TRUNCATION_ENABLED", defaults.AntiTruncationEnabled),
		PprofEnabled:          getenvBool("PPROF_ENABLED", false),
		RequestLogEnabled:     getenvBool("REQUEST_LOG_ENABLED", false),
		CompatibilityMode:     getenvBool("COMPATIBILITY_MODE", defaults.CompatibilityMode),

		DialTimeoutSec:           defaults.DialTimeoutSec,
		TLSHandshakeTimeoutSec:   defaults.TLSHandshakeTimeoutSec,
		ResponseHeaderTimeoutSec: defaults.ResponseHeaderTimeoutSec,
		ExpectContinueTimeoutSec: defaults.ExpectContinueTimeoutSec,

		OpenAIImagesIncludeMIME: getenvBool("OPENAI_IMAGES_INCLUDE_MIME", false),
		ToolArgsDeltaChunk:      defaults.ToolArgsDeltaChunk,
		PreferredBaseModels:     defaults.PreferredBaseModels,

		ManagementKey:         getenv("MANAGEMENT_KEY", ""),
		ManagementKeyHash:     getenv("MANAGEMENT_KEY_HASH", ""),
		ManagementReadOnly:    getenvBool("MANAGEMENT_READ_ONLY", defaults.ManagementReadOnly),
		ManagementAllowRemote: getenvBool("MANAGEMENT_ALLOW_REMOTE", false),
		AuthDir:               getenv("AUTH_DIR", defaults.AuthDir),

		CallsPerRotation:        defaults.CallsPerRotation,
		RateLimitEnabled:        getenvBool("RATE_LIMIT_ENABLED", defaults.RateLimitEnabled),
		RateLimitRPS:            defaults.RateLimitRPS,
		RateLimitBurst:          defaults.RateLimitBurst,
		UsageResetIntervalHours: defaults.UsageResetIntervalHours,

		ProxyURL: getenv("PROXY_URL", ""),
		Debug:    getenvBool("DEBUG", false),
		LogFile:  getenv("LOG_FILE", ""),

		HeaderPassThrough: getenvBool("HEADER_PASSTHROUGH", defaults.HeaderPassThrough),

		AutoBanEnabled:          getenvBool("AUTO_BAN_ENABLED", defaults.AutoBanEnabled),
		AutoBan429Threshold:     defaults.AutoBan429Threshold,
		AutoBan403Threshold:     defaults.AutoBan403Threshold,
		AutoBan401Threshold:     defaults.AutoBan401Threshold,
		AutoBan5xxThreshold:     defaults.AutoBan5xxThreshold,
		AutoBanConsecutiveFails: defaults.AutoBanConsecutiveFails,
		AutoRecoveryEnabled:     getenvBool("AUTO_RECOVERY_ENABLED", defaults.AutoRecoveryEnabled),
		AutoRecoveryIntervalMin: defaults.AutoRecoveryIntervalMin,

		AutoProbeEnabled:              getenvBool("AUTO_PROBE_ENABLED", defaults.AutoProbeEnabled),
		AutoProbeHourUTC:              defaults.AutoProbeHourUTC,
		AutoProbeModel:                defaults.AutoProbeModel,
		AutoProbeTimeoutSec:           defaults.AutoProbeTimeoutSec,
		AutoProbeDisableThresholdPct:  0,
		AutoImagePlaceholder:          defaults.AutoImagePlaceholder,
		AutoLoadEnvCreds:              strings.EqualFold(getenv("AUTO_LOAD_ENV_CREDS", "false"), "true"),
		UpstreamProvider:              strings.ToLower(getenv("UPSTREAM_PROVIDER", defaults.UpstreamProvider)),
		MaxConcurrentPerCredential:    0,
		RefreshAheadSeconds:           180,
		RefreshSingleflightTimeoutSec: 10,
		StickyTTLSeconds:              300,
		RouterCooldownBaseMS:          2000,
		RouterCooldownMaxMS:           60000,
		PersistRoutingState:           false,
		RoutingPersistIntervalSec:     60,
		RoutingDebugHeaders:           false,
		RunProfile:                    strings.ToLower(getenv("RUN_PROFILE", "")),
	}
}

func applyRetryEnvVars(cfg *Config) {
	setIntFromEnv("RETRY_429_MAX_RETRIES", func(n int) { cfg.RetryMax = n })
	setIntFromEnv("RETRY_429_INTERVAL", func(n int) { cfg.RetryIntervalSec = n })
	setIntFromEnv("RETRY_MAX_INTERVAL", func(n int) { cfg.RetryMaxIntervalSec = n })
	setIntFromEnv("ANTI_TRUNCATION_MAX_ATTEMPTS", func(n int) { cfg.AntiTruncationMax = n })
}

func applyTimeoutEnvVars(cfg *Config) {
	setIntFromEnv("DIAL_TIMEOUT_SEC", func(n int) { cfg.DialTimeoutSec = n })
	setIntFromEnv("TLS_HANDSHAKE_TIMEOUT_SEC", func(n int) { cfg.TLSHandshakeTimeoutSec = n })
	setIntFromEnv("RESPONSE_HEADER_TIMEOUT_SEC", func(n int) { cfg.ResponseHeaderTimeoutSec = n })
	setIntFromEnv("EXPECT_CONTINUE_TIMEOUT_SEC", func(n int) { cfg.ExpectContinueTimeoutSec = n })
	setIntFromEnv("REDIS_DB", func(n int) { cfg.RedisDB = n })
}

func applyUsageEnvVars(cfg *Config) {
	setIntFromEnv("USAGE_RESET_INTERVAL_HOURS", func(n int) { cfg.UsageResetIntervalHours = n })
	if v := strings.TrimSpace(getenv("USAGE_RESET_TIMEZONE", "")); v != "" {
		cfg.UsageResetTimezone = v
	}
	setIntFromEnv("USAGE_RESET_HOUR_LOCAL", func(n int) { cfg.UsageResetHourLocal = n })
	setIntFromEnv("CALLS_PER_ROTATION", func(n int) { cfg.CallsPerRotation = n })
}

func applyAutoBanEnvVars(cfg *Config) {
	setIntFromEnv("AUTO_BAN_429_THRESHOLD", func(n int) { cfg.AutoBan429Threshold = n })
	setIntFromEnv("AUTO_BAN_403_THRESHOLD", func(n int) { cfg.AutoBan403Threshold = n })
	setIntFromEnv("AUTO_BAN_401_THRESHOLD", func(n int) { cfg.AutoBan401Threshold = n })
	setIntFromEnv("AUTO_BAN_5XX_THRESHOLD", func(n int) { cfg.AutoBan5xxThreshold = n })
	setIntFromEnv("AUTO_BAN_CONSECUTIVE_FAILS", func(n int) { cfg.AutoBanConsecutiveFails = n })
	setIntFromEnv("AUTO_RECOVERY_INTERVAL_MIN", func(n int) { cfg.AutoRecoveryIntervalMin = n })
}

func applyAutoProbeEnvVars(cfg *Config) {
	setToggleFromEnv("AUTO_PROBE_ENABLED", func(v bool) { cfg.AutoProbeEnabled = v })
	setIntFromEnv("AUTO_PROBE_HOUR_UTC", func(n int) { cfg.AutoProbeHourUTC = n })
	setIntFromEnv("AUTO_PROBE_TIMEOUT_SEC", func(n int) { cfg.AutoProbeTimeoutSec = n })
	setIntFromEnv("AUTO_PROBE_DISABLE_THRESHOLD_PCT", func(n int) { cfg.AutoProbeDisableThresholdPct = n })
	if v := strings.TrimSpace(getenv("AUTO_PROBE_MODEL", "")); v != "" {
		cfg.AutoProbeModel = v
	}
}

func applyConcurrencyEnvVars(cfg *Config) {
	setIntFromEnv("MAX_CONCURRENT_PER_CREDENTIAL", func(n int) { cfg.MaxConcurrentPerCredential = n })
	setIntFromEnv("REFRESH_AHEAD_SECONDS", func(n int) { cfg.RefreshAheadSeconds = n })
	setIntFromEnv("REFRESH_SINGLEFLIGHT_TIMEOUT_SEC", func(n int) {
		cfg.RefreshSingleflightTimeoutSec = n
	})
}

func applyRoutingEnvVars(cfg *Config) {
	setIntFromEnv("STICKY_TTL_SECONDS", func(n int) { cfg.StickyTTLSeconds = n })
	setIntFromEnv("ROUTER_COOLDOWN_BASE_MS", func(n int) { cfg.RouterCooldownBaseMS = n })
	setIntFromEnv("ROUTER_COOLDOWN_MAX_MS", func(n int) { cfg.RouterCooldownMaxMS = n })
	setIntFromEnv("ROUTING_PERSIST_INTERVAL_SEC", func(n int) { cfg.RoutingPersistIntervalSec = n })

	setToggleFromEnv("PERSIST_ROUTING_STATE", func(v bool) { cfg.PersistRoutingState = v })
	setToggleFromEnv("ROUTING_DEBUG_HEADERS", func(v bool) { cfg.RoutingDebugHeaders = v })
}

func applyListEnvVars(cfg *Config) {
	if v := getenv("DISABLED_MODELS", ""); v != "" {
		cfg.DisabledModels = splitAndTrim(v, ",")
	}
	if v := getenv("PREFERRED_BASE_MODELS", ""); v != "" {
		cfg.PreferredBaseModels = splitAndTrim(v, ",")
	}
}

func applyRateLimitEnvVars(cfg *Config) {
	setIntFromEnv("RATE_LIMIT_RPS", func(n int) { cfg.RateLimitRPS = n })
	setIntFromEnv("RATE_LIMIT_BURST", func(n int) { cfg.RateLimitBurst = n })
}

func applyManagementEnvVars(cfg *Config) {
	setIntFromEnv("MANAGEMENT_REMOTE_TTL_HOURS", func(n int) { cfg.ManagementRemoteTTlHours = n })
	if v := getenv("MANAGEMENT_REMOTE_ALLOW_IPS", ""); v != "" {
		cfg.ManagementRemoteAllowIPs = splitAndTrim(v, ",")
	}
}

func applyMiscEnvVars(cfg *Config) {
	setIntFromEnv("TOOL_ARGS_DELTA_CHUNK", func(n int) { cfg.ToolArgsDeltaChunk = n })
	if v := getenv("AUTO_IMAGE_PLACEHOLDER", ""); v != "" {
		lowered := strings.ToLower(strings.TrimSpace(v))
		cfg.AutoImagePlaceholder = !(lowered == "false" || lowered == "0")
	}
	setToggleFromEnv("SANITIZER_ENABLED", func(v bool) { cfg.SanitizerEnabled = v })
	if v := getenv("SANITIZER_PATTERNS", ""); v != "" {
		cfg.SanitizerPatterns = splitAndTrim(v, ",")
	}
}

func applyRunProfile(c *Config) *Config {
	rp := strings.TrimSpace(strings.ToLower(c.RunProfile))
	switch rp {
	case "prod", "production":
		c.PprofEnabled = false
		if c.ManagementAllowRemote {
			c.HeaderPassThrough = false
		}
	}
	return c
}
