package common

import (
	"context"
	"time"
)

// ErrNotSupported 表示操作不被支持
type ErrNotSupported struct {
	Operation string
}

func (e *ErrNotSupported) Error() string {
	return "operation not supported: " + e.Operation
}

// UnsupportedCacheOps 提供默认的"不支持"缓存操作实现
// 用于不支持缓存的存储后端（如 MongoDB, PostgreSQL）
//
// 使用方法：在后端结构体中嵌入此类型
//
//	type PostgresBackend struct {
//	    ...
//	    common.UnsupportedCacheOps
//	}
type UnsupportedCacheOps struct{}

// GetCache 返回不支持错误
func (u UnsupportedCacheOps) GetCache(ctx context.Context, key string) ([]byte, error) {
	return nil, &ErrNotSupported{Operation: "GetCache"}
}

// SetCache 返回不支持错误
func (u UnsupportedCacheOps) SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return &ErrNotSupported{Operation: "SetCache"}
}

// DeleteCache 返回不支持错误
func (u UnsupportedCacheOps) DeleteCache(ctx context.Context, key string) error {
	return &ErrNotSupported{Operation: "DeleteCache"}
}
