package config

import (
	"fmt"
	"sync"
)

// Config 主配置结构体，包含所有功能域的配置
type Config struct {
	// 核心配置域
	Server          ServerConfig
	Upstream        UpstreamConfig
	Security        SecurityConfig
	Execution       ExecutionConfig
	Storage         StorageConfig
	Retry           RetryConfig
	RateLimit       RateLimitConfig
	APICompat       APICompatConfig
	ResponseShaping ResponseShapingConfig
	OAuth           OAuthConfig
	AutoBan         AutoBanConfig
	AutoProbe       AutoProbeConfig
	Routing         RoutingConfig

	// 保留向后兼容的顶级字段（用于过渡期）
	// 这些字段会在 Load() 时从子结构体中填充
	//
	// DEPRECATION NOTICE:
	// 这些顶层字段将在未来版本中移除。推荐使用领域结构（如 cfg.Server.OpenAIPort）。
	// - v2.x（当前）：保留，双向同步正常工作
	// - v3.0（计划）：标记为 @deprecated，编译时警告
	// - v4.0（未来）：完全移除
	// 详见 docs/configuration.md 中的"配置结构演进与弃用计划"章节。
	//
	// Deprecated: Use domain structures instead (e.g., cfg.Server.OpenAIPort, cfg.Retry.Enabled).
	OpenAIPort string
	// Deprecated: Use cfg.Server.GeminiPort instead.
	GeminiPort string
	// Deprecated: Use cfg.Server.BasePath instead.
	BasePath string
	// Deprecated: Use cfg.Server.WebAdminEnabled instead.
	WebAdminEnabled bool
	// Deprecated: Use cfg.Server.RunProfile instead.
	RunProfile                    string
	OpenAIKey                     string
	GeminiKey                     string
	CodeAssist                    string
	GoogleToken                   string
	GoogleProjID                  string
	UpstreamProvider              string
	ManagementKey                 string
	ManagementKeyHash             string
	ManagementReadOnly            bool
	ManagementAllowRemote         bool
	ManagementRemoteTTlHours      int
	ManagementRemoteAllowIPs      []string
	AuthDir                       string
	HeaderPassThrough             bool
	Debug                         bool
	LogFile                       string
	CallsPerRotation              int
	MaxConcurrentPerCredential    int
	AutoLoadEnvCreds              bool
	StorageBackend                string
	StorageBaseDir                string
	RedisAddr                     string
	RedisPassword                 string
	RedisDB                       int
	RedisPrefix                   string
	MongoURI                      string
	MongoDatabase                 string
	PostgresDSN                   string
	GitRemoteURL                  string
	GitBranch                     string
	GitUsername                   string
	GitPassword                   string
	GitAuthorName                 string
	GitAuthorEmail                string
	RetryEnabled                  bool
	RetryMax                      int
	RetryIntervalSec              int
	RetryMaxIntervalSec           int
	RetryOn5xx                    bool
	RetryOnNetworkError           bool
	DialTimeoutSec                int
	TLSHandshakeTimeoutSec        int
	ResponseHeaderTimeoutSec      int
	ExpectContinueTimeoutSec      int
	RateLimitEnabled              bool
	RateLimitRPS                  int
	RateLimitBurst                int
	UsageResetIntervalHours       int
	UsageResetTimezone            string
	UsageResetHourLocal           int
	OpenAIImagesIncludeMIME       bool
	ToolArgsDeltaChunk            int
	PreferredBaseModels           []string
	DisabledModels                []string
	DisableModelVariants          bool
	AntiTruncationMax             int
	AntiTruncationEnabled         bool
	FakeStreamingEnabled          bool
	FakeStreamingChunkSize        int
	FakeStreamingDelayMs          int
	AutoImagePlaceholder          bool
	RequestLogEnabled             bool
	PprofEnabled                  bool
	ProxyURL                      string
	SanitizerEnabled              bool
	SanitizerPatterns             []string
	OAuthClientID                 string
	OAuthClientSecret             string
	OAuthRedirectURL              string
	AutoBanEnabled                bool
	AutoBan429Threshold           int
	AutoBan403Threshold           int
	AutoBan401Threshold           int
	AutoBan5xxThreshold           int
	AutoBanConsecutiveFails       int
	AutoRecoveryEnabled           bool
	AutoRecoveryIntervalMin       int
	AutoProbeEnabled              bool
	AutoProbeHourUTC              int
	AutoProbeModel                string
	AutoProbeTimeoutSec           int
	AutoProbeDisableThresholdPct  int
	RefreshAheadSeconds           int
	RefreshSingleflightTimeoutSec int
	StickyTTLSeconds              int
	RouterCooldownBaseMS          int
	RouterCooldownMaxMS           int
	PersistRoutingState           bool
	RoutingPersistIntervalSec     int
	RoutingDebugHeaders           bool
}

var (
	globalConfigManager *ConfigManager
	configOnce          sync.Once
)

// SyncFromDomains 从子结构体同步数据到顶级字段（用于向后兼容）
func (c *Config) SyncFromDomains() {
	// Server
	c.OpenAIPort = c.Server.OpenAIPort
	c.GeminiPort = c.Server.GeminiPort
	c.BasePath = c.Server.BasePath
	c.WebAdminEnabled = c.Server.WebAdminEnabled
	c.RunProfile = c.Server.RunProfile

	// Upstream
	c.OpenAIKey = c.Upstream.OpenAIKey
	c.GeminiKey = c.Upstream.GeminiKey
	c.CodeAssist = c.Upstream.CodeAssist
	c.GoogleToken = c.Upstream.GoogleToken
	c.GoogleProjID = c.Upstream.GoogleProjID
	c.UpstreamProvider = c.Upstream.UpstreamProvider

	// Security
	c.ManagementKey = c.Security.ManagementKey
	c.ManagementKeyHash = c.Security.ManagementKeyHash
	c.ManagementReadOnly = c.Security.ManagementReadOnly
	c.ManagementAllowRemote = c.Security.ManagementAllowRemote
	c.ManagementRemoteTTlHours = c.Security.ManagementRemoteTTlHours
	c.ManagementRemoteAllowIPs = c.Security.ManagementRemoteAllowIPs
	c.AuthDir = c.Security.AuthDir
	c.HeaderPassThrough = c.Security.HeaderPassThrough
	c.Debug = c.Security.Debug
	c.LogFile = c.Security.LogFile

	// Execution
	c.CallsPerRotation = c.Execution.CallsPerRotation
	c.MaxConcurrentPerCredential = c.Execution.MaxConcurrentPerCredential
	c.AutoLoadEnvCreds = c.Execution.AutoLoadEnvCreds

	// Storage
	c.StorageBackend = c.Storage.Backend
	c.StorageBaseDir = c.Storage.BaseDir
	c.RedisAddr = c.Storage.RedisAddr
	c.RedisPassword = c.Storage.RedisPassword
	c.RedisDB = c.Storage.RedisDB
	c.RedisPrefix = c.Storage.RedisPrefix
	c.MongoURI = c.Storage.MongoURI
	c.MongoDatabase = c.Storage.MongoDatabase
	c.PostgresDSN = c.Storage.PostgresDSN
	c.GitRemoteURL = c.Storage.GitRemoteURL
	c.GitBranch = c.Storage.GitBranch
	c.GitUsername = c.Storage.GitUsername
	c.GitPassword = c.Storage.GitPassword
	c.GitAuthorName = c.Storage.GitAuthorName
	c.GitAuthorEmail = c.Storage.GitAuthorEmail

	// Retry
	c.RetryEnabled = c.Retry.Enabled
	c.RetryMax = c.Retry.Max
	c.RetryIntervalSec = c.Retry.IntervalSec
	c.RetryMaxIntervalSec = c.Retry.MaxIntervalSec
	c.RetryOn5xx = c.Retry.On5xx
	c.RetryOnNetworkError = c.Retry.OnNetworkError
	c.DialTimeoutSec = c.Retry.DialTimeoutSec
	c.TLSHandshakeTimeoutSec = c.Retry.TLSHandshakeTimeoutSec
	c.ResponseHeaderTimeoutSec = c.Retry.ResponseHeaderTimeoutSec
	c.ExpectContinueTimeoutSec = c.Retry.ExpectContinueTimeoutSec

	// RateLimit
	c.RateLimitEnabled = c.RateLimit.Enabled
	c.RateLimitRPS = c.RateLimit.RPS
	c.RateLimitBurst = c.RateLimit.Burst
	c.UsageResetIntervalHours = c.RateLimit.UsageResetIntervalHours
	c.UsageResetTimezone = c.RateLimit.UsageResetTimezone
	c.UsageResetHourLocal = c.RateLimit.UsageResetHourLocal

	// APICompat
	c.OpenAIImagesIncludeMIME = c.APICompat.OpenAIImagesIncludeMIME
	c.ToolArgsDeltaChunk = c.APICompat.ToolArgsDeltaChunk
	c.PreferredBaseModels = c.APICompat.PreferredBaseModels
	c.DisabledModels = c.APICompat.DisabledModels
	c.DisableModelVariants = c.APICompat.DisableModelVariants

	// ResponseShaping
	c.AntiTruncationMax = c.ResponseShaping.AntiTruncationMax
	c.AntiTruncationEnabled = c.ResponseShaping.AntiTruncationEnabled
	c.FakeStreamingEnabled = c.ResponseShaping.FakeStreamingEnabled
	c.FakeStreamingChunkSize = c.ResponseShaping.FakeStreamingChunkSize
	c.FakeStreamingDelayMs = c.ResponseShaping.FakeStreamingDelayMs
	c.AutoImagePlaceholder = c.ResponseShaping.AutoImagePlaceholder
	c.RequestLogEnabled = c.ResponseShaping.RequestLogEnabled
	c.PprofEnabled = c.ResponseShaping.PprofEnabled
	c.ProxyURL = c.ResponseShaping.ProxyURL
	c.SanitizerEnabled = c.ResponseShaping.SanitizerEnabled
	c.SanitizerPatterns = c.ResponseShaping.SanitizerPatterns

	// OAuth
	c.OAuthClientID = c.OAuth.ClientID
	c.OAuthClientSecret = c.OAuth.ClientSecret
	c.OAuthRedirectURL = c.OAuth.RedirectURL
	c.RefreshAheadSeconds = c.OAuth.RefreshAheadSeconds
	c.RefreshSingleflightTimeoutSec = c.OAuth.RefreshSingleflightTimeoutSec

	// AutoBan
	c.AutoBanEnabled = c.AutoBan.Enabled
	c.AutoBan429Threshold = c.AutoBan.Ban429Threshold
	c.AutoBan403Threshold = c.AutoBan.Ban403Threshold
	c.AutoBan401Threshold = c.AutoBan.Ban401Threshold
	c.AutoBan5xxThreshold = c.AutoBan.Ban5xxThreshold
	c.AutoBanConsecutiveFails = c.AutoBan.ConsecutiveFails
	c.AutoRecoveryEnabled = c.AutoBan.RecoveryEnabled
	c.AutoRecoveryIntervalMin = c.AutoBan.RecoveryIntervalMin

	// AutoProbe
	c.AutoProbeEnabled = c.AutoProbe.Enabled
	c.AutoProbeHourUTC = c.AutoProbe.HourUTC
	c.AutoProbeModel = c.AutoProbe.Model
	c.AutoProbeTimeoutSec = c.AutoProbe.TimeoutSec
	c.AutoProbeDisableThresholdPct = c.AutoProbe.DisableThresholdPct

	// Routing
	c.StickyTTLSeconds = c.Routing.StickyTTLSeconds
	c.RouterCooldownBaseMS = c.Routing.CooldownBaseMS
	c.RouterCooldownMaxMS = c.Routing.CooldownMaxMS
	c.PersistRoutingState = c.Routing.PersistState
	c.RoutingPersistIntervalSec = c.Routing.PersistIntervalSec
	c.RoutingDebugHeaders = c.Routing.DebugHeaders
}

// SyncToDomains 从顶级字段同步数据到子结构体（用于向后兼容）
func (c *Config) SyncToDomains() {
	c.warnLegacyOverrides("SyncToDomains")

	// Server
	c.Server.OpenAIPort = c.OpenAIPort
	c.Server.GeminiPort = c.GeminiPort
	c.Server.BasePath = c.BasePath
	c.Server.WebAdminEnabled = c.WebAdminEnabled
	c.Server.RunProfile = c.RunProfile

	// Upstream
	c.Upstream.OpenAIKey = c.OpenAIKey
	c.Upstream.GeminiKey = c.GeminiKey
	c.Upstream.CodeAssist = c.CodeAssist
	c.Upstream.GoogleToken = c.GoogleToken
	c.Upstream.GoogleProjID = c.GoogleProjID
	c.Upstream.UpstreamProvider = c.UpstreamProvider

	// Security
	c.Security.ManagementKey = c.ManagementKey
	c.Security.ManagementKeyHash = c.ManagementKeyHash
	c.Security.ManagementReadOnly = c.ManagementReadOnly
	c.Security.ManagementAllowRemote = c.ManagementAllowRemote
	c.Security.ManagementRemoteTTlHours = c.ManagementRemoteTTlHours
	c.Security.ManagementRemoteAllowIPs = c.ManagementRemoteAllowIPs
	c.Security.AuthDir = c.AuthDir
	c.Security.HeaderPassThrough = c.HeaderPassThrough
	c.Security.Debug = c.Debug
	c.Security.LogFile = c.LogFile

	// Execution
	c.Execution.CallsPerRotation = c.CallsPerRotation
	c.Execution.MaxConcurrentPerCredential = c.MaxConcurrentPerCredential
	c.Execution.AutoLoadEnvCreds = c.AutoLoadEnvCreds

	// Storage
	c.Storage.Backend = c.StorageBackend
	c.Storage.BaseDir = c.StorageBaseDir
	c.Storage.RedisAddr = c.RedisAddr
	c.Storage.RedisPassword = c.RedisPassword
	c.Storage.RedisDB = c.RedisDB
	c.Storage.RedisPrefix = c.RedisPrefix
	c.Storage.MongoURI = c.MongoURI
	c.Storage.MongoDatabase = c.MongoDatabase
	c.Storage.PostgresDSN = c.PostgresDSN
	c.Storage.GitRemoteURL = c.GitRemoteURL
	c.Storage.GitBranch = c.GitBranch
	c.Storage.GitUsername = c.GitUsername
	c.Storage.GitPassword = c.GitPassword
	c.Storage.GitAuthorName = c.GitAuthorName
	c.Storage.GitAuthorEmail = c.GitAuthorEmail

	// Retry
	c.Retry.Enabled = c.RetryEnabled
	c.Retry.Max = c.RetryMax
	c.Retry.IntervalSec = c.RetryIntervalSec
	c.Retry.MaxIntervalSec = c.RetryMaxIntervalSec
	c.Retry.On5xx = c.RetryOn5xx
	c.Retry.OnNetworkError = c.RetryOnNetworkError
	c.Retry.DialTimeoutSec = c.DialTimeoutSec
	c.Retry.TLSHandshakeTimeoutSec = c.TLSHandshakeTimeoutSec
	c.Retry.ResponseHeaderTimeoutSec = c.ResponseHeaderTimeoutSec
	c.Retry.ExpectContinueTimeoutSec = c.ExpectContinueTimeoutSec

	// RateLimit
	c.RateLimit.Enabled = c.RateLimitEnabled
	c.RateLimit.RPS = c.RateLimitRPS
	c.RateLimit.Burst = c.RateLimitBurst
	c.RateLimit.UsageResetIntervalHours = c.UsageResetIntervalHours
	c.RateLimit.UsageResetTimezone = c.UsageResetTimezone
	c.RateLimit.UsageResetHourLocal = c.UsageResetHourLocal

	// APICompat
	c.APICompat.OpenAIImagesIncludeMIME = c.OpenAIImagesIncludeMIME
	c.APICompat.ToolArgsDeltaChunk = c.ToolArgsDeltaChunk
	c.APICompat.PreferredBaseModels = c.PreferredBaseModels
	c.APICompat.DisabledModels = c.DisabledModels
	c.APICompat.DisableModelVariants = c.DisableModelVariants

	// ResponseShaping
	c.ResponseShaping.AntiTruncationMax = c.AntiTruncationMax
	c.ResponseShaping.AntiTruncationEnabled = c.AntiTruncationEnabled
	c.ResponseShaping.FakeStreamingEnabled = c.FakeStreamingEnabled
	c.ResponseShaping.FakeStreamingChunkSize = c.FakeStreamingChunkSize
	c.ResponseShaping.FakeStreamingDelayMs = c.FakeStreamingDelayMs
	c.ResponseShaping.AutoImagePlaceholder = c.AutoImagePlaceholder
	c.ResponseShaping.RequestLogEnabled = c.RequestLogEnabled
	c.ResponseShaping.PprofEnabled = c.PprofEnabled
	c.ResponseShaping.ProxyURL = c.ProxyURL
	c.ResponseShaping.SanitizerEnabled = c.SanitizerEnabled
	c.ResponseShaping.SanitizerPatterns = c.SanitizerPatterns

	// OAuth
	c.OAuth.ClientID = c.OAuthClientID
	c.OAuth.ClientSecret = c.OAuthClientSecret
	c.OAuth.RedirectURL = c.OAuthRedirectURL
	c.OAuth.RefreshAheadSeconds = c.RefreshAheadSeconds
	c.OAuth.RefreshSingleflightTimeoutSec = c.RefreshSingleflightTimeoutSec

	// AutoBan
	c.AutoBan.Enabled = c.AutoBanEnabled
	c.AutoBan.Ban429Threshold = c.AutoBan429Threshold
	c.AutoBan.Ban403Threshold = c.AutoBan403Threshold
	c.AutoBan.Ban401Threshold = c.AutoBan401Threshold
	c.AutoBan.Ban5xxThreshold = c.AutoBan5xxThreshold
	c.AutoBan.ConsecutiveFails = c.AutoBanConsecutiveFails
	c.AutoBan.RecoveryEnabled = c.AutoRecoveryEnabled
	c.AutoBan.RecoveryIntervalMin = c.AutoRecoveryIntervalMin

	// AutoProbe
	c.AutoProbe.Enabled = c.AutoProbeEnabled
	c.AutoProbe.HourUTC = c.AutoProbeHourUTC
	c.AutoProbe.Model = c.AutoProbeModel
	c.AutoProbe.TimeoutSec = c.AutoProbeTimeoutSec
	c.AutoProbe.DisableThresholdPct = c.AutoProbeDisableThresholdPct

	// Routing
	c.Routing.StickyTTLSeconds = c.StickyTTLSeconds
	c.Routing.CooldownBaseMS = c.RouterCooldownBaseMS
	c.Routing.CooldownMaxMS = c.RouterCooldownMaxMS
	c.Routing.PersistState = c.PersistRoutingState
	c.Routing.PersistIntervalSec = c.RoutingPersistIntervalSec
	c.Routing.DebugHeaders = c.RoutingDebugHeaders
}

// Load loads configuration from file and environment
func Load() *Config {
	return LoadWithFile("")
}

// LoadWithFile loads configuration from specified file path
func LoadWithFile(configPath string) *Config {
	// Initialize global config manager once
	configOnce.Do(func() {
		var err error
		globalConfigManager, err = NewConfigManager(configPath)
		if err != nil {
			// Fall back to environment-only config
			globalConfigManager = nil
		}
	})

	// Get config from manager if available
	if globalConfigManager != nil {
		fc := globalConfigManager.GetConfig()
		return fileConfigToConfig(fc)
	}

	// Fall back to environment-only config
	return loadFromEnv()
}

// GetConfigManager returns the global config manager
func GetConfigManager() *ConfigManager {
	return globalConfigManager
}

// UpdateConfig updates configuration dynamically
func UpdateConfig(updates map[string]interface{}) error {
	if globalConfigManager == nil {
		return fmt.Errorf("config manager not initialized")
	}
	return globalConfigManager.UpdateConfig(updates)
}
