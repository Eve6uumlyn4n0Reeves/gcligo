package config

import (
	"strconv"
	"strings"
)

// fileConfigToConfig converts FileConfig to Config.
func fileConfigToConfig(fc *FileConfig) *Config {
	apiKeys := ""
	if len(fc.APIKeys) > 0 {
		apiKeys = fc.APIKeys[0]
	}

	out := &Config{
		OpenAIPort:              strconv.Itoa(fc.OpenAIPort),
		GeminiPort:              strconv.Itoa(fc.GeminiPort),
		OpenAIKey:               firstNonEmpty(fc.OpenAIKey, apiKeys),
		GeminiKey:               firstNonEmpty(fc.GeminiKey, apiKeys),
		CodeAssist:              fc.CodeAssistEndpoint,
		GoogleToken:             fc.GoogleBearerToken,
		GoogleProjID:            fc.GoogleProjectID,
		StorageBackend:          strings.ToLower(fc.StorageBackend),
		StorageBaseDir:          fc.StorageBaseDir,
		RedisAddr:               fc.RedisAddr,
		RedisPassword:           fc.RedisPassword,
		RedisDB:                 fc.RedisDB,
		RedisPrefix:             fc.RedisPrefix,
		MongoURI:                fc.MongoDBURI,
		MongoDatabase:           fc.MongoDatabase,
		PostgresDSN:             fc.PostgresDSN,
		GitRemoteURL:            fc.GitRemoteURL,
		GitBranch:               fc.GitBranch,
		GitUsername:             fc.GitUsername,
		GitPassword:             fc.GitPassword,
		GitAuthorName:           fc.GitAuthorName,
		GitAuthorEmail:          fc.GitAuthorEmail,
		AutoBanEnabled:          fc.AutoBanEnabled,
		AutoBan429Threshold:     fc.AutoBan429Threshold,
		AutoBan403Threshold:     fc.AutoBan403Threshold,
		AutoBan401Threshold:     fc.AutoBan401Threshold,
		AutoBan5xxThreshold:     fc.AutoBan5xxThreshold,
		AutoBanConsecutiveFails: fc.AutoBanConsecutiveFails,
		AutoRecoveryEnabled:     fc.AutoRecoveryEnabled,
		AutoRecoveryIntervalMin: fc.AutoRecoveryIntervalMin,

		RetryEnabled:        fc.RetryEnabled,
		RetryMax:            fc.RetryMax,
		RetryIntervalSec:    fc.RetryIntervalSec,
		RetryMaxIntervalSec: fc.RetryMaxIntervalSec,
		RetryOn5xx:          fc.RetryOn5xx,
		RetryOnNetworkError: fc.RetryOnNetworkError,

		AntiTruncationMax:     fc.AntiTruncationMax,
		AntiTruncationEnabled: fc.AntiTruncationEnabled,
		CompatibilityMode:     fc.CompatibilityMode,
		PprofEnabled:          false,
		RequestLogEnabled:     fc.RequestLog,

		DialTimeoutSec:           fc.DialTimeoutSec,
		TLSHandshakeTimeoutSec:   fc.TLSHandshakeTimeoutSec,
		ResponseHeaderTimeoutSec: fc.ResponseHeaderTimeoutSec,
		ExpectContinueTimeoutSec: fc.ExpectContinueTimeoutSec,

		DisabledModels:          fc.DisabledModels,
		OpenAIImagesIncludeMIME: fc.OpenAIImagesIncludeMime,
		ToolArgsDeltaChunk:      fc.ToolArgsDeltaChunk,
		PreferredBaseModels:     fc.PreferredBaseModels,

		ManagementKey:            fc.ManagementKey,
		ManagementKeyHash:        fc.ManagementKeyHash,
		ManagementAllowRemote:    fc.ManagementAllowRemote,
		ManagementRemoteTTlHours: fc.ManagementRemoteTTlHours,
		ManagementRemoteAllowIPs: fc.ManagementRemoteAllowIPs,
		AuthDir:                  fc.AuthDir,
		CallsPerRotation:         fc.CallsPerRotation,
		RateLimitEnabled:         fc.RateLimitEnabled,
		RateLimitRPS:             fc.RateLimitRPS,
		RateLimitBurst:           fc.RateLimitBurst,
		UsageResetIntervalHours:  fc.UsageResetIntervalHours,
		UsageResetTimezone:       fc.UsageResetTimezone,
		UsageResetHourLocal:      fc.UsageResetHourLocal,

		ProxyURL: fc.ProxyURL,
		Debug:    fc.Debug,
		LogFile:  fc.LogFile,

		FakeStreamingEnabled:   fc.FakeStreamingEnabled,
		FakeStreamingChunkSize: fc.FakeStreamingChunkSize,
		FakeStreamingDelayMs:   fc.FakeStreamingDelayMs,
		AutoImagePlaceholder:   fc.AutoImagePlaceholder,
		SanitizerEnabled:       fc.SanitizerEnabled,
		SanitizerPatterns:      fc.SanitizerPatterns,
		RegexReplacements:      fc.RegexReplacements,

		OAuthClientID:     fc.OAuthClientID,
		OAuthClientSecret: fc.OAuthClientSecret,
		OAuthRedirectURL:  fc.OAuthRedirectURL,

		HeaderPassThrough: fc.HeaderPassThrough,

		WebAdminEnabled: fc.WebAdminEnabled,
		BasePath:        normalizeBasePath(fc.BasePath),

		AutoProbeEnabled:             fc.AutoProbeEnabled,
		AutoProbeHourUTC:             fc.AutoProbeHourUTC,
		AutoProbeModel:               fc.AutoProbeModel,
		AutoProbeTimeoutSec:          fc.AutoProbeTimeoutSec,
		AutoProbeDisableThresholdPct: fc.AutoProbeDisableThresholdPct,

		AutoLoadEnvCreds: fc.AutoLoadEnvCreds,
	}

	if rp := strings.ToLower(fc.RunProfile); rp != "" {
		out.RunProfile = rp
	}
	out = applyRunProfile(out)

	// 同步顶级字段到子结构体
	out.SyncToDomains()

	return out
}
