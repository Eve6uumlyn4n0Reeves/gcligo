package usage

import (
	"context"
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	storage := &NoOpStorage{}
	tracker := NewTracker(storage)
	if tracker == nil {
		t.Fatal("NewTracker returned nil")
	}
	if tracker.stats == nil {
		t.Error("Tracker stats not initialized")
	}
	if tracker.storage == nil {
		t.Error("Tracker storage not set")
	}
}

func TestTracker_Record(t *testing.T) {
	tracker := NewTracker(&NoOpStorage{})

	record := &RequestRecord{
		Timestamp:    time.Now(),
		CredentialID: "cred-123",
		API:          "gemini",
		Model:        "gemini-2.0-flash-exp",
		Success:      true,
		StatusCode:   200,
		Tokens: &TokenUsage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	}

	tracker.Record(record)

	stats := tracker.GetStats()
	if stats.TotalRequests != 1 {
		t.Errorf("Expected TotalRequests=1, got %d", stats.TotalRequests)
	}
	if stats.SuccessCount != 1 {
		t.Errorf("Expected SuccessCount=1, got %d", stats.SuccessCount)
	}
	if stats.TotalTokens != 150 {
		t.Errorf("Expected TotalTokens=150, got %d", stats.TotalTokens)
	}

	// Check credential stats
	if len(stats.Credentials) != 1 {
		t.Errorf("Expected 1 credential, got %d", len(stats.Credentials))
	}
	cred, ok := stats.Credentials["cred-123"]
	if !ok {
		t.Fatal("Credential not found")
	}
	if cred.TotalCalls != 1 {
		t.Errorf("Expected credential TotalCalls=1, got %d", cred.TotalCalls)
	}

	// Check API stats
	if len(stats.APIs) != 1 {
		t.Errorf("Expected 1 API, got %d", len(stats.APIs))
	}
	api, ok := stats.APIs["gemini"]
	if !ok {
		t.Fatal("API not found")
	}
	if api.TotalRequests != 1 {
		t.Errorf("Expected API TotalRequests=1, got %d", api.TotalRequests)
	}
}

func TestTracker_RecordMultiple(t *testing.T) {
	tracker := NewTracker(&NoOpStorage{})

	// Record multiple requests
	for i := 0; i < 10; i++ {
		record := &RequestRecord{
			Timestamp:    time.Now(),
			CredentialID: "cred-123",
			API:          "gemini",
			Model:        "gemini-2.0-flash-exp",
			Success:      i%2 == 0, // Alternate success/failure
			StatusCode:   200,
			Tokens: &TokenUsage{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  150,
			},
		}
		tracker.Record(record)
	}

	stats := tracker.GetStats()
	if stats.TotalRequests != 10 {
		t.Errorf("Expected TotalRequests=10, got %d", stats.TotalRequests)
	}
	if stats.SuccessCount != 5 {
		t.Errorf("Expected SuccessCount=5, got %d", stats.SuccessCount)
	}
	if stats.FailureCount != 5 {
		t.Errorf("Expected FailureCount=5, got %d", stats.FailureCount)
	}
	if stats.TotalTokens != 1500 {
		t.Errorf("Expected TotalTokens=1500, got %d", stats.TotalTokens)
	}
}

func TestTracker_TimeStats(t *testing.T) {
	tracker := NewTracker(&NoOpStorage{})

	now := time.Now()
	record := &RequestRecord{
		Timestamp:    now,
		CredentialID: "cred-123",
		API:          "gemini",
		Model:        "gemini-2.0-flash-exp",
		Success:      true,
		StatusCode:   200,
		Tokens: &TokenUsage{
			TotalTokens: 150,
		},
	}

	tracker.Record(record)

	stats := tracker.GetStats()

	// Check daily stats
	dateKey := now.Format("2006-01-02")
	daily, ok := stats.DailyStats[dateKey]
	if !ok {
		t.Fatal("Daily stats not found")
	}
	if daily.Requests != 1 {
		t.Errorf("Expected daily requests=1, got %d", daily.Requests)
	}
	if daily.Tokens != 150 {
		t.Errorf("Expected daily tokens=150, got %d", daily.Tokens)
	}

	// Check hourly stats
	hour := now.Hour()
	hourly, ok := stats.HourlyStats[hour]
	if !ok {
		t.Fatal("Hourly stats not found")
	}
	if hourly.Requests != 1 {
		t.Errorf("Expected hourly requests=1, got %d", hourly.Requests)
	}
	if hourly.Tokens != 150 {
		t.Errorf("Expected hourly tokens=150, got %d", hourly.Tokens)
	}
}

func TestTracker_GetCredentialStats(t *testing.T) {
	tracker := NewTracker(&NoOpStorage{})

	record := &RequestRecord{
		Timestamp:    time.Now(),
		CredentialID: "cred-123",
		API:          "gemini",
		Model:        "gemini-2.0-flash-exp",
		Success:      true,
		StatusCode:   200,
		Tokens: &TokenUsage{
			TotalTokens: 150,
		},
	}

	tracker.Record(record)

	// Get existing credential
	cred := tracker.GetCredentialStats("cred-123")
	if cred == nil {
		t.Fatal("Credential stats not found")
	}
	if cred.TotalCalls != 1 {
		t.Errorf("Expected TotalCalls=1, got %d", cred.TotalCalls)
	}

	// Get non-existing credential
	cred2 := tracker.GetCredentialStats("non-existing")
	if cred2 != nil {
		t.Error("Expected nil for non-existing credential")
	}
}

func TestTracker_IsQuotaExceeded(t *testing.T) {
	tracker := NewTracker(&NoOpStorage{})

	// Create credential with quota
	tracker.stats.Credentials["cred-123"] = NewCredentialUsage("cred-123")
	tracker.stats.Credentials["cred-123"].DailyLimit = 5
	tracker.stats.Credentials["cred-123"].DailyUsage = 0

	// Not exceeded initially
	if tracker.IsQuotaExceeded("cred-123") {
		t.Error("Quota should not be exceeded initially")
	}

	// Add usage
	for i := 0; i < 5; i++ {
		record := &RequestRecord{
			Timestamp:    time.Now(),
			CredentialID: "cred-123",
			API:          "gemini",
			Model:        "gemini-2.5-pro",
			Success:      true,
			StatusCode:   200,
		}
		tracker.Record(record)
	}

	// Should be exceeded
	if !tracker.IsQuotaExceeded("cred-123") {
		t.Error("Quota should be exceeded after 5 gemini-2.5-pro calls")
	}

	// Non-existing credential
	if tracker.IsQuotaExceeded("non-existing") {
		t.Error("Non-existing credential should not be quota exceeded")
	}
}

func TestTracker_QuotaReset(t *testing.T) {
	tracker := NewTracker(&NoOpStorage{})

	// Create credential with past quota reset time
	cred := NewCredentialUsage("cred-123")
	cred.DailyLimit = 10
	cred.DailyUsage = 5
	cred.QuotaResetTime = time.Now().UTC().Add(-1 * time.Hour)
	tracker.stats.Credentials["cred-123"] = cred

	// Record a request - should trigger quota reset
	record := &RequestRecord{
		Timestamp:    time.Now(),
		CredentialID: "cred-123",
		API:          "gemini",
		Model:        "gemini-2.5-pro",
		Success:      true,
		StatusCode:   200,
	}
	tracker.Record(record)

	// Check that quota was reset
	updatedCred := tracker.GetCredentialStats("cred-123")
	if updatedCred.DailyUsage != 1 {
		t.Errorf("Expected DailyUsage=1 after reset, got %d", updatedCred.DailyUsage)
	}
}

func TestTracker_StartStop(t *testing.T) {
	tracker := NewTracker(&NoOpStorage{})
	ctx := context.Background()

	// Start tracker
	if err := tracker.Start(ctx); err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}

	// Record some data
	record := &RequestRecord{
		Timestamp:    time.Now(),
		CredentialID: "cred-123",
		API:          "gemini",
		Model:        "gemini-2.0-flash-exp",
		Success:      true,
		StatusCode:   200,
		Tokens: &TokenUsage{
			TotalTokens: 150,
		},
	}
	tracker.Record(record)

	// Stop tracker
	if err := tracker.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop tracker: %v", err)
	}

	// Verify data is still accessible
	stats := tracker.GetStats()
	if stats.TotalRequests != 1 {
		t.Errorf("Expected TotalRequests=1 after stop, got %d", stats.TotalRequests)
	}
}

