package usage

import (
	"sync"
	"time"
)

// Stats represents usage statistics for the entire system
type Stats struct {
	mu sync.RWMutex

	// Global counters
	TotalRequests int64 `json:"total_requests"`
	SuccessCount  int64 `json:"success_count"`
	FailureCount  int64 `json:"failure_count"`
	TotalTokens   int64 `json:"total_tokens"`

	// Per-credential statistics
	Credentials map[string]*CredentialUsage `json:"credentials"`

	// Time-based statistics
	DailyStats  map[string]*DailyStats  `json:"daily_stats"`  // key: "2025-01-06"
	HourlyStats map[int]*HourlyStats    `json:"hourly_stats"` // key: 0-23

	// Per-API statistics (gemini/openai)
	APIs map[string]*APIStats `json:"apis"`
}

// CredentialUsage tracks usage for a single credential
type CredentialUsage struct {
	ID                 string                 `json:"id"`
	TotalCalls         int64                  `json:"total_calls"`
	SuccessCalls       int64                  `json:"success_calls"`
	FailureCalls       int64                  `json:"failure_calls"`
	Gemini25ProCalls   int64                  `json:"gemini_2_5_pro_calls"`
	TotalTokens        int64                  `json:"total_tokens"`
	InputTokens        int64                  `json:"input_tokens"`
	OutputTokens       int64                  `json:"output_tokens"`
	ReasoningTokens    int64                  `json:"reasoning_tokens"`
	CachedTokens       int64                  `json:"cached_tokens"`
	DailyLimit         int64                  `json:"daily_limit"`
	DailyUsage         int64                  `json:"daily_usage"`
	QuotaResetTime     time.Time              `json:"quota_reset_time"`
	LastUsed           time.Time              `json:"last_used"`
	ModelBreakdown     map[string]*ModelStats `json:"model_breakdown"`
	mu                 sync.RWMutex
}

// ModelStats tracks usage for a specific model
type ModelStats struct {
	ModelName       string    `json:"model_name"`
	Calls           int64     `json:"calls"`
	Tokens          int64     `json:"tokens"`
	InputTokens     int64     `json:"input_tokens"`
	OutputTokens    int64     `json:"output_tokens"`
	ReasoningTokens int64     `json:"reasoning_tokens"`
	CachedTokens    int64     `json:"cached_tokens"`
	LastUsed        time.Time `json:"last_used"`
}

// DailyStats tracks statistics for a specific day
type DailyStats struct {
	Date     string `json:"date"` // "2025-01-06"
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
	Success  int64  `json:"success"`
	Failure  int64  `json:"failure"`
}

// HourlyStats tracks statistics for a specific hour (0-23)
type HourlyStats struct {
	Hour     int   `json:"hour"` // 0-23
	Requests int64 `json:"requests"`
	Tokens   int64 `json:"tokens"`
	Success  int64 `json:"success"`
	Failure  int64 `json:"failure"`
}

// APIStats tracks statistics for a specific API (gemini/openai)
type APIStats struct {
	Name          string                 `json:"name"`
	TotalRequests int64                  `json:"total_requests"`
	TotalTokens   int64                  `json:"total_tokens"`
	Models        map[string]*ModelStats `json:"models"`
}

// TokenUsage represents token consumption for a single request
type TokenUsage struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
	CachedTokens    int64 `json:"cached_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

// RequestRecord represents a single request for statistics tracking
type RequestRecord struct {
	Timestamp    time.Time   `json:"timestamp"`
	CredentialID string      `json:"credential_id"`
	API          string      `json:"api"`           // "gemini" or "openai"
	Model        string      `json:"model"`         // e.g., "gemini-2.0-flash-exp"
	Success      bool        `json:"success"`
	StatusCode   int         `json:"status_code"`
	Tokens       *TokenUsage `json:"tokens,omitempty"`
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{
		Credentials: make(map[string]*CredentialUsage),
		DailyStats:  make(map[string]*DailyStats),
		HourlyStats: make(map[int]*HourlyStats),
		APIs:        make(map[string]*APIStats),
	}
}

// NewCredentialUsage creates a new CredentialUsage instance
func NewCredentialUsage(id string) *CredentialUsage {
	return &CredentialUsage{
		ID:             id,
		ModelBreakdown: make(map[string]*ModelStats),
		QuotaResetTime: getNextUTC7AM(),
	}
}

// NewModelStats creates a new ModelStats instance
func NewModelStats(modelName string) *ModelStats {
	return &ModelStats{
		ModelName: modelName,
	}
}

// NewAPIStats creates a new APIStats instance
func NewAPIStats(name string) *APIStats {
	return &APIStats{
		Name:   name,
		Models: make(map[string]*ModelStats),
	}
}

// getNextUTC7AM calculates the next UTC 07:00 time for quota reset
func getNextUTC7AM() time.Time {
	now := time.Now().UTC()
	today7AM := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, time.UTC)

	if now.Before(today7AM) {
		return today7AM
	}
	return today7AM.Add(24 * time.Hour)
}

// IsGemini25Pro checks if a model is gemini-2.5-pro variant
func IsGemini25Pro(model string) bool {
	// Remove common prefixes
	for _, prefix := range []string{"流式抗截断/", "假流式/"} {
		if len(model) > len(prefix) && model[:len(prefix)] == prefix {
			model = model[len(prefix):]
			break
		}
	}

	// Remove common suffixes
	for _, suffix := range []string{"-maxthinking", "-nothinking", "-search", "-thinking"} {
		if len(model) > len(suffix) && model[len(model)-len(suffix):] == suffix {
			model = model[:len(model)-len(suffix)]
			break
		}
	}

	return model == "gemini-2.5-pro"
}

// ShouldResetQuota checks if quota should be reset
func (cu *CredentialUsage) ShouldResetQuota() bool {
	cu.mu.RLock()
	defer cu.mu.RUnlock()
	return time.Now().UTC().After(cu.QuotaResetTime)
}

// ResetQuota resets the daily quota
func (cu *CredentialUsage) ResetQuota() {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	cu.DailyUsage = 0
	cu.QuotaResetTime = getNextUTC7AM()
}

// IsQuotaExceeded checks if daily quota is exceeded
func (cu *CredentialUsage) IsQuotaExceeded() bool {
	cu.mu.RLock()
	defer cu.mu.RUnlock()
	if cu.DailyLimit <= 0 {
		return false // unlimited
	}
	return cu.DailyUsage >= cu.DailyLimit
}

// IncrementUsage increments the usage counters
func (cu *CredentialUsage) IncrementUsage(model string, tokens *TokenUsage, success bool) {
	cu.mu.Lock()
	defer cu.mu.Unlock()

	cu.TotalCalls++
	if success {
		cu.SuccessCalls++
	} else {
		cu.FailureCalls++
	}

	if IsGemini25Pro(model) {
		cu.Gemini25ProCalls++
		cu.DailyUsage++
	}

	if tokens != nil {
		cu.TotalTokens += tokens.TotalTokens
		cu.InputTokens += tokens.InputTokens
		cu.OutputTokens += tokens.OutputTokens
		cu.ReasoningTokens += tokens.ReasoningTokens
		cu.CachedTokens += tokens.CachedTokens
	}

	cu.LastUsed = time.Now()

	// Update model breakdown
	if _, ok := cu.ModelBreakdown[model]; !ok {
		cu.ModelBreakdown[model] = NewModelStats(model)
	}
	ms := cu.ModelBreakdown[model]
	ms.Calls++
	if tokens != nil {
		ms.Tokens += tokens.TotalTokens
		ms.InputTokens += tokens.InputTokens
		ms.OutputTokens += tokens.OutputTokens
		ms.ReasoningTokens += tokens.ReasoningTokens
		ms.CachedTokens += tokens.CachedTokens
	}
	ms.LastUsed = time.Now()
}

// Snapshot returns a read-only copy of the credential usage
func (cu *CredentialUsage) Snapshot() *CredentialUsage {
	cu.mu.RLock()
	defer cu.mu.RUnlock()

	snapshot := &CredentialUsage{
		ID:               cu.ID,
		TotalCalls:       cu.TotalCalls,
		SuccessCalls:     cu.SuccessCalls,
		FailureCalls:     cu.FailureCalls,
		Gemini25ProCalls: cu.Gemini25ProCalls,
		TotalTokens:      cu.TotalTokens,
		InputTokens:      cu.InputTokens,
		OutputTokens:     cu.OutputTokens,
		ReasoningTokens:  cu.ReasoningTokens,
		CachedTokens:     cu.CachedTokens,
		DailyLimit:       cu.DailyLimit,
		DailyUsage:       cu.DailyUsage,
		QuotaResetTime:   cu.QuotaResetTime,
		LastUsed:         cu.LastUsed,
		ModelBreakdown:   make(map[string]*ModelStats),
	}

	for k, v := range cu.ModelBreakdown {
		snapshot.ModelBreakdown[k] = &ModelStats{
			ModelName:       v.ModelName,
			Calls:           v.Calls,
			Tokens:          v.Tokens,
			InputTokens:     v.InputTokens,
			OutputTokens:    v.OutputTokens,
			ReasoningTokens: v.ReasoningTokens,
			CachedTokens:    v.CachedTokens,
			LastUsed:        v.LastUsed,
		}
	}

	return snapshot
}

