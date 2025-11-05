package constants

import "time"

// HTTP Client 连接池配置 - 针对"大水管"场景优化
const (
	// 基础连接池设置（保守配置）
	BaseMaxIdleConns        = 4096
	BaseMaxIdleConnsPerHost = 4096
	BaseIdleConnTimeout     = 90 * time.Second

	// 高性能连接池设置（大水管优化）
	HighThroughputMaxIdleConns        = 8192 // 全局最大空闲连接
	HighThroughputMaxIdleConnsPerHost = 512  // 每主机最大空闲连接
	HighThroughputMaxConnsPerHost     = 1024 // 每主机最大总连接
	HighThroughputIdleConnTimeout     = 120 * time.Second

	// 缓冲区大小
	DefaultWriteBufferSize = 64 * 1024 // 64KB 写缓冲
	DefaultReadBufferSize  = 64 * 1024 // 64KB 读缓冲

	// Keep-Alive 设置
	DefaultKeepAlive = 30 * time.Second
)

// HTTP 超时配置
const (
	DefaultDialTimeout           = 10 * time.Second
	DefaultTLSHandshakeTimeout   = 10 * time.Second
	DefaultResponseHeaderTimeout = 60 * time.Second
	DefaultExpectContinueTimeout = 2 * time.Second

	// 高性能场景的超时设置
	HighThroughputDialTimeout           = 5 * time.Second
	HighThroughputTLSHandshakeTimeout   = 8 * time.Second
	HighThroughputResponseHeaderTimeout = 30 * time.Second
)

// 代理配置
const (
	ProxyConnectTimeout = 10 * time.Second
	ProxyReadTimeout    = 30 * time.Second
)

// TransportConfig 定义传输层配置选项
type TransportConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	IdleConnTimeout     time.Duration
	WriteBufferSize     int
	ReadBufferSize      int
	EnableHTTP2         bool
	DisableCompression  bool
	DualStack           bool
}

// GetBaseTransportConfig 返回基础传输配置
func GetBaseTransportConfig() TransportConfig {
	return TransportConfig{
		MaxIdleConns:        BaseMaxIdleConns,
		MaxIdleConnsPerHost: BaseMaxIdleConnsPerHost,
		MaxConnsPerHost:     0, // 不限制
		IdleConnTimeout:     BaseIdleConnTimeout,
		WriteBufferSize:     0, // 使用默认
		ReadBufferSize:      0, // 使用默认
		EnableHTTP2:         false,
		DisableCompression:  false,
		DualStack:           false,
	}
}

// GetHighThroughputTransportConfig 返回高吞吐量传输配置
func GetHighThroughputTransportConfig() TransportConfig {
	return TransportConfig{
		MaxIdleConns:        HighThroughputMaxIdleConns,
		MaxIdleConnsPerHost: HighThroughputMaxIdleConnsPerHost,
		MaxConnsPerHost:     HighThroughputMaxConnsPerHost,
		IdleConnTimeout:     HighThroughputIdleConnTimeout,
		WriteBufferSize:     DefaultWriteBufferSize,
		ReadBufferSize:      DefaultReadBufferSize,
		EnableHTTP2:         true,
		DisableCompression:  false,
		DualStack:           true,
	}
}
