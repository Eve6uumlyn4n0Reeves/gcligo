package monitoring

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// EnhancedMetrics provides detailed metrics tracking
type EnhancedMetrics struct {
	mu sync.RWMutex

	// Upstream request metrics by provider
	upstreamRequests    map[string]int64            // provider -> count
	upstreamDurations   map[string][]float64        // provider -> durations
	upstreamErrors      map[string]map[string]int64 // provider -> error_type -> count
	upstreamRetries     map[string]int64            // provider -> retry_count
	upstreamStatusCodes map[string]map[int]int64    // provider -> status_code -> count

	// Request metrics by endpoint
	endpointRequests  map[string]int64     // endpoint -> count
	endpointDurations map[string][]float64 // endpoint -> durations
	endpointErrors    map[string]int64     // endpoint -> error_count

	// Streaming metrics
	streamingRequests    int64
	streamingChunks      int64
	streamingDisconnects map[string]int64 // reason -> count

	// Credential metrics
	credentialRotations   int64
	credentialFailures    map[string]int64   // cred_id -> failure_count
	credentialHealthScore map[string]float64 // cred_id -> score

	// Cache metrics
	cacheHits   int64
	cacheMisses int64

	// Token usage
	totalTokens      int64
	promptTokens     int64
	completionTokens int64

	// Transaction metrics
	transactionAttempts map[string]int64 // backend -> attempts
	transactionSuccess  map[string]int64 // backend -> commits
	transactionFailures map[string]int64 // backend -> rollbacks/failures

	// Storage metrics
	storageOps       map[string]map[string]*storageOpAggregate // backend -> operation -> aggregate
	storageSlowOps   map[string]map[string]int64               // backend -> operation -> slow count
	storagePoolStats map[string]StoragePoolStats               // backend -> pool stats snapshot

	// Plan apply metrics
	planOps map[planOpKey]*PlanOpStats
}

type storageOpAggregate struct {
	Count     int64
	Errors    int64
	Durations []float64
}

// StoragePoolStats captures basic pool statistics for storage backends with pooling.
type StoragePoolStats struct {
	Active int64
	Idle   int64
	Hits   int64
	Misses int64
}

// StorageOpStats represents summarized metrics for a storage operation.
type StorageOpStats struct {
	Count     int64
	Errors    int64
	Durations []float64
}

type planOpKey struct {
	Backend string
	Stage   string
	Status  string
}

// PlanOpStats captures plan apply counters and duration aggregates.
type PlanOpStats struct {
	Count        int64
	DurationSumS float64
}

// NewEnhancedMetrics creates a new metrics tracker
func NewEnhancedMetrics() *EnhancedMetrics {
	return &EnhancedMetrics{
		upstreamRequests:      make(map[string]int64),
		upstreamDurations:     make(map[string][]float64),
		upstreamErrors:        make(map[string]map[string]int64),
		upstreamRetries:       make(map[string]int64),
		upstreamStatusCodes:   make(map[string]map[int]int64),
		endpointRequests:      make(map[string]int64),
		endpointDurations:     make(map[string][]float64),
		endpointErrors:        make(map[string]int64),
		streamingDisconnects:  make(map[string]int64),
		credentialFailures:    make(map[string]int64),
		credentialHealthScore: make(map[string]float64),
		transactionAttempts:   make(map[string]int64),
		transactionSuccess:    make(map[string]int64),
		transactionFailures:   make(map[string]int64),
		storageOps:            make(map[string]map[string]*storageOpAggregate),
		storageSlowOps:        make(map[string]map[string]int64),
		storagePoolStats:      make(map[string]StoragePoolStats),
		planOps:               make(map[planOpKey]*PlanOpStats),
	}
}

// RecordUpstreamRequest records an upstream request
func (m *EnhancedMetrics) RecordUpstreamRequest(provider string, duration time.Duration, statusCode int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.upstreamRequests[provider]++

	durationSec := duration.Seconds()
	m.upstreamDurations[provider] = append(m.upstreamDurations[provider], durationSec)

	// Limit duration slice size
	if len(m.upstreamDurations[provider]) > 1000 {
		m.upstreamDurations[provider] = m.upstreamDurations[provider][500:]
	}

	// Record status code
	if m.upstreamStatusCodes[provider] == nil {
		m.upstreamStatusCodes[provider] = make(map[int]int64)
	}
	m.upstreamStatusCodes[provider][statusCode]++

	// Record errors
	if err != nil {
		if m.upstreamErrors[provider] == nil {
			m.upstreamErrors[provider] = make(map[string]int64)
		}
		errorType := classifyError(err)
		m.upstreamErrors[provider][errorType]++
	}
}

// RecordUpstreamRetry records a retry attempt
func (m *EnhancedMetrics) RecordUpstreamRetry(provider string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.upstreamRetries[provider]++
}

// RecordEndpointRequest records an endpoint request
func (m *EnhancedMetrics) RecordEndpointRequest(endpoint string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.endpointRequests[endpoint]++

	durationSec := duration.Seconds()
	m.endpointDurations[endpoint] = append(m.endpointDurations[endpoint], durationSec)

	// Limit duration slice size
	if len(m.endpointDurations[endpoint]) > 1000 {
		m.endpointDurations[endpoint] = m.endpointDurations[endpoint][500:]
	}

	if err != nil {
		m.endpointErrors[endpoint]++
	}
}

// RecordStreamingChunk records a streaming chunk
func (m *EnhancedMetrics) RecordStreamingChunk() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.streamingChunks++
}

// RecordStreamingDisconnect records a streaming disconnection
func (m *EnhancedMetrics) RecordStreamingDisconnect(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.streamingDisconnects[reason]++
}

// RecordCredentialRotation records a credential rotation
func (m *EnhancedMetrics) RecordCredentialRotation() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.credentialRotations++
}

// RecordCredentialFailure records a credential failure
func (m *EnhancedMetrics) RecordCredentialFailure(credID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.credentialFailures[credID]++
}

// UpdateCredentialHealth updates credential health score
func (m *EnhancedMetrics) UpdateCredentialHealth(credID string, score float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.credentialHealthScore[credID] = score
}

// RecordCacheHit records a cache hit
func (m *EnhancedMetrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cacheHits++
}

// RecordCacheMiss records a cache miss
func (m *EnhancedMetrics) RecordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cacheMisses++
}

// RecordTokenUsage records token usage
func (m *EnhancedMetrics) RecordTokenUsage(promptTokens, completionTokens int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.promptTokens += promptTokens
	m.completionTokens += completionTokens
	m.totalTokens += promptTokens + completionTokens
}

func (m *EnhancedMetrics) RecordTransactionAttempt(backend string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := normalizeBackendLabel(backend)
	m.transactionAttempts[key]++
}

func (m *EnhancedMetrics) RecordTransactionCommit(backend string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := normalizeBackendLabel(backend)
	m.transactionSuccess[key]++
}

func (m *EnhancedMetrics) RecordTransactionFailure(backend string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := normalizeBackendLabel(backend)
	m.transactionFailures[key]++
}

// RecordStorageOperation tracks a storage backend operation.
func (m *EnhancedMetrics) RecordStorageOperation(backend, operation string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := normalizeBackendLabel(backend)
	if m.storageOps[key] == nil {
		m.storageOps[key] = make(map[string]*storageOpAggregate)
	}
	agg := m.storageOps[key][operation]
	if agg == nil {
		agg = &storageOpAggregate{}
		m.storageOps[key][operation] = agg
	}
	agg.Count++
	if err != nil {
		agg.Errors++
	}
	agg.Durations = append(agg.Durations, duration.Seconds())
	if len(agg.Durations) > 1000 {
		agg.Durations = agg.Durations[len(agg.Durations)/2:]
	}

	if duration >= 250*time.Millisecond {
		if m.storageSlowOps[key] == nil {
			m.storageSlowOps[key] = make(map[string]int64)
		}
		m.storageSlowOps[key][operation]++
	}
}

// UpdateStoragePoolStats captures pool metrics for a backend.
func (m *EnhancedMetrics) UpdateStoragePoolStats(backend string, stats StoragePoolStats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storagePoolStats[normalizeBackendLabel(backend)] = stats
}

// StorageMetrics returns copies of storage operation metrics and pool statistics.
func (m *EnhancedMetrics) StorageMetrics() (map[string]map[string]StorageOpStats, map[string]map[string]int64, map[string]StoragePoolStats) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ops := make(map[string]map[string]StorageOpStats, len(m.storageOps))
	for backend, opMap := range m.storageOps {
		backendMap := make(map[string]StorageOpStats, len(opMap))
		for operation, agg := range opMap {
			durations := append([]float64(nil), agg.Durations...)
			backendMap[operation] = StorageOpStats{
				Count:     agg.Count,
				Errors:    agg.Errors,
				Durations: durations,
			}
		}
		ops[backend] = backendMap
	}

	slow := make(map[string]map[string]int64, len(m.storageSlowOps))
	for backend, opMap := range m.storageSlowOps {
		backendMap := make(map[string]int64, len(opMap))
		for operation, count := range opMap {
			backendMap[operation] = count
		}
		slow[backend] = backendMap
	}

	pools := make(map[string]StoragePoolStats, len(m.storagePoolStats))
	for backend, stats := range m.storagePoolStats {
		pools[backend] = stats
	}

	return ops, slow, pools
}

// RecordPlanApply captures plan apply attempts across backends/stages.
func (m *EnhancedMetrics) RecordPlanApply(backend, stage, status string, duration time.Duration) {
	if backend == "" {
		backend = "unknown"
	}
	if stage == "" {
		stage = "apply"
	}
	if status == "" {
		status = "success"
	}

	key := planOpKey{Backend: backend, Stage: stage, Status: status}

	m.mu.Lock()
	defer m.mu.Unlock()
	stats := m.planOps[key]
	if stats == nil {
		stats = &PlanOpStats{}
		m.planOps[key] = stats
	}
	stats.Count++
	stats.DurationSumS += duration.Seconds()
}

// PlanMetrics returns copies of the recorded plan apply metrics.
func (m *EnhancedMetrics) PlanMetrics() map[string]map[string]map[string]PlanOpStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[string]map[string]map[string]PlanOpStats, len(m.planOps))
	for key, stats := range m.planOps {
		stageMap, ok := out[key.Backend]
		if !ok {
			stageMap = make(map[string]map[string]PlanOpStats)
			out[key.Backend] = stageMap
		}
		statusMap, ok := stageMap[key.Stage]
		if !ok {
			statusMap = make(map[string]PlanOpStats)
			stageMap[key.Stage] = statusMap
		}
		statusMap[key.Status] = PlanOpStats{
			Count:        stats.Count,
			DurationSumS: stats.DurationSumS,
		}
	}
	return out
}

// GetSnapshot returns a snapshot of current metrics
func (m *EnhancedMetrics) GetSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := make(map[string]interface{})

	// Upstream metrics
	upstream := make(map[string]interface{})
	for provider, count := range m.upstreamRequests {
		upstream[provider] = map[string]interface{}{
			"requests":     count,
			"avg_duration": calculateAverage(m.upstreamDurations[provider]),
			"p50_duration": calculatePercentile(m.upstreamDurations[provider], 0.5),
			"p95_duration": calculatePercentile(m.upstreamDurations[provider], 0.95),
			"p99_duration": calculatePercentile(m.upstreamDurations[provider], 0.99),
			"retries":      m.upstreamRetries[provider],
			"errors":       m.upstreamErrors[provider],
			"status_codes": m.upstreamStatusCodes[provider],
		}
	}
	snapshot["upstream"] = upstream

	// Endpoint metrics
	endpoints := make(map[string]interface{})
	for endpoint, count := range m.endpointRequests {
		endpoints[endpoint] = map[string]interface{}{
			"requests":     count,
			"avg_duration": calculateAverage(m.endpointDurations[endpoint]),
			"errors":       m.endpointErrors[endpoint],
		}
	}
	snapshot["endpoints"] = endpoints

	// Transaction metrics
	txAttempts := make(map[string]int64, len(m.transactionAttempts))
	for k, v := range m.transactionAttempts {
		txAttempts[k] = v
	}
	txSuccess := make(map[string]int64, len(m.transactionSuccess))
	for k, v := range m.transactionSuccess {
		txSuccess[k] = v
	}
	txFailures := make(map[string]int64, len(m.transactionFailures))
	for k, v := range m.transactionFailures {
		txFailures[k] = v
	}
	snapshot["transactions"] = map[string]interface{}{
		"attempts": txAttempts,
		"commits":  txSuccess,
		"failures": txFailures,
	}

	// Streaming metrics
	snapshot["streaming"] = map[string]interface{}{
		"requests":    m.streamingRequests,
		"chunks":      m.streamingChunks,
		"disconnects": m.streamingDisconnects,
	}

	// Credential metrics
	snapshot["credentials"] = map[string]interface{}{
		"rotations":     m.credentialRotations,
		"failures":      m.credentialFailures,
		"health_scores": m.credentialHealthScore,
	}

	// Cache metrics
	snapshot["cache"] = map[string]interface{}{
		"hits":     m.cacheHits,
		"misses":   m.cacheMisses,
		"hit_rate": calculateCacheHitRate(m.cacheHits, m.cacheMisses),
	}

	// Token usage
	snapshot["tokens"] = map[string]interface{}{
		"total":      m.totalTokens,
		"prompt":     m.promptTokens,
		"completion": m.completionTokens,
	}

	storageOps := make(map[string]map[string]interface{})
	for backend, opMap := range m.storageOps {
		backendMap := make(map[string]interface{}, len(opMap))
		for operation, agg := range opMap {
			backendMap[operation] = map[string]interface{}{
				"count":        agg.Count,
				"errors":       agg.Errors,
				"avg_duration": calculateAverage(agg.Durations),
			}
		}
		storageOps[backend] = backendMap
	}
	slowOps := make(map[string]map[string]int64, len(m.storageSlowOps))
	for backend, opMap := range m.storageSlowOps {
		backendMap := make(map[string]int64, len(opMap))
		for operation, count := range opMap {
			backendMap[operation] = count
		}
		slowOps[backend] = backendMap
	}
	poolStats := make(map[string]StoragePoolStats, len(m.storagePoolStats))
	for backend, stats := range m.storagePoolStats {
		poolStats[backend] = stats
	}
	snapshot["storage"] = map[string]interface{}{
		"operations": storageOps,
		"slow":       slowOps,
		"pool":       poolStats,
	}

	return snapshot
}

// Helper functions
func classifyError(err error) string {
	if err == nil {
		return "none"
	}

	errStr := err.Error()
	switch {
	case contains(errStr, "timeout"):
		return "timeout"
	case contains(errStr, "connection"):
		return "connection"
	case contains(errStr, "429"):
		return "rate_limit"
	case contains(errStr, "500"), contains(errStr, "502"), contains(errStr, "503"):
		return "server_error"
	case contains(errStr, "401"), contains(errStr, "403"):
		return "auth_error"
	default:
		return "other"
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculatePercentile computes the nearest-rank percentile on a sorted copy
// of the input slice to avoid mutating the original order.
// percentile is expressed in [0,1].
func calculatePercentile(values []float64, percentile float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	// Clamp percentile to [0,1]
	if percentile < 0 {
		percentile = 0
	}
	if percentile > 1 {
		percentile = 1
	}
	// Make a copy and sort ascending
	cp := make([]float64, n)
	copy(cp, values)
	sort.Float64s(cp)

	if percentile == 0 {
		return cp[0]
	}
	// Nearest-rank method (1-based rank)
	rank := int(math.Ceil(percentile * float64(n)))
	if rank < 1 {
		rank = 1
	} else if rank > n {
		rank = n
	}
	return cp[rank-1]
}

func calculateCacheHitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

func normalizeBackendLabel(label string) string {
	label = strings.TrimSpace(strings.ToLower(label))
	if label == "" {
		return "unknown"
	}
	return label
}
