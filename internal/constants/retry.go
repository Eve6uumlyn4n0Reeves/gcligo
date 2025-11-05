package constants

import "time"

// 重试策略常量
const (
	DefaultMaxRetries    = 3
	DefaultRetryInterval = 1 * time.Second
	DefaultMaxRetryDelay = 30 * time.Second
	RetryBackoffFactor   = 2.0

	// 特定错误类型的重试延迟
	RateLimitRetryDelay          = 60 * time.Second // 429错误
	ServiceUnavailableRetryDelay = 30 * time.Second // 503错误
	GatewayErrorRetryDelay       = 15 * time.Second // 502/504错误
	DefaultErrorRetryDelay       = 5 * time.Second  // 其他错误

	// 网络错误重试配置
	NetworkErrorMaxRetries = 5
	NetworkErrorBaseDelay  = 2 * time.Second

	// 上游请求重试配置
	UpstreamMaxRetries    = 3
	UpstreamRetryDelay    = 1 * time.Second
	UpstreamMaxRetryDelay = 10 * time.Second
)

// 错误阈值常量
const (
	// 自动封禁阈值
	DefaultAutoBan429Threshold     = 3
	DefaultAutoBan403Threshold     = 5
	DefaultAutoBan401Threshold     = 3
	DefaultAutoBanConsecutiveFails = 10

	// 自动恢复配置
	DefaultAutoRecoveryIntervalMin = 10
	AutoRecoveryHealthThreshold    = 0.7 // 健康分数阈值

	// 健康检查配置
	HealthCheckInterval = 1 * time.Minute
	HealthCheckTimeout  = 10 * time.Second
	CredentialHealthTTL = 5 * time.Minute
)

// 错误处理配置
const (
	MaxErrorMessageLength   = 200
	ErrorContextMaxLength   = 500
	ErrorStackTraceMaxDepth = 10

	// 错误分类权重
	CriticalErrorWeight = 3.0
	MajorErrorWeight    = 2.0
	MinorErrorWeight    = 1.0
	WarningWeight       = 0.5
)
