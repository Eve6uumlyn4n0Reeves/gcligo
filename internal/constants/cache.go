package constants

import "time"

// 缓存相关常量
const (
	// API 缓存配置
	APICacheTTL           = 30 * time.Second
	ModelRegistryCacheTTL = 5 * time.Minute
	CredentialsCacheTTL   = 1 * time.Minute
	ConfigCacheTTL        = 2 * time.Minute

	// 上游模型发现缓存
	UpstreamDiscoveryTTL     = 30 * time.Minute
	UpstreamDiscoveryTimeout = 20 * time.Second
	ForceRefreshTimeout      = 30 * time.Second

	// WebSocket 日志缓存
	WSLogBufferSize    = 100
	WSLogRetentionTime = 1 * time.Hour

	// 前端缓存配置
	FrontendCacheSize = 1000 // 最大缓存条目数
	FrontendCacheTTL  = 5 * time.Minute
)

// 批量操作配置
const (
	DefaultBatchSize    = 10
	DefaultBatchDelay   = 100 * time.Millisecond
	MaxBatchSize        = 100
	BatchTimeoutPerItem = 5 * time.Second

	// 批量凭证操作
	CredentialBatchSize  = 5
	CredentialBatchDelay = 200 * time.Millisecond

	// 批量模型操作
	ModelRegistryBatchSize  = 20
	ModelRegistryBatchDelay = 50 * time.Millisecond
)

// 缓存清理配置
const (
	CacheCleanupInterval = 10 * time.Minute
	CacheMaxMemoryMB     = 100 // 最大缓存内存使用（MB）
)
