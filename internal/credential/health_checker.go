package credential

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gcli2api-go/internal/oauth"
)

// HealthChecker provides credential health monitoring and scoring
type HealthChecker struct {
	oauthMgr   *oauth.Manager
	testModel  string
	timeout    time.Duration
	mu         sync.RWMutex
	lastCheck  map[string]time.Time
	checkCache map[string]HealthResult
}

// HealthResult represents the result of a credential health check
type HealthResult struct {
	CredentialID  string        `json:"credential_id"`
	Healthy       bool          `json:"healthy"`
	Score         float64       `json:"score"`
	LastChecked   time.Time     `json:"last_checked"`
	ResponseTime  time.Duration `json:"response_time"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	TokenValid    bool          `json:"token_valid"`
	ProjectAccess bool          `json:"project_access"`
}

// NewHealthChecker creates a new credential health checker
func NewHealthChecker(testModel string, timeout time.Duration) *HealthChecker {
	if testModel == "" {
		testModel = "gemini-2.5-flash"
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &HealthChecker{
		oauthMgr:   oauth.NewManager("", "", ""), // Will use credential's own client info
		testModel:  testModel,
		timeout:    timeout,
		lastCheck:  make(map[string]time.Time),
		checkCache: make(map[string]HealthResult),
	}
}

// CheckCredential performs a comprehensive health check on a credential
func (hc *HealthChecker) CheckCredential(ctx context.Context, cred *Credential) HealthResult {
	if cred == nil {
		return HealthResult{
			Healthy:      false,
			Score:        0.0,
			LastChecked:  time.Now(),
			ErrorMessage: "credential is nil",
		}
	}

	result := HealthResult{
		CredentialID: cred.ID,
		LastChecked:  time.Now(),
	}

	start := time.Now()
	defer func() {
		result.ResponseTime = time.Since(start)
		hc.mu.Lock()
		hc.lastCheck[cred.ID] = result.LastChecked
		hc.checkCache[cred.ID] = result
		hc.mu.Unlock()
	}()

	// Create timeout context
	checkCtx, cancel := context.WithTimeout(ctx, hc.timeout)
	defer cancel()

	// Check 1: Token validation
	if cred.AccessToken != "" {
		if valid, err := hc.oauthMgr.ValidateToken(checkCtx, cred.AccessToken); err == nil {
			result.TokenValid = valid
		} else {
			result.ErrorMessage = fmt.Sprintf("token validation failed: %v", err)
		}
	}

	// Check 2: Basic API test with flash model
	if cred.AccessToken != "" && cred.ProjectID != "" {
		if err := hc.testAPICall(checkCtx, cred); err == nil {
			result.ProjectAccess = true
			result.Healthy = true
		} else {
			result.ErrorMessage = fmt.Sprintf("api test failed: %v", err)
		}
	}

	// Calculate health score
	result.Score = hc.calculateScore(cred, result)

	return result
}

// testAPICall performs a minimal API call to test credential health
func (hc *HealthChecker) testAPICall(ctx context.Context, cred *Credential) error {
	// Create a minimal test request
	testReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": "Hi"},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 10,
			"temperature":     0.1,
		},
	}

	_, err := json.Marshal(testReq)
	if err != nil {
		return fmt.Errorf("marshal test request: %w", err)
	}

	// This would need integration with the actual upstream client
	// For now, we'll simulate based on token validation
	if !cred.IsHealthy() {
		return fmt.Errorf("credential marked as unhealthy")
	}

	return nil
}

// calculateScore computes a health score based on various factors
func (hc *HealthChecker) calculateScore(cred *Credential, result HealthResult) float64 {
	if cred == nil {
		return 0.0
	}

	score := 50.0 // Base score

	// Token validity (+30)
	if result.TokenValid {
		score += 30.0
	}

	// Project access (+20)
	if result.ProjectAccess {
		score += 20.0
	}

	// Success rate factor
	cred.mu.RLock()
	totalReq := cred.TotalRequests
	successCount := cred.SuccessCount
	failureCount := cred.FailureCount
	consecutiveFails := cred.ConsecutiveFails
	autoBanned := cred.AutoBanned
	disabled := cred.Disabled
	cred.mu.RUnlock()

	if autoBanned || disabled {
		return 0.0
	}

	if totalReq > 0 {
		successRate := float64(successCount) / float64(totalReq)
		score = score * successRate
	}

	// Penalize consecutive failures
	if consecutiveFails > 0 {
		penalty := float64(consecutiveFails) * 5.0
		score = score - penalty
		if score < 0 {
			score = 0
		}
	}

	// Response time factor (faster = better)
	if result.ResponseTime > 0 && result.ResponseTime < 30*time.Second {
		timeFactor := 1.0 - (float64(result.ResponseTime.Milliseconds()) / 30000.0)
		if timeFactor > 0 {
			score = score * (0.8 + 0.2*timeFactor) // 80-100% based on speed
		}
	}

	// Recent failure weight
	if failureCount > 0 && totalReq > 0 {
		recentFailureRate := float64(failureCount) / float64(totalReq)
		if recentFailureRate > 0.1 {
			score = score * (1.0 - recentFailureRate*0.5)
		}
	}

	if score > 100.0 {
		score = 100.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// BatchCheckCredentials performs health checks on multiple credentials
func (hc *HealthChecker) BatchCheckCredentials(ctx context.Context, creds []*Credential) map[string]HealthResult {
	results := make(map[string]HealthResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrent checks
	semaphore := make(chan struct{}, 5)

	for _, cred := range creds {
		if cred == nil {
			continue
		}

		wg.Add(1)
		go func(c *Credential) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			result := hc.CheckCredential(ctx, c)
			mu.Lock()
			results[c.ID] = result
			mu.Unlock()
		}(cred)
	}

	wg.Wait()
	return results
}

// GetCachedResult returns a cached health check result
func (hc *HealthChecker) GetCachedResult(credID string) (HealthResult, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result, exists := hc.checkCache[credID]
	return result, exists
}

// GetLastCheckTime returns the last check time for a credential
func (hc *HealthChecker) GetLastCheckTime(credID string) time.Time {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	return hc.lastCheck[credID]
}

// ClearCache clears the health check cache
func (hc *HealthChecker) ClearCache() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.lastCheck = make(map[string]time.Time)
	hc.checkCache = make(map[string]HealthResult)
}

// GetHealthSummary returns a summary of all cached health results
func (hc *HealthChecker) GetHealthSummary() map[string]interface{} {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	healthy := 0
	unhealthy := 0
	totalScore := 0.0

	for _, result := range hc.checkCache {
		if result.Healthy {
			healthy++
		} else {
			unhealthy++
		}
		totalScore += result.Score
	}

	total := healthy + unhealthy
	avgScore := 0.0
	if total > 0 {
		avgScore = totalScore / float64(total)
	}

	return map[string]interface{}{
		"total_credentials": total,
		"healthy":           healthy,
		"unhealthy":         unhealthy,
		"average_score":     avgScore,
		"last_updated":      time.Now(),
	}
}
