package config

// ServerConfig 服务器和端点配置
type ServerConfig struct {
	OpenAIPort      string
	GeminiPort      string
	BasePath        string
	WebAdminEnabled bool
	RunProfile      string
}

// UpstreamConfig 上游凭证和提供商配置
type UpstreamConfig struct {
	OpenAIKey        string
	GeminiKey        string
	CodeAssist       string
	GoogleToken      string
	GoogleProjID     string
	UpstreamProvider string
}

// SecurityConfig 安全和管理访问配置
type SecurityConfig struct {
	ManagementKey            string
	ManagementKeyHash        string
	ManagementReadOnly       bool
	ManagementAllowRemote    bool
	ManagementRemoteTTlHours int
	ManagementRemoteAllowIPs []string
	AuthDir                  string
	HeaderPassThrough        bool
	Debug                    bool
	LogFile                  string
}

// ExecutionConfig 执行控制配置
type ExecutionConfig struct {
	CallsPerRotation           int
	MaxConcurrentPerCredential int
	AutoLoadEnvCreds           bool
}

// StorageConfig 存储后端配置
type StorageConfig struct {
	Backend        string // file, redis, mongodb, postgres
	BaseDir        string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	RedisPrefix    string
	MongoURI       string
	MongoDatabase  string
	PostgresDSN    string
	GitRemoteURL   string
	GitBranch      string
	GitUsername    string
	GitPassword    string
	GitAuthorName  string
	GitAuthorEmail string
}

// RetryConfig 重试和超时设置
type RetryConfig struct {
	Enabled                  bool
	Max                      int
	IntervalSec              int
	MaxIntervalSec           int
	On5xx                    bool
	OnNetworkError           bool
	DialTimeoutSec           int
	TLSHandshakeTimeoutSec   int
	ResponseHeaderTimeoutSec int
	ExpectContinueTimeoutSec int
}

// RateLimitConfig 速率限制和使用重置配置
type RateLimitConfig struct {
	Enabled                 bool
	RPS                     int
	Burst                   int
	UsageResetIntervalHours int
	UsageResetTimezone      string
	UsageResetHourLocal     int
}

// APICompatConfig API 兼容性配置
type APICompatConfig struct {
	OpenAIImagesIncludeMIME bool
	ToolArgsDeltaChunk      int
	PreferredBaseModels     []string
	DisabledModels          []string
	DisableModelVariants    bool
}

// ResponseShapingConfig 响应塑形和流式处理配置
type ResponseShapingConfig struct {
	AntiTruncationMax      int
	AntiTruncationEnabled  bool
	FakeStreamingEnabled   bool
	FakeStreamingChunkSize int
	FakeStreamingDelayMs   int
	AutoImagePlaceholder   bool
	RequestLogEnabled      bool
	PprofEnabled           bool
	ProxyURL               string
	SanitizerEnabled       bool
	SanitizerPatterns      []string
}

// OAuthConfig OAuth 客户端凭证配置
type OAuthConfig struct {
	ClientID                      string
	ClientSecret                  string
	RedirectURL                   string
	RefreshAheadSeconds           int
	RefreshSingleflightTimeoutSec int
}

// AutoBanConfig 自动禁用和恢复配置
type AutoBanConfig struct {
	Enabled             bool
	Ban429Threshold     int
	Ban403Threshold     int
	Ban401Threshold     int
	Ban5xxThreshold     int
	ConsecutiveFails    int
	RecoveryEnabled     bool
	RecoveryIntervalMin int
}

// AutoProbeConfig 自动探测（活性检查）配置
type AutoProbeConfig struct {
	Enabled             bool
	HourUTC             int
	Model               string
	TimeoutSec          int
	DisableThresholdPct int
}

// RoutingConfig 路由策略配置
type RoutingConfig struct {
	StickyTTLSeconds   int
	CooldownBaseMS     int
	CooldownMaxMS      int
	PersistState       bool
	PersistIntervalSec int
	DebugHeaders       bool
}
