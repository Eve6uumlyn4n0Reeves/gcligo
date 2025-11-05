package credential

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCredentialHealthScore(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(*Credential)
		expectedScore float64
		minScore      float64
		maxScore      float64
	}{
		{
			name: "perfect health",
			setupFunc: func(c *Credential) {
				c.TotalRequests = 100
				c.SuccessCount = 100
				c.LastSuccess = time.Now()
			},
			minScore: 0.9,
			maxScore: 1.2,
		},
		{
			name: "with failures",
			setupFunc: func(c *Credential) {
				c.TotalRequests = 100
				c.SuccessCount = 80
				c.FailureCount = 20
				c.ConsecutiveFails = 2
			},
			minScore: 0.5,
			maxScore: 0.9,
		},
		{
			name: "rate limited",
			setupFunc: func(c *Credential) {
				c.TotalRequests = 100
				c.SuccessCount = 70
				c.ErrorCodeCounts = map[int]int{429: 5}
			},
			minScore: 0.0,
			maxScore: 0.5,
		},
		{
			name: "auto-banned",
			setupFunc: func(c *Credential) {
				c.AutoBanned = true
			},
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &Credential{
				ErrorCodeCounts: make(map[int]int),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(cred)
			}

			score := cred.GetScore()

			if tt.expectedScore > 0 {
				assert.Equal(t, tt.expectedScore, score)
			} else {
				assert.GreaterOrEqual(t, score, tt.minScore)
				assert.LessOrEqual(t, score, tt.maxScore)
			}
		})
	}
}

func TestAutobanLogic(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		errorCount   int
		expectBanned bool
	}{
		{"rate limit threshold", 429, 3, true},
		{"forbidden threshold", 403, 5, true},
		{"unauthorized threshold", 401, 3, true},
		{"server error threshold", 500, 10, true},
		{"below threshold", 429, 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &Credential{
				ErrorCodeCounts: make(map[int]int),
			}

			// Simulate errors
			for i := 0; i < tt.errorCount; i++ {
				cred.MarkFailure("test error", tt.statusCode)
			}

			assert.Equal(t, tt.expectBanned, cred.AutoBanned)
		})
	}
}

func TestCredentialRecovery(t *testing.T) {
	cred := &Credential{
		AutoBanned:       true,
		BannedAt:         time.Now().Add(-3 * time.Hour),
		BannedReason:     "Rate limit",
		ConsecutiveFails: 5,
		ErrorCodeCounts:  map[int]int{429: 3},
	}

	assert.True(t, cred.CanRecover(), "Should be recoverable after 3 hours")

	cred.Recover()

	assert.False(t, cred.AutoBanned, "Should not be banned after recovery")
	assert.Equal(t, 0, cred.ConsecutiveFails, "Should reset consecutive fails")
	assert.Equal(t, 0, len(cred.ErrorCodes), "Should clear error codes")
}

func TestQuotaManagement(t *testing.T) {
	cred := &Credential{
		DailyLimit:      100,
		DailyUsage:      0,
		QuotaResetTime:  time.Now().Add(24 * time.Hour),
		ErrorCodeCounts: make(map[int]int),
	}

	// Simulate requests
	for i := 0; i < 95; i++ {
		cred.MarkSuccess()
	}

	assert.Equal(t, int64(95), cred.DailyUsage)
	assert.True(t, cred.IsHealthy(), "Should still be healthy at 95/100")

	// Push to limit
	for i := 0; i < 10; i++ {
		cred.MarkSuccess()
	}

	assert.False(t, cred.IsHealthy(), "Should be unhealthy when quota exceeded")
}

func TestFailureWeightAccumulationAndCap(t *testing.T) {
	cred := &Credential{ErrorCodeCounts: make(map[int]int)}

	for i := 0; i < 5; i++ {
		cred.MarkFailure("rate limit", 429)
	}

	assert.Greater(t, cred.FailureWeight, 0.0)
	assert.LessOrEqual(t, cred.FailureWeight, 10.0, "Failure weight should cap at 10")
	assert.Positive(t, cred.FailureCount)
	assert.Positive(t, cred.ConsecutiveFails)
}

func TestFailureWeightDecay(t *testing.T) {
	cred := &Credential{}
	cred.FailureWeight = 5.0

	now := time.Now()
	cred.LastFailureWeightDecay = now.Add(-30 * time.Minute)
	cred.decayFailureWeightUnsafe(now, false)

	expected := 5.0 * math.Pow(0.5, 3) // three half-life intervals (30 / 10)
	assert.InDelta(t, expected, cred.FailureWeight, 0.01)

	cred.LastFailureWeightDecay = now.Add(-10 * time.Minute)
	cred.decayFailureWeightUnsafe(now, true)
	expectedAggressive := expected * math.Pow(0.5, 2) // aggressive half-life of 5 minutes
	assert.InDelta(t, expectedAggressive, cred.FailureWeight, 0.01)
}

func BenchmarkGetScore(b *testing.B) {
	cred := &Credential{
		TotalRequests:   1000,
		SuccessCount:    900,
		LastSuccess:     time.Now(),
		ErrorCodeCounts: make(map[int]int),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cred.GetScore()
	}
}
