package adapter

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStorageAdapter_Creation(t *testing.T) {
	t.Run("create with valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := &FileStorageConfig{
			CredentialsDir: filepath.Join(tmpDir, "credentials"),
			StatesDir:      filepath.Join(tmpDir, "states"),
			Config: map[string]interface{}{
				"test_key": "test_value",
			},
		}

		adapter, err := NewFileStorageAdapter(config)
		require.NoError(t, err)
		require.NotNil(t, adapter)

		// Verify directories were created
		assert.DirExists(t, config.CredentialsDir)
		assert.DirExists(t, config.StatesDir)

		// Verify config was set
		cfg := adapter.GetConfig()
		assert.Equal(t, "test_value", cfg["test_key"])
	})

	t.Run("create with nil config returns error", func(t *testing.T) {
		adapter, err := NewFileStorageAdapter(nil)
		assert.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("create with invalid credentials dir", func(t *testing.T) {
		config := &FileStorageConfig{
			CredentialsDir: "/invalid/path/that/cannot/be/created",
			StatesDir:      "/tmp/states",
		}

		adapter, err := NewFileStorageAdapter(config)
		assert.Error(t, err)
		assert.Nil(t, adapter)
	})
}

func TestFileStorageAdapter_CRUD(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileStorageConfig{
		CredentialsDir: filepath.Join(tmpDir, "credentials"),
		StatesDir:      filepath.Join(tmpDir, "states"),
	}

	adapter, err := NewFileStorageAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("store and load credential", func(t *testing.T) {
		now := time.Now()
		cred := &Credential{
			ID:          "test-cred-1",
			Name:        "Test Credential",
			Type:        "oauth",
			AccessToken: "token123",
			ClientID:    "client123",
			ExpiresAt:   &now,
			Metadata: map[string]interface{}{
				"project": "test-project",
			},
		}

		err := adapter.StoreCredential(ctx, cred)
		require.NoError(t, err)

		loaded, err := adapter.LoadCredential(ctx, "test-cred-1")
		require.NoError(t, err)
		assert.Equal(t, cred.ID, loaded.ID)
		assert.Equal(t, cred.Name, loaded.Name)
		assert.Equal(t, cred.Type, loaded.Type)
		assert.Equal(t, cred.ClientID, loaded.ClientID)
	})

	t.Run("load non-existent credential returns error", func(t *testing.T) {
		_, err := adapter.LoadCredential(ctx, "non-existent")
		assert.Error(t, err)
	})

	t.Run("update credential", func(t *testing.T) {
		cred := &Credential{
			ID:   "test-cred-2",
			Name: "Original Name",
			Type: "oauth",
		}

		err := adapter.StoreCredential(ctx, cred)
		require.NoError(t, err)

		cred.Name = "Updated Name"
		err = adapter.UpdateCredential(ctx, cred)
		require.NoError(t, err)

		loaded, err := adapter.LoadCredential(ctx, "test-cred-2")
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", loaded.Name)
	})

	t.Run("delete credential", func(t *testing.T) {
		cred := &Credential{
			ID:   "test-cred-3",
			Name: "To Be Deleted",
			Type: "api_key",
		}

		err := adapter.StoreCredential(ctx, cred)
		require.NoError(t, err)

		err = adapter.DeleteCredential(ctx, "test-cred-3")
		require.NoError(t, err)

		_, err = adapter.LoadCredential(ctx, "test-cred-3")
		assert.Error(t, err)
	})
}

func TestFileStorageAdapter_BatchOperations(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileStorageConfig{
		CredentialsDir: filepath.Join(tmpDir, "credentials"),
		StatesDir:      filepath.Join(tmpDir, "states"),
	}

	adapter, err := NewFileStorageAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Setup test data
	for i := 1; i <= 5; i++ {
		cred := &Credential{
			ID:   fmt.Sprintf("batch-cred-%d", i),
			Name: fmt.Sprintf("Batch Credential %d", i),
			Type: "oauth",
		}
		err := adapter.StoreCredential(ctx, cred)
		require.NoError(t, err)
	}

	t.Run("get all credentials", func(t *testing.T) {
		creds, err := adapter.GetAllCredentials(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(creds), 5)
	})

	t.Run("delete multiple credentials", func(t *testing.T) {
		credIDs := []string{"batch-cred-1", "batch-cred-2"}
		err := adapter.DeleteCredentials(ctx, credIDs)
		require.NoError(t, err)

		_, err = adapter.LoadCredential(ctx, "batch-cred-1")
		assert.Error(t, err)

		_, err = adapter.LoadCredential(ctx, "batch-cred-2")
		assert.Error(t, err)
	})

	t.Run("enable and disable credentials", func(t *testing.T) {
		credIDs := []string{"batch-cred-3", "batch-cred-4"}

		err := adapter.DisableCredentials(ctx, credIDs)
		require.NoError(t, err)

		states, err := adapter.GetAllCredentialStates(ctx)
		require.NoError(t, err)

		for _, id := range credIDs {
			state, exists := states[id]
			require.True(t, exists)
			assert.True(t, state.Disabled)
		}

		err = adapter.EnableCredentials(ctx, credIDs)
		require.NoError(t, err)

		states, err = adapter.GetAllCredentialStates(ctx)
		require.NoError(t, err)

		for _, id := range credIDs {
			state, exists := states[id]
			require.True(t, exists)
			assert.False(t, state.Disabled)
		}
	})
}

func TestFileStorageAdapter_StateManagement(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileStorageConfig{
		CredentialsDir: filepath.Join(tmpDir, "credentials"),
		StatesDir:      filepath.Join(tmpDir, "states"),
	}

	adapter, err := NewFileStorageAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("update credential states", func(t *testing.T) {
		cred := &Credential{
			ID:   "state-test-1",
			Name: "State Test",
			Type: "oauth",
		}
		err := adapter.StoreCredential(ctx, cred)
		require.NoError(t, err)

		now := time.Now()
		states := map[string]*CredentialState{
			"state-test-1": {
				ID:           "state-test-1",
				Disabled:     false,
				LastUsed:     &now,
				SuccessCount: 10,
				FailureCount: 2,
				HealthScore:  0.85,
				ErrorRate:    0.15,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		}

		err = adapter.UpdateCredentialStates(ctx, states)
		require.NoError(t, err)

		allStates, err := adapter.GetAllCredentialStates(ctx)
		require.NoError(t, err)

		state, exists := allStates["state-test-1"]
		require.True(t, exists)
		assert.Equal(t, 10, state.SuccessCount)
		assert.Equal(t, 2, state.FailureCount)
		assert.Equal(t, 0.85, state.HealthScore)
	})
}

func TestFileStorageAdapter_UsageStats(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileStorageConfig{
		CredentialsDir: filepath.Join(tmpDir, "credentials"),
		StatesDir:      filepath.Join(tmpDir, "states"),
	}

	adapter, err := NewFileStorageAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()

	cred := &Credential{
		ID:   "usage-test-1",
		Name: "Usage Test",
		Type: "oauth",
	}
	err = adapter.StoreCredential(ctx, cred)
	require.NoError(t, err)

	t.Run("update and get usage stats", func(t *testing.T) {
		stats := map[string]interface{}{
			"requests":       100,
			"tokens_used":    5000,
			"avg_latency_ms": 250,
		}

		err := adapter.UpdateUsageStats(ctx, "usage-test-1", stats)
		require.NoError(t, err)

		retrieved, err := adapter.GetUsageStats(ctx, "usage-test-1")
		require.NoError(t, err)
		assert.Equal(t, 100, retrieved["requests"])
		assert.Equal(t, 5000, retrieved["tokens_used"])
	})
}

func TestFileStorageAdapter_HealthChecks(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileStorageConfig{
		CredentialsDir: filepath.Join(tmpDir, "credentials"),
		StatesDir:      filepath.Join(tmpDir, "states"),
	}

	adapter, err := NewFileStorageAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create healthy and unhealthy credentials
	healthyCred := &Credential{
		ID:   "healthy-1",
		Name: "Healthy Credential",
		Type: "oauth",
		State: &CredentialState{
			ID:          "healthy-1",
			Disabled:    false,
			HealthScore: 0.9,
			ErrorRate:   0.1,
		},
	}

	unhealthyCred := &Credential{
		ID:   "unhealthy-1",
		Name: "Unhealthy Credential",
		Type: "oauth",
		State: &CredentialState{
			ID:          "unhealthy-1",
			Disabled:    false,
			HealthScore: 0.3,
			ErrorRate:   0.7,
		},
	}

	disabledCred := &Credential{
		ID:   "disabled-1",
		Name: "Disabled Credential",
		Type: "oauth",
		State: &CredentialState{
			ID:          "disabled-1",
			Disabled:    true,
			HealthScore: 0.8,
			ErrorRate:   0.2,
		},
	}

	err = adapter.StoreCredential(ctx, healthyCred)
	require.NoError(t, err)
	err = adapter.StoreCredential(ctx, unhealthyCred)
	require.NoError(t, err)
	err = adapter.StoreCredential(ctx, disabledCred)
	require.NoError(t, err)

	t.Run("get healthy credentials", func(t *testing.T) {
		healthy, err := adapter.GetHealthyCredentials(ctx)
		require.NoError(t, err)

		// Should include healthy-1 but not unhealthy-1 or disabled-1
		healthyIDs := make(map[string]bool)
		for _, cred := range healthy {
			healthyIDs[cred.ID] = true
		}

		assert.True(t, healthyIDs["healthy-1"], "healthy-1 should be in healthy credentials")
		assert.False(t, healthyIDs["unhealthy-1"], "unhealthy-1 should not be in healthy credentials")
		assert.False(t, healthyIDs["disabled-1"], "disabled-1 should not be in healthy credentials")
	})

	t.Run("get unhealthy credentials", func(t *testing.T) {
		// GetUnhealthyCredentials returns disabled credentials
		unhealthy, err := adapter.GetUnhealthyCredentials(ctx)
		require.NoError(t, err)

		unhealthyIDs := make(map[string]bool)
		for _, cred := range unhealthy {
			unhealthyIDs[cred.ID] = true
		}

		// disabled-1 should be in unhealthy credentials (it's disabled)
		assert.True(t, unhealthyIDs["disabled-1"], "disabled-1 should be in unhealthy credentials")
	})

	t.Run("ping returns no error", func(t *testing.T) {
		err := adapter.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("close returns no error", func(t *testing.T) {
		err := adapter.Close()
		assert.NoError(t, err)
	})
}

func TestFileStorageAdapter_DiscoverCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileStorageConfig{
		CredentialsDir: filepath.Join(tmpDir, "credentials"),
		StatesDir:      filepath.Join(tmpDir, "states"),
	}

	adapter, err := NewFileStorageAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("discover credentials", func(t *testing.T) {
		// Store some credentials
		for i := 1; i <= 3; i++ {
			cred := &Credential{
				ID:   fmt.Sprintf("discover-%d", i),
				Name: fmt.Sprintf("Discover %d", i),
				Type: "oauth",
			}
			err := adapter.StoreCredential(ctx, cred)
			require.NoError(t, err)
		}

		discovered, err := adapter.DiscoverCredentials(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(discovered), 3)
	})
}

func TestFileStorageAdapter_ConfigManagement(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileStorageConfig{
		CredentialsDir: filepath.Join(tmpDir, "credentials"),
		StatesDir:      filepath.Join(tmpDir, "states"),
		Config: map[string]interface{}{
			"initial_key": "initial_value",
		},
	}

	adapter, err := NewFileStorageAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("get initial config", func(t *testing.T) {
		cfg := adapter.GetConfig()
		assert.Equal(t, "initial_value", cfg["initial_key"])
	})

	t.Run("set new config", func(t *testing.T) {
		newConfig := map[string]interface{}{
			"new_key": "new_value",
			"number":  42,
		}

		err := adapter.SetConfig(ctx, newConfig)
		require.NoError(t, err)

		cfg := adapter.GetConfig()
		assert.Equal(t, "new_value", cfg["new_key"])
		assert.Equal(t, 42, cfg["number"])
	})

	t.Run("config is copied not referenced", func(t *testing.T) {
		cfg1 := adapter.GetConfig()
		cfg1["modified"] = "value"

		cfg2 := adapter.GetConfig()
		_, exists := cfg2["modified"]
		assert.False(t, exists, "config should be copied, not referenced")
	})
}

func TestFileStorageAdapter_ValidateCredential(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileStorageConfig{
		CredentialsDir: filepath.Join(tmpDir, "credentials"),
		StatesDir:      filepath.Join(tmpDir, "states"),
	}

	adapter, err := NewFileStorageAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("validate credential with missing ID", func(t *testing.T) {
		cred := &Credential{
			Name: "No ID",
			Type: "oauth",
		}

		err := adapter.ValidateCredential(ctx, cred)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ID cannot be empty")
	})

	t.Run("validate credential with missing type", func(t *testing.T) {
		cred := &Credential{
			ID:   "test-id",
			Name: "No Type",
		}

		err := adapter.ValidateCredential(ctx, cred)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type cannot be empty")
	})

	t.Run("validate oauth credential without token", func(t *testing.T) {
		cred := &Credential{
			ID:   "oauth-no-token",
			Type: "oauth",
		}

		err := adapter.ValidateCredential(ctx, cred)
		assert.Error(t, err)
	})

	t.Run("validate api_key credential without token", func(t *testing.T) {
		cred := &Credential{
			ID:   "apikey-no-token",
			Type: "api_key",
		}

		err := adapter.ValidateCredential(ctx, cred)
		assert.Error(t, err)
	})
}
