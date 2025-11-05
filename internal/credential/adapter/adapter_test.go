package adapter

import (
	"testing"
	"time"
)

func TestApplyFilter(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	credentials := []*Credential{
		{
			ID:   "cred1",
			Type: "oauth",
			State: &CredentialState{
				Disabled:    false,
				HealthScore: 0.9,
				ErrorRate:   0.1,
				LastUsed:    &now,
			},
		},
		{
			ID:   "cred2",
			Type: "api_key",
			State: &CredentialState{
				Disabled:    true,
				HealthScore: 0.5,
				ErrorRate:   0.5,
				LastUsed:    &yesterday,
			},
		},
		{
			ID:   "cred3",
			Type: "oauth",
			State: &CredentialState{
				Disabled:    false,
				HealthScore: 0.3,
				ErrorRate:   0.7,
				LastUsed:    &now,
			},
		},
	}

	t.Run("No filter returns all", func(t *testing.T) {
		result := ApplyFilter(credentials, nil)
		if len(result) != 3 {
			t.Errorf("Expected 3 credentials, got %d", len(result))
		}
	})

	t.Run("Filter by disabled", func(t *testing.T) {
		disabled := false
		filter := &CredentialFilter{Disabled: &disabled}
		result := ApplyFilter(credentials, filter)

		if len(result) != 2 {
			t.Errorf("Expected 2 active credentials, got %d", len(result))
		}

		for _, cred := range result {
			if cred.State.Disabled {
				t.Error("Found disabled credential in filtered results")
			}
		}
	})

	t.Run("Filter by type", func(t *testing.T) {
		filter := &CredentialFilter{Type: "oauth"}
		result := ApplyFilter(credentials, filter)

		if len(result) != 2 {
			t.Errorf("Expected 2 oauth credentials, got %d", len(result))
		}

		for _, cred := range result {
			if cred.Type != "oauth" {
				t.Errorf("Expected oauth type, got %s", cred.Type)
			}
		}
	})

	t.Run("Filter by min health", func(t *testing.T) {
		minHealth := 0.6
		filter := &CredentialFilter{MinHealth: &minHealth}
		result := ApplyFilter(credentials, filter)

		if len(result) != 1 {
			t.Errorf("Expected 1 credential with health >= 0.6, got %d", len(result))
		}

		if result[0].ID != "cred1" {
			t.Errorf("Expected cred1, got %s", result[0].ID)
		}
	})

	t.Run("Filter by max health", func(t *testing.T) {
		maxHealth := 0.6
		filter := &CredentialFilter{MaxHealth: &maxHealth}
		result := ApplyFilter(credentials, filter)

		if len(result) != 2 {
			t.Errorf("Expected 2 credentials with health <= 0.6, got %d", len(result))
		}
	})

	t.Run("Filter by max error rate", func(t *testing.T) {
		maxError := 0.2
		filter := &CredentialFilter{MaxError: &maxError}
		result := ApplyFilter(credentials, filter)

		if len(result) != 1 {
			t.Errorf("Expected 1 credential with error <= 0.2, got %d", len(result))
		}
	})

	t.Run("Filter by last used", func(t *testing.T) {
		hourAgo := now.Add(-1 * time.Hour)
		filter := &CredentialFilter{LastUsed: &hourAgo}
		result := ApplyFilter(credentials, filter)

		// Should return credentials used after hourAgo (cred1 and cred3 used now)
		if len(result) != 2 {
			t.Errorf("Expected 2 credentials used after an hour ago, got %d", len(result))
		}
	})

	t.Run("Combined filters", func(t *testing.T) {
		disabled := false
		minHealth := 0.8
		filter := &CredentialFilter{
			Disabled:  &disabled,
			Type:      "oauth",
			MinHealth: &minHealth,
		}
		result := ApplyFilter(credentials, filter)

		if len(result) != 1 {
			t.Errorf("Expected 1 credential matching all filters, got %d", len(result))
		}

		if result[0].ID != "cred1" {
			t.Errorf("Expected cred1, got %s", result[0].ID)
		}
	})
}

func TestCredentialState(t *testing.T) {
	t.Run("Create credential state", func(t *testing.T) {
		now := time.Now()
		state := &CredentialState{
			ID:           "state1",
			Disabled:     false,
			LastUsed:     &now,
			FailureCount: 0,
			SuccessCount: 10,
			HealthScore:  0.95,
			ErrorRate:    0.05,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if state.ID != "state1" {
			t.Errorf("Expected ID 'state1', got %s", state.ID)
		}

		if state.HealthScore != 0.95 {
			t.Errorf("Expected health score 0.95, got %f", state.HealthScore)
		}
	})
}

func TestCredential(t *testing.T) {
	t.Run("Create credential", func(t *testing.T) {
		now := time.Now()
		cred := &Credential{
			ID:          "cred1",
			Name:        "Test Credential",
			Type:        "oauth",
			AccessToken: "token123",
			ExpiresAt:   &now,
			Metadata: map[string]interface{}{
				"project": "test-project",
			},
			State: &CredentialState{
				ID:          "cred1",
				HealthScore: 1.0,
			},
		}

		if cred.ID != "cred1" {
			t.Errorf("Expected ID 'cred1', got %s", cred.ID)
		}

		if cred.Type != "oauth" {
			t.Errorf("Expected type 'oauth', got %s", cred.Type)
		}

		if cred.State.HealthScore != 1.0 {
			t.Errorf("Expected health score 1.0, got %f", cred.State.HealthScore)
		}
	})
}

func TestCredentialStats(t *testing.T) {
	t.Run("Create credential stats", func(t *testing.T) {
		now := time.Now()
		stats := &CredentialStats{
			TotalCredentials:    10,
			ActiveCredentials:   8,
			DisabledCredentials: 2,
			HealthScore:         0.85,
			ErrorRate:           0.15,
			TotalUsage:          1000,
			TotalSuccess:        850,
			TotalFailure:        150,
			TypeDistribution: map[string]int{
				"oauth":   6,
				"api_key": 4,
			},
			LastUpdated: now,
		}

		if stats.TotalCredentials != 10 {
			t.Errorf("Expected 10 total credentials, got %d", stats.TotalCredentials)
		}

		if stats.ActiveCredentials != 8 {
			t.Errorf("Expected 8 active credentials, got %d", stats.ActiveCredentials)
		}

		if stats.TypeDistribution["oauth"] != 6 {
			t.Errorf("Expected 6 oauth credentials, got %d", stats.TypeDistribution["oauth"])
		}
	})
}

func TestPointerHelpers(t *testing.T) {
	t.Run("boolPtr", func(t *testing.T) {
		ptr := boolPtr(true)
		if ptr == nil {
			t.Fatal("Expected non-nil pointer")
		}
		if *ptr != true {
			t.Errorf("Expected true, got %v", *ptr)
		}
	})

	t.Run("float64Ptr", func(t *testing.T) {
		ptr := float64Ptr(0.5)
		if ptr == nil {
			t.Fatal("Expected non-nil pointer")
		}
		if *ptr != 0.5 {
			t.Errorf("Expected 0.5, got %f", *ptr)
		}
	})
}
