package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SlowQueryThreshold 慢查询阈值
const SlowQueryThreshold = 100 * time.Millisecond

// SlowQueryLogger 慢查询日志记录器
type SlowQueryLogger struct {
	mu        sync.RWMutex
	threshold time.Duration
	enabled   bool
	queries   []SlowQuery
	maxSize   int
}

// SlowQuery 慢查询记录
type SlowQuery struct {
	Timestamp time.Time     `json:"timestamp"`
	Operation string        `json:"operation"`
	Duration  time.Duration `json:"duration"`
	Details   string        `json:"details"`
	Stack     string        `json:"stack,omitempty"`
}

// NewSlowQueryLogger 创建慢查询日志记录器
func NewSlowQueryLogger(threshold time.Duration, maxSize int) *SlowQueryLogger {
	if threshold <= 0 {
		threshold = SlowQueryThreshold
	}
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &SlowQueryLogger{
		threshold: threshold,
		enabled:   true,
		queries:   make([]SlowQuery, 0, maxSize),
		maxSize:   maxSize,
	}
}

// Enable 启用慢查询日志
func (l *SlowQueryLogger) Enable() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = true
}

// Disable 禁用慢查询日志
func (l *SlowQueryLogger) Disable() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = false
}

// IsEnabled 检查是否启用
func (l *SlowQueryLogger) IsEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.enabled
}

// SetThreshold 设置慢查询阈值
func (l *SlowQueryLogger) SetThreshold(threshold time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.threshold = threshold
}

// GetThreshold 获取慢查询阈值
func (l *SlowQueryLogger) GetThreshold() time.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.threshold
}

// Track 跟踪操作执行时间
func (l *SlowQueryLogger) Track(ctx context.Context, operation string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	if l.IsEnabled() && duration >= l.GetThreshold() {
		l.Log(SlowQuery{
			Timestamp: start,
			Operation: operation,
			Duration:  duration,
			Details:   fmt.Sprintf("error: %v", err),
		})
	}

	return err
}

// TrackWithDetails 跟踪操作执行时间（带详细信息）
func (l *SlowQueryLogger) TrackWithDetails(ctx context.Context, operation string, details string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	if l.IsEnabled() && duration >= l.GetThreshold() {
		l.Log(SlowQuery{
			Timestamp: start,
			Operation: operation,
			Duration:  duration,
			Details:   details,
		})
	}

	return err
}

// Log 记录慢查询
func (l *SlowQueryLogger) Log(query SlowQuery) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.enabled {
		return
	}

	// 如果超过最大大小，移除最旧的记录
	if len(l.queries) >= l.maxSize {
		l.queries = l.queries[1:]
	}

	l.queries = append(l.queries, query)
}

// GetQueries 获取所有慢查询记录
func (l *SlowQueryLogger) GetQueries() []SlowQuery {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 返回副本
	result := make([]SlowQuery, len(l.queries))
	copy(result, l.queries)
	return result
}

// GetRecentQueries 获取最近的N条慢查询记录
func (l *SlowQueryLogger) GetRecentQueries(n int) []SlowQuery {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if n <= 0 || n > len(l.queries) {
		n = len(l.queries)
	}

	start := len(l.queries) - n
	result := make([]SlowQuery, n)
	copy(result, l.queries[start:])
	return result
}

// Clear 清空慢查询记录
func (l *SlowQueryLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.queries = make([]SlowQuery, 0, l.maxSize)
}

// GetStats 获取慢查询统计信息
func (l *SlowQueryLogger) GetStats() SlowQueryStats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.queries) == 0 {
		return SlowQueryStats{
			Count:     0,
			Threshold: l.threshold,
		}
	}

	var totalDuration time.Duration
	var maxDuration time.Duration
	var minDuration time.Duration = time.Hour * 24 // 初始化为一个大值

	operationCounts := make(map[string]int)

	for _, q := range l.queries {
		totalDuration += q.Duration
		if q.Duration > maxDuration {
			maxDuration = q.Duration
		}
		if q.Duration < minDuration {
			minDuration = q.Duration
		}
		operationCounts[q.Operation]++
	}

	avgDuration := totalDuration / time.Duration(len(l.queries))

	return SlowQueryStats{
		Count:           len(l.queries),
		Threshold:       l.threshold,
		AvgDuration:     avgDuration,
		MaxDuration:     maxDuration,
		MinDuration:     minDuration,
		OperationCounts: operationCounts,
	}
}

// SlowQueryStats 慢查询统计信息
type SlowQueryStats struct {
	Count           int            `json:"count"`
	Threshold       time.Duration  `json:"threshold"`
	AvgDuration     time.Duration  `json:"avg_duration"`
	MaxDuration     time.Duration  `json:"max_duration"`
	MinDuration     time.Duration  `json:"min_duration"`
	OperationCounts map[string]int `json:"operation_counts"`
}

// 全局慢查询日志记录器
var globalSlowQueryLogger = NewSlowQueryLogger(SlowQueryThreshold, 1000)

// GetGlobalSlowQueryLogger 获取全局慢查询日志记录器
func GetGlobalSlowQueryLogger() *SlowQueryLogger {
	return globalSlowQueryLogger
}

// TrackSlowQuery 使用全局记录器跟踪慢查询
func TrackSlowQuery(ctx context.Context, operation string, fn func() error) error {
	return globalSlowQueryLogger.Track(ctx, operation, fn)
}

// TrackSlowQueryWithDetails 使用全局记录器跟踪慢查询（带详细信息）
func TrackSlowQueryWithDetails(ctx context.Context, operation string, details string, fn func() error) error {
	return globalSlowQueryLogger.TrackWithDetails(ctx, operation, details, fn)
}
