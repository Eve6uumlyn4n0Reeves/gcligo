package usage

import (
	"testing"
	"time"
)

func TestNewStats(t *testing.T) {
	stats := NewStats()
	if stats == nil {
		t.Fatal("NewStats returned nil")
	}
	if stats.Credentials == nil {
		t.Error("Credentials map not initialized")
	}
	if stats.DailyStats == nil {
		t.Error("DailyStats map not initialized")
	}
	if stats.HourlyStats == nil {
		t.Error("HourlyStats map not initialized")
	}
	if stats.APIs == nil {
		t.Error("APIs map not initialized")
	}
}

func TestNewCredentialUsage(t *testing.T) {
	id := "test-cred-123"
	cu := NewCredentialUsage(id)
	if cu == nil {
		t.Fatal("NewCredentialUsage returned nil")
	}
	if cu.ID != id {
		t.Errorf("Expected ID %s, got %s", id, cu.ID)
	}
	if cu.ModelBreakdown == nil {
		t.Error("ModelBreakdown map not initialized")
	}
	if cu.QuotaResetTime.IsZero() {
		t.Error("QuotaResetTime not initialized")
	}
}

func TestIsGemini25Pro(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"gemini-2.5-pro", true},
		{"gemini-2.5-pro-maxthinking", true},
		{"gemini-2.5-pro-nothinking", true},
		{"gemini-2.5-pro-search", true},
		{"流式抗截断/gemini-2.5-pro", true},
		{"假流式/gemini-2.5-pro-thinking", true},
		{"gemini-2.0-flash-exp", false},
		{"gemini-1.5-pro", false},
		{"gpt-4", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := IsGemini25Pro(tt.model)
			if result != tt.expected {
				t.Errorf("IsGemini25Pro(%q) = %v, want %v", tt.model, result, tt.expected)
			}
		})
	}
}

func TestCredentialUsage_IncrementUsage(t *testing.T) {
	cu := NewCredentialUsage("test-cred")
	cu.DailyLimit = 100

	tokens := &TokenUsage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}

	// Test successful call
	cu.IncrementUsage("gemini-2.0-flash-exp", tokens, true)
	if cu.TotalCalls != 1 {
		t.Errorf("Expected TotalCalls=1, got %d", cu.TotalCalls)
	}
	if cu.SuccessCalls != 1 {
		t.Errorf("Expected SuccessCalls=1, got %d", cu.SuccessCalls)
	}
	if cu.TotalTokens != 150 {
		t.Errorf("Expected TotalTokens=150, got %d", cu.TotalTokens)
	}
	if cu.InputTokens != 100 {
		t.Errorf("Expected InputTokens=100, got %d", cu.InputTokens)
	}
	if cu.OutputTokens != 50 {
		t.Errorf("Expected OutputTokens=50, got %d", cu.OutputTokens)
	}

	// Test gemini-2.5-pro call
	cu.IncrementUsage("gemini-2.5-pro", tokens, true)
	if cu.Gemini25ProCalls != 1 {
		t.Errorf("Expected Gemini25ProCalls=1, got %d", cu.Gemini25ProCalls)
	}
	if cu.DailyUsage != 1 {
		t.Errorf("Expected DailyUsage=1, got %d", cu.DailyUsage)
	}

	// Test failed call
	cu.IncrementUsage("gemini-2.0-flash-exp", nil, false)
	if cu.FailureCalls != 1 {
		t.Errorf("Expected FailureCalls=1, got %d", cu.FailureCalls)
	}

	// Check model breakdown
	if len(cu.ModelBreakdown) != 2 {
		t.Errorf("Expected 2 models in breakdown, got %d", len(cu.ModelBreakdown))
	}
}

func TestCredentialUsage_QuotaManagement(t *testing.T) {
	cu := NewCredentialUsage("test-cred")
	cu.DailyLimit = 10
	cu.DailyUsage = 0

	// Not exceeded initially
	if cu.IsQuotaExceeded() {
		t.Error("Quota should not be exceeded initially")
	}

	// Add usage
	for i := 0; i < 10; i++ {
		cu.IncrementUsage("gemini-2.5-pro", nil, true)
	}

	// Should be at limit
	if !cu.IsQuotaExceeded() {
		t.Error("Quota should be exceeded after 10 calls")
	}

	// Test reset
	cu.ResetQuota()
	if cu.DailyUsage != 0 {
		t.Errorf("Expected DailyUsage=0 after reset, got %d", cu.DailyUsage)
	}
	if cu.IsQuotaExceeded() {
		t.Error("Quota should not be exceeded after reset")
	}
}

func TestCredentialUsage_Snapshot(t *testing.T) {
	cu := NewCredentialUsage("test-cred")
	cu.TotalCalls = 100
	cu.TotalTokens = 5000
	cu.ModelBreakdown["model1"] = &ModelStats{
		ModelName: "model1",
		Calls:     50,
		Tokens:    2500,
	}

	snapshot := cu.Snapshot()
	if snapshot == nil {
		t.Fatal("Snapshot returned nil")
	}
	if snapshot.ID != cu.ID {
		t.Error("Snapshot ID mismatch")
	}
	if snapshot.TotalCalls != cu.TotalCalls {
		t.Error("Snapshot TotalCalls mismatch")
	}
	if snapshot.TotalTokens != cu.TotalTokens {
		t.Error("Snapshot TotalTokens mismatch")
	}
	if len(snapshot.ModelBreakdown) != len(cu.ModelBreakdown) {
		t.Error("Snapshot ModelBreakdown length mismatch")
	}

	// Modify original should not affect snapshot
	cu.TotalCalls = 200
	if snapshot.TotalCalls == 200 {
		t.Error("Snapshot should be independent of original")
	}
}

func TestGetNextUTC7AM(t *testing.T) {
	next := getNextUTC7AM()
	if next.IsZero() {
		t.Error("getNextUTC7AM returned zero time")
	}
	if next.Hour() != 7 {
		t.Errorf("Expected hour=7, got %d", next.Hour())
	}
	if next.Minute() != 0 || next.Second() != 0 {
		t.Error("Expected minute and second to be 0")
	}
	if next.Before(time.Now().UTC()) {
		t.Error("Next reset time should be in the future")
	}
}

func TestCredentialUsage_ShouldResetQuota(t *testing.T) {
	cu := NewCredentialUsage("test-cred")

	// Set quota reset time to past
	cu.QuotaResetTime = time.Now().UTC().Add(-1 * time.Hour)
	if !cu.ShouldResetQuota() {
		t.Error("Should reset quota when reset time is in the past")
	}

	// Set quota reset time to future
	cu.QuotaResetTime = time.Now().UTC().Add(1 * time.Hour)
	if cu.ShouldResetQuota() {
		t.Error("Should not reset quota when reset time is in the future")
	}
}

