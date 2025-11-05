package management

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// BatchLimitConfig 定义批量接口的限流配置。
type BatchLimitConfig struct {
	Enabled              bool
	MaxIDsPerRequest     int
	RequestsPerMinute    int
	MaxOperationsPerHour int
}

// DefaultBatchLimitConfig 提供默认限流配置。
var DefaultBatchLimitConfig = BatchLimitConfig{
	Enabled:              true,
	MaxIDsPerRequest:     500,
	RequestsPerMinute:    20,
	MaxOperationsPerHour: 10000,
}

// BatchLimiter 提供批量接口级别的限流能力。
type BatchLimiter struct {
	cfg BatchLimitConfig

	requestLimiter   *rate.Limiter
	operationCounter *slidingWindowCounter
}

// NewBatchLimiter 构建限流器。
func NewBatchLimiter(cfg BatchLimitConfig) *BatchLimiter {
	if !cfg.Enabled {
		return &BatchLimiter{cfg: cfg}
	}

	requestRate := rate.Limit(float64(cfg.RequestsPerMinute) / 60.0)
	requestLimiter := rate.NewLimiter(requestRate, cfg.RequestsPerMinute)
	operationCounter := newSlidingWindowCounter(time.Hour, cfg.MaxOperationsPerHour)

	return &BatchLimiter{
		cfg:              cfg,
		requestLimiter:   requestLimiter,
		operationCounter: operationCounter,
	}
}

// CheckRequest 执行限流检查，返回是否通过、错误信息以及建议的重试等待时间。
func (bl *BatchLimiter) CheckRequest(operation string, count int) (bool, string, time.Duration) {
	if !bl.cfg.Enabled {
		return true, "", 0
	}

	if count <= 0 {
		return false, "ids array cannot be empty", 0
	}

	if count > bl.cfg.MaxIDsPerRequest {
		msg := fmt.Sprintf(
			"batch size %d exceeds maximum %d; split the request into smaller chunks",
			count, bl.cfg.MaxIDsPerRequest,
		)
		log.Warnf("Batch %s rejected: %s", operation, msg)
		return false, msg, 0
	}

	if !bl.requestLimiter.Allow() {
		res := bl.requestLimiter.Reserve()
		delay := res.Delay()
		res.Cancel()
		if delay <= 0 {
			delay = time.Second * 3
		}
		msg := fmt.Sprintf(
			"rate limit exceeded (%d requests/minute); retry after %s",
			bl.cfg.RequestsPerMinute, delay.Round(time.Second),
		)
		log.Warnf("Batch %s throttled: %s", operation, msg)
		return false, msg, delay
	}

	if !bl.operationCounter.Allow(count) {
		current := bl.operationCounter.Current()
		msg := fmt.Sprintf(
			"operation quota exceeded: %d/%d operations in the last hour",
			current, bl.cfg.MaxOperationsPerHour,
		)
		log.Warnf("Batch %s quota exceeded: %s", operation, msg)
		return false, msg, 10 * time.Minute
	}

	return true, "", 0
}

// RecordSuccess 记录一次批量操作成功完成，便于统计。
func (bl *BatchLimiter) RecordSuccess(operation string, count int) {
	if !bl.cfg.Enabled {
		return
	}
	log.Debugf("Batch %s completed successfully (%d items)", operation, count)
}

// slidingWindowCounter 通过滑动窗口统计操作数。
type slidingWindowCounter struct {
	window   time.Duration
	maxCount int

	mu      sync.Mutex
	records []timestampedCount
}

type timestampedCount struct {
	ts    time.Time
	count int
}

func newSlidingWindowCounter(window time.Duration, maxCount int) *slidingWindowCounter {
	return &slidingWindowCounter{
		window:   window,
		maxCount: maxCount,
		records:  make([]timestampedCount, 0),
	}
}

func (swc *slidingWindowCounter) Allow(count int) bool {
	if count <= 0 {
		return false
	}

	swc.mu.Lock()
	defer swc.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-swc.window)

	total := 0
	filtered := swc.records[:0]
	for _, rec := range swc.records {
		if rec.ts.After(threshold) {
			filtered = append(filtered, rec)
			total += rec.count
		}
	}
	swc.records = filtered

	if total+count > swc.maxCount {
		return false
	}

	swc.records = append(swc.records, timestampedCount{ts: now, count: count})
	return true
}

func (swc *slidingWindowCounter) Current() int {
	swc.mu.Lock()
	defer swc.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-swc.window)
	total := 0
	for _, rec := range swc.records {
		if rec.ts.After(threshold) {
			total += rec.count
		}
	}
	return total
}
