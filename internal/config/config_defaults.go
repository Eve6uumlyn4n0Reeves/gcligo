package config

// getDefaultRegexReplacements returns default regex replacement rules
// These rules are inspired by the Python gcli2api project
func getDefaultRegexReplacements() []RegexReplacement {
	return []RegexReplacement{
		{
			Name:        "age_pattern",
			Pattern:     `(?:[1-9]|1[0-8])岁(?:的)?|(?:十一|十二|十三|十四|十五|十六|十七|十八|十|一|二|三|四|五|六|七|八|九)岁(?:的)?`,
			Replacement: "",
			Enabled:     false, // Disabled by default, users can enable in config
		},
		// Add more default rules here as needed
	}
}

// defaultConfig returns the default configuration using central default values.
func (cm *ConfigManager) defaultConfig() *FileConfig {
	defaults := GetDefaults()

	return &FileConfig{
		Port:            0,
		OpenAIPort:      parsePortOrDefault(defaults.OpenAIPort),
		GeminiPort:      parsePortOrDefault(defaults.GeminiPort),
		WebAdminEnabled: defaults.WebAdminEnabled,
		BasePath:        defaults.BasePath,

		StorageBackend: defaults.StorageBackend,
		StorageBaseDir: defaults.StorageBaseDir,
		RedisAddr:      defaults.RedisAddr,
		RedisPassword:  defaults.RedisPassword,
		RedisDB:        defaults.RedisDB,
		RedisPrefix:    defaults.RedisPrefix,
		MongoDBURI:     "mongodb://localhost:27017",
		MongoDatabase:  defaults.MongoDatabase,
		PostgresDSN:    "",
		GitRemoteURL:   "",
		GitBranch:      defaults.GitBranch,
		GitUsername:    "",
		GitPassword:    "",
		GitAuthorName:  defaults.GitAuthorName,
		GitAuthorEmail: defaults.GitAuthorEmail,

		Debug:   false,
		LogFile: "",

		AuthDir:          defaults.AuthDir,
		CallsPerRotation: defaults.CallsPerRotation,

		RetryEnabled:        defaults.RetryEnabled,
		RetryMax:            defaults.RetryMax,
		RetryIntervalSec:    defaults.RetryIntervalSec,
		RetryMaxIntervalSec: defaults.RetryMaxIntervalSec,
		RetryOn5xx:          defaults.RetryOn5xx,
		RetryOnNetworkError: defaults.RetryOnNetworkError,

		AntiTruncationMax:     defaults.AntiTruncationMax,
		AntiTruncationEnabled: defaults.AntiTruncationEnabled,
		CompatibilityMode:     defaults.CompatibilityMode,
		RequestLog:            false,

		CodeAssistEndpoint: defaults.CodeAssistEndpoint,

		DialTimeoutSec:           defaults.DialTimeoutSec,
		TLSHandshakeTimeoutSec:   defaults.TLSHandshakeTimeoutSec,
		ResponseHeaderTimeoutSec: defaults.ResponseHeaderTimeoutSec,
		ExpectContinueTimeoutSec: defaults.ExpectContinueTimeoutSec,

		RateLimitEnabled: defaults.RateLimitEnabled,
		RateLimitRPS:     defaults.RateLimitRPS,
		RateLimitBurst:   defaults.RateLimitBurst,

		UsageResetIntervalHours: defaults.UsageResetIntervalHours,
		UsageResetTimezone:      defaults.UsageResetTimezone,
		UsageResetHourLocal:     defaults.UsageResetHourLocal,

		AutoBanEnabled:          defaults.AutoBanEnabled,
		AutoBan429Threshold:     defaults.AutoBan429Threshold,
		AutoBan403Threshold:     defaults.AutoBan403Threshold,
		AutoBan401Threshold:     defaults.AutoBan401Threshold,
		AutoBan5xxThreshold:     defaults.AutoBan5xxThreshold,
		AutoBanConsecutiveFails: defaults.AutoBanConsecutiveFails,

		AutoRecoveryEnabled:     defaults.AutoRecoveryEnabled,
		AutoRecoveryIntervalMin: defaults.AutoRecoveryIntervalMin,

		PersistRoutingState:       false,
		RoutingPersistIntervalSec: 0,

		FakeStreamingEnabled:   defaults.FakeStreamingEnabled,
		FakeStreamingChunkSize: defaults.FakeStreamingChunkSize,
		FakeStreamingDelayMs:   defaults.FakeStreamingDelayMs,

		AutoImagePlaceholder: defaults.AutoImagePlaceholder,
		HeaderPassThrough:    defaults.HeaderPassThrough,
		ToolArgsDeltaChunk:   defaults.ToolArgsDeltaChunk,
		SanitizerEnabled:     defaults.SanitizerEnabled,
		SanitizerPatterns:    append([]string(nil), defaults.SanitizerPatterns...),

		PreferredBaseModels: append([]string(nil), defaults.PreferredBaseModels...),
		DisabledModels:      append([]string(nil), defaults.DisabledModels...),
		RegexReplacements:   getDefaultRegexReplacements(),

		AutoProbeEnabled:             defaults.AutoProbeEnabled,
		AutoProbeHourUTC:             defaults.AutoProbeHourUTC,
		AutoProbeModel:               defaults.AutoProbeModel,
		AutoProbeTimeoutSec:          defaults.AutoProbeTimeoutSec,
		AutoProbeDisableThresholdPct: 0,

		AutoLoadEnvCreds: false,

		ManagementAllowRemote:    false,
		ManagementRemoteTTlHours: 0,
		ManagementRemoteAllowIPs: []string{},
	}
}
