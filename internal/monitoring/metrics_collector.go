package monitoring

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	mu sync.RWMutex

	// 请求计数
	totalRequests   atomic.Int64
	successRequests atomic.Int64
	failedRequests  atomic.Int64

	// 响应时间
	totalDuration atomic.Int64 // 纳秒
	minDuration   atomic.Int64 // 纳秒
	maxDuration   atomic.Int64 // 纳秒

	// 按端点统计
	endpointStats map[string]*EndpointStats

	// 按状态码统计
	statusCodeStats map[int]*atomic.Int64

	// 时间窗口统计
	windowStats *WindowStats

	// 启动时间
	startTime time.Time
}

// EndpointStats 端点统计
type EndpointStats struct {
	Requests      atomic.Int64
	Success       atomic.Int64
	Failed        atomic.Int64
	TotalDuration atomic.Int64 // 纳秒
}

// WindowStats 时间窗口统计
type WindowStats struct {
	mu            sync.RWMutex
	windowSize    time.Duration
	buckets       []WindowBucket
	currentBucket int
}

// WindowBucket 时间窗口桶
type WindowBucket struct {
	Timestamp time.Time
	Requests  int64
	Success   int64
	Failed    int64
	Duration  int64 // 纳秒
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(windowSize time.Duration, bucketCount int) *MetricsCollector {
	if windowSize <= 0 {
		windowSize = time.Minute
	}
	if bucketCount <= 0 {
		bucketCount = 60
	}

	mc := &MetricsCollector{
		endpointStats:   make(map[string]*EndpointStats),
		statusCodeStats: make(map[int]*atomic.Int64),
		startTime:       time.Now(),
		windowStats: &WindowStats{
			windowSize: windowSize,
			buckets:    make([]WindowBucket, bucketCount),
		},
	}

	// 初始化最小持续时间为最大值
	mc.minDuration.Store(int64(time.Hour))

	return mc
}

// RecordRequest 记录请求
func (mc *MetricsCollector) RecordRequest(endpoint string, statusCode int, duration time.Duration, success bool) {
	// 更新总计数
	mc.totalRequests.Add(1)
	if success {
		mc.successRequests.Add(1)
	} else {
		mc.failedRequests.Add(1)
	}

	// 更新持续时间统计
	durationNs := duration.Nanoseconds()
	mc.totalDuration.Add(durationNs)

	// 更新最小持续时间
	for {
		oldMin := mc.minDuration.Load()
		if durationNs >= oldMin {
			break
		}
		if mc.minDuration.CompareAndSwap(oldMin, durationNs) {
			break
		}
	}

	// 更新最大持续时间
	for {
		oldMax := mc.maxDuration.Load()
		if durationNs <= oldMax {
			break
		}
		if mc.maxDuration.CompareAndSwap(oldMax, durationNs) {
			break
		}
	}

	// 更新端点统计
	mc.mu.Lock()
	stats, ok := mc.endpointStats[endpoint]
	if !ok {
		stats = &EndpointStats{}
		mc.endpointStats[endpoint] = stats
	}
	mc.mu.Unlock()

	stats.Requests.Add(1)
	if success {
		stats.Success.Add(1)
	} else {
		stats.Failed.Add(1)
	}
	stats.TotalDuration.Add(durationNs)

	// 更新状态码统计
	mc.mu.Lock()
	codeStats, ok := mc.statusCodeStats[statusCode]
	if !ok {
		codeStats = &atomic.Int64{}
		mc.statusCodeStats[statusCode] = codeStats
	}
	mc.mu.Unlock()

	codeStats.Add(1)

	// 更新时间窗口统计
	mc.windowStats.Record(success, durationNs)
}

// GetStats 获取统计信息
func (mc *MetricsCollector) GetStats() MetricsStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	totalReqs := mc.totalRequests.Load()
	successReqs := mc.successRequests.Load()
	failedReqs := mc.failedRequests.Load()
	totalDur := mc.totalDuration.Load()
	minDur := mc.minDuration.Load()
	maxDur := mc.maxDuration.Load()

	var avgDuration time.Duration
	if totalReqs > 0 {
		avgDuration = time.Duration(totalDur / totalReqs)
	}

	// 计算成功率
	var successRate float64
	if totalReqs > 0 {
		successRate = float64(successReqs) / float64(totalReqs) * 100
	}

	// 收集端点统计
	endpointStats := make(map[string]EndpointMetrics)
	for endpoint, stats := range mc.endpointStats {
		reqs := stats.Requests.Load()
		var avgDur time.Duration
		if reqs > 0 {
			avgDur = time.Duration(stats.TotalDuration.Load() / reqs)
		}

		endpointStats[endpoint] = EndpointMetrics{
			Requests:    reqs,
			Success:     stats.Success.Load(),
			Failed:      stats.Failed.Load(),
			AvgDuration: avgDur,
		}
	}

	// 收集状态码统计
	statusCodeStats := make(map[int]int64)
	for code, stats := range mc.statusCodeStats {
		statusCodeStats[code] = stats.Load()
	}

	return MetricsStats{
		TotalRequests:   totalReqs,
		SuccessRequests: successReqs,
		FailedRequests:  failedReqs,
		SuccessRate:     successRate,
		AvgDuration:     avgDuration,
		MinDuration:     time.Duration(minDur),
		MaxDuration:     time.Duration(maxDur),
		EndpointStats:   endpointStats,
		StatusCodeStats: statusCodeStats,
		Uptime:          time.Since(mc.startTime),
	}
}

// MetricsStats 指标统计
type MetricsStats struct {
	TotalRequests   int64                      `json:"total_requests"`
	SuccessRequests int64                      `json:"success_requests"`
	FailedRequests  int64                      `json:"failed_requests"`
	SuccessRate     float64                    `json:"success_rate"`
	AvgDuration     time.Duration              `json:"avg_duration"`
	MinDuration     time.Duration              `json:"min_duration"`
	MaxDuration     time.Duration              `json:"max_duration"`
	EndpointStats   map[string]EndpointMetrics `json:"endpoint_stats"`
	StatusCodeStats map[int]int64              `json:"status_code_stats"`
	Uptime          time.Duration              `json:"uptime"`
}

// EndpointMetrics 端点指标
type EndpointMetrics struct {
	Requests    int64         `json:"requests"`
	Success     int64         `json:"success"`
	Failed      int64         `json:"failed"`
	AvgDuration time.Duration `json:"avg_duration"`
}

// Record 记录时间窗口统计
func (ws *WindowStats) Record(success bool, durationNs int64) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	now := time.Now()
	bucket := &ws.buckets[ws.currentBucket]

	// 如果当前桶已过期，移动到下一个桶
	if now.Sub(bucket.Timestamp) > ws.windowSize/time.Duration(len(ws.buckets)) {
		ws.currentBucket = (ws.currentBucket + 1) % len(ws.buckets)
		bucket = &ws.buckets[ws.currentBucket]
		bucket.Timestamp = now
		bucket.Requests = 0
		bucket.Success = 0
		bucket.Failed = 0
		bucket.Duration = 0
	}

	bucket.Requests++
	if success {
		bucket.Success++
	} else {
		bucket.Failed++
	}
	bucket.Duration += durationNs
}

// GetWindowStats 获取时间窗口统计
func (ws *WindowStats) GetWindowStats() WindowMetrics {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	var totalRequests, totalSuccess, totalFailed, totalDuration int64

	for _, bucket := range ws.buckets {
		totalRequests += bucket.Requests
		totalSuccess += bucket.Success
		totalFailed += bucket.Failed
		totalDuration += bucket.Duration
	}

	var avgDuration time.Duration
	if totalRequests > 0 {
		avgDuration = time.Duration(totalDuration / totalRequests)
	}

	var successRate float64
	if totalRequests > 0 {
		successRate = float64(totalSuccess) / float64(totalRequests) * 100
	}

	return WindowMetrics{
		Requests:    totalRequests,
		Success:     totalSuccess,
		Failed:      totalFailed,
		SuccessRate: successRate,
		AvgDuration: avgDuration,
	}
}

// WindowMetrics 时间窗口指标
type WindowMetrics struct {
	Requests    int64         `json:"requests"`
	Success     int64         `json:"success"`
	Failed      int64         `json:"failed"`
	SuccessRate float64       `json:"success_rate"`
	AvgDuration time.Duration `json:"avg_duration"`
}

// Reset 重置指标收集器
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.totalRequests.Store(0)
	mc.successRequests.Store(0)
	mc.failedRequests.Store(0)
	mc.totalDuration.Store(0)
	mc.minDuration.Store(int64(time.Hour))
	mc.maxDuration.Store(0)

	mc.endpointStats = make(map[string]*EndpointStats)
	mc.statusCodeStats = make(map[int]*atomic.Int64)
	mc.startTime = time.Now()

	// 重置时间窗口统计
	mc.windowStats.mu.Lock()
	for i := range mc.windowStats.buckets {
		mc.windowStats.buckets[i] = WindowBucket{}
	}
	mc.windowStats.currentBucket = 0
	mc.windowStats.mu.Unlock()
}

// 全局指标收集器
var globalMetricsCollector = NewMetricsCollector(time.Minute, 60)

// GetGlobalMetricsCollector 获取全局指标收集器
func GetGlobalMetricsCollector() *MetricsCollector {
	return globalMetricsCollector
}
