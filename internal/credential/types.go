package credential

import (
	"math"
	"sync"
	"time"
)

// Credential represents a single credential (OAuth or API key)
type Credential struct {
	ID           string
	Type         string // "oauth" or "api_key"
	Source       string `json:"-"` // 来源标识，便于反向写回
	Email        string
	ProjectID    string
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	TokenURI     string `json:"token_uri,omitempty"`
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	APIKey       string // For API key type

	// ✅ Enhanced state tracking
	Disabled      bool
	FailureCount  int
	LastFailure   time.Time
	LastSuccess   time.Time
	TotalRequests int64
	SuccessCount  int64
	FailureReason string

	// ✅ Error code tracking for auto-ban
	ErrorCodes      []int       // Recent error codes encountered
	ErrorCodeCounts map[int]int // Count of each error code
	LastErrorCode   int         // Most recent error code

	// ✅ Auto-ban system
	AutoBanned       bool      // Whether credential was automatically banned
	BannedAt         time.Time // When the credential was banned
	BannedReason     string    // Reason for ban (e.g., "429 rate limit", "403 forbidden")
	BanUntil         time.Time // Temporary ban expiration time
	ConsecutiveFails int       // Consecutive failures without success

	// ✅ Health scoring
	HealthScore            float64   // Current health score (0.0 to 1.0)
	LastScoreCalc          time.Time // When health score was last calculated
	FailureWeight          float64   // Weighted penalty accumulated from failures
	LastFailureWeightDecay time.Time // Timestamp for last decay application

	// ✅ Quota management
	DailyLimit     int64     // Daily request limit (0 = unlimited)
	DailyUsage     int64     // Current daily usage
	QuotaResetTime time.Time // When quota resets (UTC)

	// Call count for rotation
	CallsSinceRotation int32

	mu sync.RWMutex
}

// CredentialState captures mutable runtime fields we want to persist across restarts.
type CredentialState struct {
	Disabled           bool        `json:"disabled"`
	AutoBanned         bool        `json:"auto_banned"`
	BannedReason       string      `json:"banned_reason,omitempty"`
	BannedAt           time.Time   `json:"banned_at,omitempty"`
	BanUntil           time.Time   `json:"ban_until,omitempty"`
	FailureCount       int         `json:"failure_count"`
	ConsecutiveFails   int         `json:"consecutive_fails"`
	LastFailure        time.Time   `json:"last_failure,omitempty"`
	LastSuccess        time.Time   `json:"last_success,omitempty"`
	LastErrorCode      int         `json:"last_error_code"`
	ErrorCodeCounts    map[int]int `json:"error_code_counts,omitempty"`
	FailureReason      string      `json:"failure_reason,omitempty"`
	TotalRequests      int64       `json:"total_requests"`
	SuccessCount       int64       `json:"success_count"`
	DailyLimit         int64       `json:"daily_limit"`
	DailyUsage         int64       `json:"daily_usage"`
	QuotaResetTime     time.Time   `json:"quota_reset_time,omitempty"`
	CallsSinceRotation int32       `json:"calls_since_rotation"`
	HealthScore        float64     `json:"health_score"`
	LastScoreCalc      time.Time   `json:"last_score_calc,omitempty"`
	FailureWeight      float64     `json:"failure_weight,omitempty"`
	LastFailureWeight  time.Time   `json:"last_failure_weight,omitempty"`
}

var failureSeverityWeights = map[int]float64{
	429: 2.5,
	403: 1.8,
	401: 2.2,
	500: 1.2,
	502: 1.2,
	503: 1.2,
}

func severityForStatus(code int) float64 {
	if weight, ok := failureSeverityWeights[code]; ok {
		return weight
	}
	if code >= 500 && code < 600 {
		return 1.0
	}
	if code >= 400 && code < 500 {
		return 0.8
	}
	return 0.5
}

func (c *Credential) addFailureWeightUnsafe(code int) {
	weight := severityForStatus(code)
	now := time.Now()
	c.decayFailureWeightUnsafe(now, false)
	c.FailureWeight += weight
	if c.FailureWeight > 10 {
		c.FailureWeight = 10
	}
	c.LastFailureWeightDecay = now
}

func (c *Credential) decayFailureWeightUnsafe(now time.Time, aggressive bool) {
	if c.FailureWeight <= 0 {
		c.LastFailureWeightDecay = now
		return
	}
	if c.LastFailureWeightDecay.IsZero() {
		c.LastFailureWeightDecay = now
		return
	}
	elapsed := now.Sub(c.LastFailureWeightDecay)
	if elapsed <= 0 {
		return
	}
	halfLife := 10 * time.Minute
	if aggressive {
		halfLife = 5 * time.Minute
	}
	decay := math.Pow(0.5, float64(elapsed)/float64(halfLife))
	c.FailureWeight *= decay
	if c.FailureWeight < 0.05 {
		c.FailureWeight = 0
	}
	c.LastFailureWeightDecay = now
}

// IsExpired checks if the OAuth token is expired
func (c *Credential) IsExpired() bool {
	if c.Type != "oauth" {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

// ✅ IsHealthy checks if the credential is in good health (enhanced)
func (c *Credential) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if disabled or auto-banned
	if c.Disabled || c.AutoBanned {
		// Check if temporary ban has expired
		if c.AutoBanned && !c.BanUntil.IsZero() && time.Now().After(c.BanUntil) {
			// Ban expired, but still return false until recovery process re-enables it
			return false
		}
		return false
	}

	// Consider unhealthy if too many consecutive failures
	if c.ConsecutiveFails > 5 {
		return false
	}

	if c.FailureWeight > 4 {
		return false
	}

	// Check daily quota
	if c.DailyLimit > 0 && c.DailyUsage >= c.DailyLimit {
		return false
	}

	// Consider unhealthy if last failure was recent and no success since
	if !c.LastFailure.IsZero() && c.LastSuccess.Before(c.LastFailure) {
		if time.Since(c.LastFailure) < 5*time.Minute {
			return false
		}
	}

	// Check if too many rate limit errors
	if count, ok := c.ErrorCodeCounts[429]; ok && count > 3 {
		return false
	}

	return true
}

// ✅ GetScore calculates a health score for credential selection (enhanced)
func (c *Credential) GetScore() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if !c.LastScoreCalc.IsZero() && now.Sub(c.LastScoreCalc) < time.Minute {
		c.decayFailureWeightUnsafe(now, false)
		return c.HealthScore
	}

	score := c.calculateScoreUnsafe()
	c.HealthScore = score
	c.LastScoreCalc = now
	return score
}

// ✅ MarkSuccess records a successful request (enhanced)
func (c *Credential) MarkSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LastSuccess = time.Now()
	c.SuccessCount++
	c.TotalRequests++
	c.DailyUsage++
	c.FailureCount = 0     // Reset consecutive failures
	c.ConsecutiveFails = 0 // Reset consecutive fails
	c.CallsSinceRotation++

	// Clear error codes on success (decay mechanism)
	if len(c.ErrorCodes) > 0 {
		c.ErrorCodes = c.ErrorCodes[:0]
	}
	// Reduce error code counts
	for code := range c.ErrorCodeCounts {
		if c.ErrorCodeCounts[code] > 0 {
			c.ErrorCodeCounts[code]--
		}
	}

	c.decayFailureWeightUnsafe(time.Now(), true)

	// Update health score
	c.HealthScore = c.calculateScoreUnsafe()
	c.LastScoreCalc = time.Now()

	// Check if quota should be reset (daily reset)
	if time.Now().After(c.QuotaResetTime) {
		c.DailyUsage = 1 // Reset to 1 (current request)
		c.QuotaResetTime = time.Now().Add(24 * time.Hour)
	}
}

// ✅ MarkFailure records a failed request (enhanced with error code tracking)
func (c *Credential) MarkFailure(reason string, statusCode int) {
	c.MarkFailureWithConfig(reason, statusCode, DefaultAutoBanConfig)
}

// MarkFailureWithConfig allows custom auto-ban thresholds.
func (c *Credential) MarkFailureWithConfig(reason string, statusCode int, cfg AutoBanConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LastFailure = time.Now()
	c.FailureCount++
	c.ConsecutiveFails++
	c.TotalRequests++
	c.FailureReason = reason
	c.CallsSinceRotation++

	c.LastErrorCode = statusCode
	if statusCode > 0 {
		c.ErrorCodes = append(c.ErrorCodes, statusCode)
		if len(c.ErrorCodes) > 20 {
			c.ErrorCodes = c.ErrorCodes[len(c.ErrorCodes)-20:]
		}
		if c.ErrorCodeCounts == nil {
			c.ErrorCodeCounts = make(map[int]int)
		}
		c.ErrorCodeCounts[statusCode]++
	}

	c.addFailureWeightUnsafe(statusCode)

	threshold429 := cfg.Threshold429
	if threshold429 <= 0 {
		threshold429 = DefaultAutoBanConfig.Threshold429
	}
	threshold403 := cfg.Threshold403
	if threshold403 <= 0 {
		threshold403 = DefaultAutoBanConfig.Threshold403
	}
	threshold401 := cfg.Threshold401
	if threshold401 <= 0 {
		threshold401 = DefaultAutoBanConfig.Threshold401
	}
	threshold5xx := cfg.Threshold5xx
	if threshold5xx <= 0 {
		threshold5xx = DefaultAutoBanConfig.Threshold5xx
	}
	consecutiveLimit := cfg.ConsecutiveFailLimit
	if consecutiveLimit <= 0 {
		consecutiveLimit = DefaultAutoBanConfig.ConsecutiveFailLimit
	}

	shouldBan := false
	banReason := ""
	banDuration := time.Duration(0)

	if cfg.Enabled {
		switch statusCode {
		case 429:
			if c.ErrorCodeCounts[429] >= threshold429 {
				shouldBan = true
				banReason = "Rate limit exceeded (429)"
				banDuration = 30 * time.Minute
			}
		case 403:
			if c.ErrorCodeCounts[403] >= threshold403 {
				shouldBan = true
				banReason = "Forbidden access (403)"
				banDuration = time.Hour
			}
		case 401:
			if c.ErrorCodeCounts[401] >= threshold401 {
				shouldBan = true
				banReason = "Unauthorized (401)"
				banDuration = 2 * time.Hour
			}
		case 500, 502, 503:
			if c.ErrorCodeCounts[500]+c.ErrorCodeCounts[502]+c.ErrorCodeCounts[503] >= threshold5xx {
				shouldBan = true
				banReason = "Server errors (5xx)"
				banDuration = 15 * time.Minute
			}
		}

		if c.ConsecutiveFails >= consecutiveLimit {
			shouldBan = true
			banReason = "Too many consecutive failures"
			banDuration = time.Hour
		}
	}

	if shouldBan {
		c.AutoBanned = true
		c.BannedAt = time.Now()
		c.BannedReason = banReason
		if banDuration > 0 {
			c.BanUntil = time.Now().Add(banDuration)
		}
	}

	c.HealthScore = c.calculateScoreUnsafe()
	c.LastScoreCalc = time.Now()
}

// calculateScoreUnsafe calculates health score without locking (internal use)
func (c *Credential) calculateScoreUnsafe() float64 {
	now := time.Now()
	c.decayFailureWeightUnsafe(now, false)

	if c.Disabled || c.AutoBanned || c.TotalRequests == 0 {
		return 0
	}

	successRate := float64(c.SuccessCount) / float64(c.TotalRequests)

	recencyPenalty := 1.0
	if !c.LastFailure.IsZero() {
		minutesSinceFailure := time.Since(c.LastFailure).Minutes()
		if minutesSinceFailure < 10 {
			recencyPenalty = minutesSinceFailure / 10.0
		}
	}

	recencyBonus := 1.0
	if !c.LastSuccess.IsZero() {
		minutesSinceSuccess := time.Since(c.LastSuccess).Minutes()
		if minutesSinceSuccess < 5 {
			recencyBonus = 1.2
		}
	}

	consecutivePenalty := 1.0
	if c.ConsecutiveFails > 0 {
		consecutivePenalty = 1.0 / (1.0 + float64(c.ConsecutiveFails)*0.2)
	}

	errorPenalty := 1.0
	if count429, ok := c.ErrorCodeCounts[429]; ok && count429 > 0 {
		errorPenalty *= 0.5
	}
	if count403, ok := c.ErrorCodeCounts[403]; ok && count403 > 0 {
		errorPenalty *= 0.7
	}
	if count500, ok := c.ErrorCodeCounts[500]; ok && count500 > 2 {
		errorPenalty *= 0.8
	}

	quotaPenalty := 1.0
	if c.DailyLimit > 0 {
		usageRatio := float64(c.DailyUsage) / float64(c.DailyLimit)
		if usageRatio > 0.9 {
			quotaPenalty = 0.1
		} else if usageRatio > 0.75 {
			quotaPenalty = 0.5
		} else if usageRatio > 0.5 {
			quotaPenalty = 0.8
		}
	}

	failurePenalty := 1.0
	if c.FailureWeight > 0 {
		failurePenalty = 1.0 / (1.0 + c.FailureWeight)
	}

	score := successRate * recencyPenalty * recencyBonus * consecutivePenalty * errorPenalty * quotaPenalty * failurePenalty
	if score > 1.0 {
		score = 1.0
	} else if score < 0 {
		score = 0
	}

	return score
}

// ResetStats clears runtime statistics and ban state
func (c *Credential) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.TotalRequests = 0
	c.SuccessCount = 0
	c.FailureCount = 0
	c.FailureReason = ""
	c.LastFailure = time.Time{}
	c.LastSuccess = time.Time{}
	c.LastErrorCode = 0
	c.CallsSinceRotation = 0
	c.ConsecutiveFails = 0
	c.HealthScore = 0
	c.LastScoreCalc = time.Time{}
	c.FailureWeight = 0
	c.LastFailureWeightDecay = time.Time{}
	c.AutoBanned = false
	c.BannedAt = time.Time{}
	c.BannedReason = ""
	c.BanUntil = time.Time{}
	c.DailyUsage = 0
	if len(c.ErrorCodes) > 0 {
		c.ErrorCodes = c.ErrorCodes[:0]
	}
	c.ErrorCodeCounts = make(map[int]int)
}

// ShouldRotate checks if credential should be rotated based on call count
func (c *Credential) ShouldRotate(threshold int32) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CallsSinceRotation >= threshold
}

// ResetCallCount resets the rotation counter
func (c *Credential) ResetCallCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CallsSinceRotation = 0
}

// ✅ Clone creates a deep copy of the credential (enhanced)
func (c *Credential) Clone() *Credential {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Deep copy error codes
	errorCodes := make([]int, len(c.ErrorCodes))
	copy(errorCodes, c.ErrorCodes)

	// Deep copy error code counts
	errorCodeCounts := make(map[int]int)
	for k, v := range c.ErrorCodeCounts {
		errorCodeCounts[k] = v
	}

	return &Credential{
		ID:                     c.ID,
		Type:                   c.Type,
		Source:                 c.Source,
		Email:                  c.Email,
		ProjectID:              c.ProjectID,
		AccessToken:            c.AccessToken,
		RefreshToken:           c.RefreshToken,
		ExpiresAt:              c.ExpiresAt,
		APIKey:                 c.APIKey,
		Disabled:               c.Disabled,
		FailureCount:           c.FailureCount,
		LastFailure:            c.LastFailure,
		LastSuccess:            c.LastSuccess,
		TotalRequests:          c.TotalRequests,
		SuccessCount:           c.SuccessCount,
		FailureReason:          c.FailureReason,
		ErrorCodes:             errorCodes,
		ErrorCodeCounts:        errorCodeCounts,
		LastErrorCode:          c.LastErrorCode,
		AutoBanned:             c.AutoBanned,
		BannedAt:               c.BannedAt,
		BannedReason:           c.BannedReason,
		BanUntil:               c.BanUntil,
		ConsecutiveFails:       c.ConsecutiveFails,
		HealthScore:            c.HealthScore,
		LastScoreCalc:          c.LastScoreCalc,
		FailureWeight:          c.FailureWeight,
		LastFailureWeightDecay: c.LastFailureWeightDecay,
		DailyLimit:             c.DailyLimit,
		DailyUsage:             c.DailyUsage,
		QuotaResetTime:         c.QuotaResetTime,
		CallsSinceRotation:     c.CallsSinceRotation,
	}
}

// SnapshotState captures mutable runtime data for persistence.
func (c *Credential) SnapshotState() *CredentialState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	state := &CredentialState{
		Disabled:           c.Disabled,
		AutoBanned:         c.AutoBanned,
		BannedReason:       c.BannedReason,
		BannedAt:           c.BannedAt,
		BanUntil:           c.BanUntil,
		FailureCount:       c.FailureCount,
		ConsecutiveFails:   c.ConsecutiveFails,
		LastFailure:        c.LastFailure,
		LastSuccess:        c.LastSuccess,
		LastErrorCode:      c.LastErrorCode,
		FailureReason:      c.FailureReason,
		TotalRequests:      c.TotalRequests,
		SuccessCount:       c.SuccessCount,
		DailyLimit:         c.DailyLimit,
		DailyUsage:         c.DailyUsage,
		QuotaResetTime:     c.QuotaResetTime,
		CallsSinceRotation: c.CallsSinceRotation,
		HealthScore:        c.HealthScore,
		LastScoreCalc:      c.LastScoreCalc,
		FailureWeight:      c.FailureWeight,
		LastFailureWeight:  c.LastFailureWeightDecay,
	}
	if len(c.ErrorCodeCounts) > 0 {
		state.ErrorCodeCounts = make(map[int]int, len(c.ErrorCodeCounts))
		for k, v := range c.ErrorCodeCounts {
			state.ErrorCodeCounts[k] = v
		}
	}
	return state
}

// RestoreState applies persisted runtime data onto the credential.
func (c *Credential) RestoreState(state *CredentialState) {
	if state == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Disabled = state.Disabled
	c.AutoBanned = state.AutoBanned
	c.BannedReason = state.BannedReason
	c.BannedAt = state.BannedAt
	c.BanUntil = state.BanUntil
	c.FailureCount = state.FailureCount
	c.ConsecutiveFails = state.ConsecutiveFails
	c.LastFailure = state.LastFailure
	c.LastSuccess = state.LastSuccess
	c.LastErrorCode = state.LastErrorCode
	c.FailureReason = state.FailureReason
	c.TotalRequests = state.TotalRequests
	c.SuccessCount = state.SuccessCount
	c.DailyLimit = state.DailyLimit
	c.DailyUsage = state.DailyUsage
	c.QuotaResetTime = state.QuotaResetTime
	c.CallsSinceRotation = state.CallsSinceRotation
	c.HealthScore = state.HealthScore
	c.LastScoreCalc = state.LastScoreCalc
	c.FailureWeight = state.FailureWeight
	c.LastFailureWeightDecay = state.LastFailureWeight
	if len(state.ErrorCodeCounts) > 0 {
		c.ErrorCodeCounts = make(map[int]int, len(state.ErrorCodeCounts))
		for k, v := range state.ErrorCodeCounts {
			c.ErrorCodeCounts[k] = v
		}
	} else {
		c.ErrorCodeCounts = make(map[int]int)
	}
}

// ✅ CanRecover checks if a banned credential can be recovered
func (c *Credential) CanRecover() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.AutoBanned {
		return false
	}

	// Check if temporary ban has expired
	if !c.BanUntil.IsZero() && time.Now().After(c.BanUntil) {
		return true
	}

	// Check if enough time has passed since ban
	if time.Since(c.BannedAt) > 2*time.Hour {
		return true
	}

	return false
}

// ✅ Recover attempts to recover a banned credential
func (c *Credential) Recover() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.AutoBanned = false
	c.BannedAt = time.Time{}
	c.BannedReason = ""
	c.BanUntil = time.Time{}
	c.ConsecutiveFails = 0
	c.FailureCount = 0
	c.FailureWeight = 0
	c.LastFailureWeightDecay = time.Now()

	// Clear error codes
	c.ErrorCodes = c.ErrorCodes[:0]
	c.ErrorCodeCounts = make(map[int]int)

	// Recalculate health score
	c.HealthScore = c.calculateScoreUnsafe()
	c.LastScoreCalc = time.Now()
}
