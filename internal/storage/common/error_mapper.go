package common

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// ErrorMapper 提供统一的错误映射功能
type ErrorMapper struct{}

// NewErrorMapper 创建新的错误映射器
func NewErrorMapper() *ErrorMapper {
	return &ErrorMapper{}
}

// ErrNotFound 表示资源未找到
type ErrNotFound struct {
	Key string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("not found: %s", e.Key)
}

// ErrAlreadyExists 表示资源已存在
type ErrAlreadyExists struct {
	Key string
}

func (e *ErrAlreadyExists) Error() string {
	return fmt.Sprintf("already exists: %s", e.Key)
}

// ErrInvalidData 表示数据无效
type ErrInvalidData struct {
	Reason string
}

func (e *ErrInvalidData) Error() string {
	return fmt.Sprintf("invalid data: %s", e.Reason)
}

// MapRedisError 将 Redis 错误映射为通用错误
func (m *ErrorMapper) MapRedisError(err error, key string) error {
	if err == nil {
		return nil
	}

	if err == redis.Nil {
		return &ErrNotFound{Key: key}
	}

	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("operation canceled: %w", err)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("operation timeout: %w", err)
	}

	return err
}

// MapMongoError 将 MongoDB 错误映射为通用错误
func (m *ErrorMapper) MapMongoError(err error, key string) error {
	if err == nil {
		return nil
	}

	if err == mongo.ErrNoDocuments {
		return &ErrNotFound{Key: key}
	}

	if mongo.IsDuplicateKeyError(err) {
		return &ErrAlreadyExists{Key: key}
	}

	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("operation canceled: %w", err)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("operation timeout: %w", err)
	}

	return err
}

// MapPostgresError 将 PostgreSQL 错误映射为通用错误
func (m *ErrorMapper) MapPostgresError(err error, key string) error {
	if err == nil {
		return nil
	}

	// PostgreSQL 特定错误处理
	errMsg := err.Error()

	// 检查是否是 "no rows" 错误
	if errMsg == "sql: no rows in result set" {
		return &ErrNotFound{Key: key}
	}

	// 检查是否是唯一约束冲突
	if contains(errMsg, "duplicate key") || contains(errMsg, "unique constraint") {
		return &ErrAlreadyExists{Key: key}
	}

	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("operation canceled: %w", err)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("operation timeout: %w", err)
	}

	return err
}

// IsNotFound 检查错误是否为 NotFound 类型
func (m *ErrorMapper) IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var notFoundErr *ErrNotFound
	return errors.As(err, &notFoundErr)
}

// IsAlreadyExists 检查错误是否为 AlreadyExists 类型
func (m *ErrorMapper) IsAlreadyExists(err error) bool {
	if err == nil {
		return false
	}

	var existsErr *ErrAlreadyExists
	return errors.As(err, &existsErr)
}

// IsInvalidData 检查错误是否为 InvalidData 类型
func (m *ErrorMapper) IsInvalidData(err error) bool {
	if err == nil {
		return false
	}

	var invalidErr *ErrInvalidData
	return errors.As(err, &invalidErr)
}

// WrapError 包装错误并添加上下文信息
func (m *ErrorMapper) WrapError(err error, operation, resource string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s %s: %w", operation, resource, err)
}

// contains 检查字符串是否包含子串（辅助函数）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
