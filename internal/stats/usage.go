package stats

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gcli2api-go/internal/models"
	"gcli2api-go/internal/storage"
	"gcli2api-go/internal/utils"
	log "github.com/sirupsen/logrus"
)

// UsageStats tracks API usage statistics
type UsageStats struct {
	backend        storage.Backend
	mu             sync.RWMutex
	resetSchedule  time.Time
	resetInterval  time.Duration
	resetLocation  *time.Location
	resetHourLocal int
}

const (
	aggregateTotalKey    = "__system__/total"
	aggregateModelPrefix = "__system__/model/"
)

const (
	// AggregateKindTotal indicates the aggregate bucket for all requests.
	AggregateKindTotal = "total"
	// AggregateKindModel indicates the aggregate bucket for a specific model.
	AggregateKindModel = "model"
)

// ClassifyAggregateKey reports whether a usage key is an aggregate bucket and returns its kind/value.
func ClassifyAggregateKey(key string) (kind, value string, ok bool) {
	if key == aggregateTotalKey {
		return AggregateKindTotal, "", true
	}
	if strings.HasPrefix(key, aggregateModelPrefix) {
		return AggregateKindModel, strings.TrimPrefix(key, aggregateModelPrefix), true
	}
	return "", "", false
}

// UsageRecord represents usage for a specific API key
type UsageRecord struct {
	APIKey           string
	TotalRequests    int64
	SuccessRequests  int64
	FailedRequests   int64
	TotalTokens      int64
	PromptTokens     int64
	CompletionTokens int64
	LastUsed         time.Time
	CreatedAt        time.Time
}

// NewUsageStats creates a new usage stats tracker
func NewUsageStats(backend storage.Backend, resetInterval time.Duration, tz string, hour int) *UsageStats {
	if resetInterval == 0 {
		resetInterval = 24 * time.Hour // Default: daily reset
	}

	loc, err := utils.ParseLocation(tz)
	if err != nil {
		log.WithError(err).Warnf("invalid usage reset timezone %q, falling back to UTC+7", tz)
		loc, _ = utils.ParseLocation("UTC+7")
	}
	if hour < 0 || hour > 23 {
		log.Warnf("usage reset hour %d out of range, defaulting to 0", hour)
		hour = 0
	}

	us := &UsageStats{
		backend:        backend,
		resetInterval:  resetInterval,
		resetLocation:  loc,
		resetHourLocal: hour,
	}
	us.resetSchedule = calculateNextReset(us.resetInterval, us.resetLocation, us.resetHourLocal)
	return us
}

// RecordRequest records an API request
func (u *UsageStats) RecordRequest(ctx context.Context, apiKey, model string, success bool, promptTokens, completionTokens int64) error {
	// No-op when backend unavailable
	if u == nil || u.backend == nil {
		return &storage.ErrNotSupported{Operation: "UsageStats.RecordRequest"}
	}
	// Check if reset is needed
	u.checkAndReset(ctx)

	u.mu.Lock()
	defer u.mu.Unlock()

	record := func(key string) error {
		if key == "" {
			return nil
		}
		if err := u.backend.IncrementUsage(ctx, key, "total_requests", 1); err != nil {
			return err
		}
		if success {
			if err := u.backend.IncrementUsage(ctx, key, "success_requests", 1); err != nil {
				return err
			}
		} else {
			if err := u.backend.IncrementUsage(ctx, key, "failed_requests", 1); err != nil {
				return err
			}
		}
		if promptTokens > 0 {
			if err := u.backend.IncrementUsage(ctx, key, "prompt_tokens", promptTokens); err != nil {
				return err
			}
		}
		if completionTokens > 0 {
			if err := u.backend.IncrementUsage(ctx, key, "completion_tokens", completionTokens); err != nil {
				return err
			}
		}
		totalTokens := promptTokens + completionTokens
		if totalTokens > 0 {
			if err := u.backend.IncrementUsage(ctx, key, "total_tokens", totalTokens); err != nil {
				return err
			}
		}
		return nil
	}

	if err := record(apiKey); err != nil {
		return err
	}
	_ = record(aggregateTotalKey)
	if m := strings.TrimSpace(model); m != "" {
		base := models.ParseModelName(m).BaseName
		_ = record(aggregateModelPrefix + base)
	}
	return nil
}

// GetUsage retrieves usage statistics for an API key
func (u *UsageStats) GetUsage(ctx context.Context, apiKey string) (*UsageRecord, error) {
	if u == nil || u.backend == nil {
		return nil, &storage.ErrNotSupported{Operation: "UsageStats.GetUsage"}
	}
	data, err := u.backend.GetUsage(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	record := &UsageRecord{
		APIKey: apiKey,
	}

	if v, ok := data["total_requests"].(int64); ok {
		record.TotalRequests = v
	} else if v, ok := data["total_requests"].(string); ok {
		fmt.Sscanf(v, "%d", &record.TotalRequests)
	} else if v, ok := data["total_requests"].(float64); ok {
		record.TotalRequests = int64(v)
	}

	if v, ok := data["success_requests"].(int64); ok {
		record.SuccessRequests = v
	} else if v, ok := data["success_requests"].(string); ok {
		fmt.Sscanf(v, "%d", &record.SuccessRequests)
	} else if v, ok := data["success_requests"].(float64); ok {
		record.SuccessRequests = int64(v)
	}

	if v, ok := data["failed_requests"].(int64); ok {
		record.FailedRequests = v
	} else if v, ok := data["failed_requests"].(string); ok {
		fmt.Sscanf(v, "%d", &record.FailedRequests)
	} else if v, ok := data["failed_requests"].(float64); ok {
		record.FailedRequests = int64(v)
	}

	if v, ok := data["total_tokens"].(int64); ok {
		record.TotalTokens = v
	} else if v, ok := data["total_tokens"].(string); ok {
		fmt.Sscanf(v, "%d", &record.TotalTokens)
	} else if v, ok := data["total_tokens"].(float64); ok {
		record.TotalTokens = int64(v)
	}

	if v, ok := data["prompt_tokens"].(int64); ok {
		record.PromptTokens = v
	} else if v, ok := data["prompt_tokens"].(string); ok {
		fmt.Sscanf(v, "%d", &record.PromptTokens)
	} else if v, ok := data["prompt_tokens"].(float64); ok {
		record.PromptTokens = int64(v)
	}

	if v, ok := data["completion_tokens"].(int64); ok {
		record.CompletionTokens = v
	} else if v, ok := data["completion_tokens"].(string); ok {
		fmt.Sscanf(v, "%d", &record.CompletionTokens)
	} else if v, ok := data["completion_tokens"].(float64); ok {
		record.CompletionTokens = int64(v)
	}

	return record, nil
}

// GetAllUsage retrieves all usage statistics
func (u *UsageStats) GetAllUsage(ctx context.Context) (map[string]*UsageRecord, error) {
	if u == nil || u.backend == nil {
		return nil, &storage.ErrNotSupported{Operation: "UsageStats.GetAllUsage"}
	}
	allData, err := u.backend.ListUsage(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*UsageRecord)
	for apiKey := range allData {
		record, err := u.GetUsage(ctx, apiKey)
		if err == nil {
			result[apiKey] = record
		}
	}

	return result, nil
}

// ResetUsage resets usage for a specific API key
func (u *UsageStats) ResetUsage(ctx context.Context, apiKey string) error {
	if u == nil || u.backend == nil {
		return &storage.ErrNotSupported{Operation: "UsageStats.ResetUsage"}
	}
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.backend.ResetUsage(ctx, apiKey)
}

// ResetAll resets all usage statistics
func (u *UsageStats) ResetAll(ctx context.Context) error {
	if u == nil || u.backend == nil {
		return &storage.ErrNotSupported{Operation: "UsageStats.ResetAll"}
	}
	u.mu.Lock()
	defer u.mu.Unlock()

	allData, err := u.backend.ListUsage(ctx)
	if err != nil {
		return err
	}

	for apiKey := range allData {
		if err := u.backend.ResetUsage(ctx, apiKey); err != nil {
			log.Errorf("Failed to reset usage for %s: %v", apiKey, err)
		}
	}

	u.resetSchedule = calculateNextReset(u.resetInterval, u.resetLocation, u.resetHourLocal)
	nextLog := u.resetSchedule
	if u.resetLocation != nil {
		nextLog = nextLog.In(u.resetLocation)
	}
	log.Infof("All usage statistics reset. Next reset at %v", nextLog)

	return nil
}

// checkAndReset checks if it's time to reset and performs reset if needed
func (u *UsageStats) checkAndReset(ctx context.Context) {
	if time.Now().UTC().After(u.resetSchedule) {
		log.Info("Scheduled usage reset triggered")
		if err := u.ResetAll(ctx); err != nil {
			log.Errorf("Failed to reset usage: %v", err)
		}
	}
}

// StartPeriodicReset starts a goroutine that periodically resets usage
func (u *UsageStats) StartPeriodicReset(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			u.checkAndReset(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// calculateNextReset calculates the next reset time in UTC.
func calculateNextReset(interval time.Duration, loc *time.Location, hour int) time.Time {
	nowUTC := time.Now().UTC()

	// If interval is 24 hours, reset at the specified local hour
	if interval == 24*time.Hour && loc != nil {
		if hour < 0 || hour > 23 {
			hour = 0
		}
		nowLocal := nowUTC.In(loc)
		nextLocal := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), hour, 0, 0, 0, loc)
		if !nextLocal.After(nowLocal) {
			nextLocal = nextLocal.Add(24 * time.Hour)
		}
		return nextLocal.UTC()
	}

	// Otherwise, just add interval to current time
	return nowUTC.Add(interval)
}

// GetSuccessRate calculates the success rate for an API key
func (r *UsageRecord) GetSuccessRate() float64 {
	if r.TotalRequests == 0 {
		return 0
	}
	return float64(r.SuccessRequests) / float64(r.TotalRequests) * 100
}

// GetFailureRate calculates the failure rate for an API key
func (r *UsageRecord) GetFailureRate() float64 {
	if r.TotalRequests == 0 {
		return 0
	}
	return float64(r.FailedRequests) / float64(r.TotalRequests) * 100
}
