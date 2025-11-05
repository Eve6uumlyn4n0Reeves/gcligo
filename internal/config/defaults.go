package config

import (
	"strings"

	"gcli2api-go/internal/constants"
)

// DefaultValues centralizes all default configuration values
// This ensures consistency between config.example.yaml and code defaults
type DefaultValues struct {
	// Server Configuration
	OpenAIPort      string
	GeminiPort      string
	WebAdminEnabled bool
	BasePath        string

	// Authentication & Security
	AuthDir            string
	CallsPerRotation   int
	ManagementReadOnly bool

	// Storage Configuration
	StorageBackend string
	StorageBaseDir string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	RedisPrefix    string
	MongoDatabase  string
	GitBranch      string
	GitAuthorName  string
	GitAuthorEmail string

	// Retry Configuration
	RetryEnabled        bool
	RetryMax            int
	RetryIntervalSec    int
	RetryMaxIntervalSec int
	RetryOn5xx          bool
	RetryOnNetworkError bool

	// Anti-Truncation
	AntiTruncationMax     int
	AntiTruncationEnabled bool

	// Fake Streaming
	FakeStreamingEnabled   bool
	FakeStreamingChunkSize int
	FakeStreamingDelayMs   int

	// Rate Limiting
	RateLimitEnabled bool
	RateLimitRPS     int
	RateLimitBurst   int

	// HTTP Timeouts (seconds)
	DialTimeoutSec           int
	TLSHandshakeTimeoutSec   int
	ResponseHeaderTimeoutSec int
	ExpectContinueTimeoutSec int

	// Auto-Ban Configuration
	AutoBanEnabled          bool
	AutoBan429Threshold     int
	AutoBan403Threshold     int
	AutoBan401Threshold     int
	AutoBan5xxThreshold     int
	AutoBanConsecutiveFails int

	// Auto-Recovery Configuration
	AutoRecoveryEnabled     bool
	AutoRecoveryIntervalMin int

	// Auto Probe Configuration
	AutoProbeEnabled    bool
	AutoProbeHourUTC    int
	AutoProbeModel      string
	AutoProbeTimeoutSec int

	// Quota Management
	QuotaManagementEnabled bool
	DefaultDailyLimit      int64
	QuotaResetHour         int

	// Usage Statistics
	UsageStatsEnabled       bool
	UsageResetIntervalHours int
	UsageResetTimezone      string
	UsageResetHourLocal     int

	// Advanced Features
	HeaderPassThrough    bool
	AutoImagePlaceholder bool
	ToolArgsDeltaChunk   int
	SanitizerEnabled     bool
	SanitizerPatterns    []string

	// Model Configuration
	PreferredBaseModels []string
	DisabledModels      []string

	// Upstream Configuration
	CodeAssistEndpoint string
	UpstreamProvider   string
}

// GetDefaults returns the centralized default configuration values
func GetDefaults() DefaultValues {
	return DefaultValues{
		// Server Configuration
		OpenAIPort:      "8317",
		GeminiPort:      "8318",
		WebAdminEnabled: true,
		BasePath:        "",

		// Authentication & Security
		AuthDir:            "~/.gcli2api/auths",
		CallsPerRotation:   10,
		ManagementReadOnly: false,

		// Storage Configuration
		StorageBackend: "file",
		StorageBaseDir: "~/.gcli2api/storage",
		RedisAddr:      "localhost:6379",
		RedisPassword:  "",
		RedisDB:        0,
		RedisPrefix:    "gcli2api:",
		MongoDatabase:  "gcli2api",
		GitBranch:      "main",
		GitAuthorName:  "gcli2api",
		GitAuthorEmail: "gcli2api@example.local",

		// Retry Configuration
		RetryEnabled:        true,
		RetryMax:            3,
		RetryIntervalSec:    1,
		RetryMaxIntervalSec: 8,
		RetryOn5xx:          true,
		RetryOnNetworkError: true,

		// Anti-Truncation
		AntiTruncationMax:     3,
		AntiTruncationEnabled: false,

		// Fake Streaming
		FakeStreamingEnabled:   false,
		FakeStreamingChunkSize: 20,
		FakeStreamingDelayMs:   50,

		// Rate Limiting
		RateLimitEnabled: false,
		RateLimitRPS:     100,
		RateLimitBurst:   200,

		// HTTP Timeouts (seconds)
		DialTimeoutSec:           int(constants.DefaultDialTimeout.Seconds()),
		TLSHandshakeTimeoutSec:   int(constants.DefaultTLSHandshakeTimeout.Seconds()),
		ResponseHeaderTimeoutSec: int(constants.DefaultResponseHeaderTimeout.Seconds()),
		ExpectContinueTimeoutSec: int(constants.DefaultExpectContinueTimeout.Seconds()),

		// Auto-Ban Configuration
		AutoBanEnabled:          true,
		AutoBan429Threshold:     3,
		AutoBan403Threshold:     5,
		AutoBan401Threshold:     3,
		AutoBan5xxThreshold:     10,
		AutoBanConsecutiveFails: 10,

		// Auto-Recovery Configuration
		AutoRecoveryEnabled:     true,
		AutoRecoveryIntervalMin: 10,

		// Auto Probe Configuration
		AutoProbeEnabled:    true,
		AutoProbeHourUTC:    7,
		AutoProbeModel:      "gemini-2.5-flash",
		AutoProbeTimeoutSec: 10,

		// Quota Management
		QuotaManagementEnabled: true,
		DefaultDailyLimit:      1000,
		QuotaResetHour:         0,

		// Usage Statistics
		UsageStatsEnabled:       true,
		UsageResetIntervalHours: 24,
		UsageResetTimezone:      "UTC+7",
		UsageResetHourLocal:     0,

		// Advanced Features
		HeaderPassThrough:    false,
		AutoImagePlaceholder: true,
		ToolArgsDeltaChunk:   0,
		SanitizerEnabled:     false,
		SanitizerPatterns:    nil,

		// Model Configuration
		PreferredBaseModels: []string{
			"gemini-2.5-pro",
			"gemini-2.5-flash",
			"gemini-2.5-flash-image",
			"gemini-2.5-flash-image-preview",
		},
		DisabledModels: []string{},

		// Upstream Configuration
		CodeAssistEndpoint: "https://cloudcode-pa.googleapis.com",
		UpstreamProvider:   "gemini",
	}
}

// ApplyDefaults applies default values to a Config struct
func (c *Config) ApplyDefaults() {
	defaults := GetDefaults()

	if c.OpenAIPort == "" {
		c.OpenAIPort = defaults.OpenAIPort
	}
	if c.GeminiPort == "" {
		c.GeminiPort = defaults.GeminiPort
	}
	if c.AuthDir == "" {
		c.AuthDir = defaults.AuthDir
	}
	if c.StorageBackend == "" {
		c.StorageBackend = defaults.StorageBackend
	}
	if c.StorageBaseDir == "" {
		c.StorageBaseDir = defaults.StorageBaseDir
	}
	if c.CodeAssist == "" {
		c.CodeAssist = defaults.CodeAssistEndpoint
	}
	if c.CallsPerRotation == 0 {
		c.CallsPerRotation = defaults.CallsPerRotation
	}
	if c.RetryMax == 0 {
		c.RetryMax = defaults.RetryMax
	}
	if c.RetryIntervalSec == 0 {
		c.RetryIntervalSec = defaults.RetryIntervalSec
	}
	if c.RetryMaxIntervalSec == 0 {
		c.RetryMaxIntervalSec = defaults.RetryMaxIntervalSec
	}
	if c.AntiTruncationMax == 0 {
		c.AntiTruncationMax = defaults.AntiTruncationMax
	}
	if c.FakeStreamingChunkSize == 0 {
		c.FakeStreamingChunkSize = defaults.FakeStreamingChunkSize
	}
	if c.FakeStreamingDelayMs == 0 {
		c.FakeStreamingDelayMs = defaults.FakeStreamingDelayMs
	}
	if c.RateLimitRPS == 0 {
		c.RateLimitRPS = defaults.RateLimitRPS
	}
	if c.RateLimitBurst == 0 {
		c.RateLimitBurst = defaults.RateLimitBurst
	}
	if c.DialTimeoutSec == 0 {
		c.DialTimeoutSec = defaults.DialTimeoutSec
	}
	if c.TLSHandshakeTimeoutSec == 0 {
		c.TLSHandshakeTimeoutSec = defaults.TLSHandshakeTimeoutSec
	}
	if c.ResponseHeaderTimeoutSec == 0 {
		c.ResponseHeaderTimeoutSec = defaults.ResponseHeaderTimeoutSec
	}
	if c.ExpectContinueTimeoutSec == 0 {
		c.ExpectContinueTimeoutSec = defaults.ExpectContinueTimeoutSec
	}
	if c.AutoBan429Threshold == 0 {
		c.AutoBan429Threshold = defaults.AutoBan429Threshold
	}
	if c.AutoBan403Threshold == 0 {
		c.AutoBan403Threshold = defaults.AutoBan403Threshold
	}
	if c.AutoBan401Threshold == 0 {
		c.AutoBan401Threshold = defaults.AutoBan401Threshold
	}
	if c.AutoBan5xxThreshold == 0 {
		c.AutoBan5xxThreshold = defaults.AutoBan5xxThreshold
	}
	if c.AutoBanConsecutiveFails == 0 {
		c.AutoBanConsecutiveFails = defaults.AutoBanConsecutiveFails
	}
	if c.AutoRecoveryIntervalMin == 0 {
		c.AutoRecoveryIntervalMin = defaults.AutoRecoveryIntervalMin
	}
	if c.AutoProbeHourUTC == 0 {
		c.AutoProbeHourUTC = defaults.AutoProbeHourUTC
	}
	if c.AutoProbeModel == "" {
		c.AutoProbeModel = defaults.AutoProbeModel
	}
	if c.AutoProbeTimeoutSec == 0 {
		c.AutoProbeTimeoutSec = defaults.AutoProbeTimeoutSec
	}
	if c.UsageResetIntervalHours == 0 {
		c.UsageResetIntervalHours = defaults.UsageResetIntervalHours
	}
	if strings.TrimSpace(c.UsageResetTimezone) == "" {
		c.UsageResetTimezone = defaults.UsageResetTimezone
	}
	if c.UsageResetHourLocal < 0 || c.UsageResetHourLocal > 23 {
		c.UsageResetHourLocal = defaults.UsageResetHourLocal
	}
	if c.SanitizerPatterns == nil {
		c.SanitizerPatterns = defaults.SanitizerPatterns
	}
	if len(c.PreferredBaseModels) == 0 {
		c.PreferredBaseModels = defaults.PreferredBaseModels
	}
	if c.UpstreamProvider == "" {
		c.UpstreamProvider = defaults.UpstreamProvider
	}
}
