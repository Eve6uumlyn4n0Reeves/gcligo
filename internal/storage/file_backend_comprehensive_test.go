package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileBackend_NewAndInitialize(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("create new file backend", func(t *testing.T) {
		backend := NewFileBackend(tmpDir)
		require.NotNil(t, backend)
		assert.Equal(t, tmpDir, backend.baseDir)
		assert.NotNil(t, backend.credentials)
		assert.NotNil(t, backend.config)
		assert.NotNil(t, backend.usage)
	})

	t.Run("initialize creates directories", func(t *testing.T) {
		backend := NewFileBackend(tmpDir)
		ctx := context.Background()

		err := backend.Initialize(ctx)
		require.NoError(t, err)

		// Verify directories created
		assert.DirExists(t, filepath.Join(tmpDir, "credentials"))
		assert.DirExists(t, filepath.Join(tmpDir, "config"))
		assert.DirExists(t, filepath.Join(tmpDir, "usage"))
	})
}

func TestFileBackend_CredentialCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)
	ctx := context.Background()

	err := backend.Initialize(ctx)
	require.NoError(t, err)
	defer backend.Close()

	t.Run("set and get credential", func(t *testing.T) {
		credID := "test-cred-1"
		credData := map[string]interface{}{
			"id":            credID,
			"email":         "test@example.com",
			"access_token":  "access-123",
			"refresh_token": "refresh-456",
			"project_id":    "project-789",
		}

		err := backend.SetCredential(ctx, credID, credData)
		require.NoError(t, err)

		retrieved, err := backend.GetCredential(ctx, credID)
		require.NoError(t, err)
		assert.Equal(t, credID, retrieved["id"])
		assert.Equal(t, "test@example.com", retrieved["email"])
		assert.Equal(t, "access-123", retrieved["access_token"])
	})

	t.Run("get non-existent credential returns error", func(t *testing.T) {
		_, err := backend.GetCredential(ctx, "non-existent")
		require.Error(t, err)
		assert.IsType(t, &ErrNotFound{}, err)
	})

	t.Run("list credentials", func(t *testing.T) {
		// Add multiple credentials
		for i := 1; i <= 3; i++ {
			credID := string(rune('A' + i))
			credData := map[string]interface{}{
				"id":    credID,
				"email": "test@example.com",
			}
			err := backend.SetCredential(ctx, credID, credData)
			require.NoError(t, err)
		}

		ids, err := backend.ListCredentials(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(ids), 3)
	})

	t.Run("update existing credential", func(t *testing.T) {
		credID := "update-test"
		originalData := map[string]interface{}{
			"id":    credID,
			"value": "original",
		}

		err := backend.SetCredential(ctx, credID, originalData)
		require.NoError(t, err)

		updatedData := map[string]interface{}{
			"id":    credID,
			"value": "updated",
		}

		err = backend.SetCredential(ctx, credID, updatedData)
		require.NoError(t, err)

		retrieved, err := backend.GetCredential(ctx, credID)
		require.NoError(t, err)
		assert.Equal(t, "updated", retrieved["value"])
	})

	t.Run("delete credential", func(t *testing.T) {
		credID := "delete-test"
		credData := map[string]interface{}{
			"id": credID,
		}

		err := backend.SetCredential(ctx, credID, credData)
		require.NoError(t, err)

		err = backend.DeleteCredential(ctx, credID)
		require.NoError(t, err)

		_, err = backend.GetCredential(ctx, credID)
		assert.Error(t, err)
	})
}

func TestFileBackend_ConfigOperations(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)
	ctx := context.Background()

	err := backend.Initialize(ctx)
	require.NoError(t, err)
	defer backend.Close()

	t.Run("set and get config", func(t *testing.T) {
		key := "test-config"
		value := map[string]interface{}{
			"setting1": "value1",
			"setting2": 123,
			"setting3": true,
		}

		err := backend.SetConfig(ctx, key, value)
		require.NoError(t, err)

		retrieved, err := backend.GetConfig(ctx, key)
		require.NoError(t, err)
		retrievedMap, ok := retrieved.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value1", retrievedMap["setting1"])
		assert.Equal(t, 123, retrievedMap["setting2"]) // int, not float64 (not from JSON)
		assert.Equal(t, true, retrievedMap["setting3"])
	})

	t.Run("get non-existent config returns error", func(t *testing.T) {
		_, err := backend.GetConfig(ctx, "non-existent")
		require.Error(t, err)
		assert.IsType(t, &ErrNotFound{}, err)
	})

	t.Run("list configs", func(t *testing.T) {
		// Add multiple configs
		for i := 1; i <= 3; i++ {
			key := string(rune('A' + i))
			value := map[string]interface{}{"index": i}
			err := backend.SetConfig(ctx, key, value)
			require.NoError(t, err)
		}

		keys, err := backend.ListConfigs(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(keys), 3)
	})

	t.Run("delete config", func(t *testing.T) {
		key := "delete-config"
		value := map[string]interface{}{"test": "value"}

		err := backend.SetConfig(ctx, key, value)
		require.NoError(t, err)

		err = backend.DeleteConfig(ctx, key)
		require.NoError(t, err)

		_, err = backend.GetConfig(ctx, key)
		assert.Error(t, err)
	})
}

func TestFileBackend_UsageOperations(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)
	ctx := context.Background()

	err := backend.Initialize(ctx)
	require.NoError(t, err)
	defer backend.Close()

	t.Run("increment usage", func(t *testing.T) {
		key := "usage-test"

		// Increment multiple times
		for i := 0; i < 5; i++ {
			err := backend.IncrementUsage(ctx, key, "requests", 1)
			require.NoError(t, err)
		}

		usage, err := backend.GetUsage(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(5), usage["requests"])
	})

	t.Run("increment multiple fields", func(t *testing.T) {
		key := "multi-field"

		err := backend.IncrementUsage(ctx, key, "requests", 10)
		require.NoError(t, err)

		err = backend.IncrementUsage(ctx, key, "tokens", 1000)
		require.NoError(t, err)

		usage, err := backend.GetUsage(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(10), usage["requests"])
		assert.Equal(t, int64(1000), usage["tokens"])
	})

	t.Run("reset usage", func(t *testing.T) {
		key := "reset-test"

		err := backend.IncrementUsage(ctx, key, "count", 100)
		require.NoError(t, err)

		err = backend.ResetUsage(ctx, key)
		require.NoError(t, err)

		// After reset, GetUsage should return error (key not found)
		_, err = backend.GetUsage(ctx, key)
		assert.Error(t, err)
		assert.IsType(t, &ErrNotFound{}, err)
	})

	t.Run("list usage", func(t *testing.T) {
		// Add usage for multiple keys
		for i := 1; i <= 3; i++ {
			key := string(rune('A' + i))
			err := backend.IncrementUsage(ctx, key, "count", int64(i))
			require.NoError(t, err)
		}

		usageMap, err := backend.ListUsage(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(usageMap), 3)
	})
}

func TestFileBackend_BatchOperations(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)
	ctx := context.Background()

	err := backend.Initialize(ctx)
	require.NoError(t, err)
	defer backend.Close()

	t.Run("batch get credentials", func(t *testing.T) {
		// Setup: create multiple credentials
		ids := []string{"batch-1", "batch-2", "batch-3"}
		for _, id := range ids {
			data := map[string]interface{}{
				"id":    id,
				"email": id + "@example.com",
			}
			err := backend.SetCredential(ctx, id, data)
			require.NoError(t, err)
		}

		// Batch get
		results, err := backend.BatchGetCredentials(ctx, ids)
		require.NoError(t, err)
		assert.Equal(t, 3, len(results))

		for _, id := range ids {
			assert.Contains(t, results, id)
			assert.Equal(t, id, results[id]["id"])
		}
	})

	t.Run("batch get with non-existent IDs", func(t *testing.T) {
		ids := []string{"exists", "non-existent"}

		// Create only one
		data := map[string]interface{}{"id": "exists"}
		err := backend.SetCredential(ctx, "exists", data)
		require.NoError(t, err)

		results, err := backend.BatchGetCredentials(ctx, ids)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results), "should only return existing credentials")
		assert.Contains(t, results, "exists")
		assert.NotContains(t, results, "non-existent")
	})

	t.Run("batch set credentials", func(t *testing.T) {
		batch := map[string]map[string]interface{}{
			"batch-set-1": {"id": "batch-set-1", "value": "one"},
			"batch-set-2": {"id": "batch-set-2", "value": "two"},
			"batch-set-3": {"id": "batch-set-3", "value": "three"},
		}

		err := backend.BatchSetCredentials(ctx, batch)
		require.NoError(t, err)

		// Verify all were set
		for id := range batch {
			retrieved, err := backend.GetCredential(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, id, retrieved["id"])
		}
	})
}

func TestFileBackend_Health(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFileBackend(tmpDir)
	ctx := context.Background()

	err := backend.Initialize(ctx)
	require.NoError(t, err)

	t.Run("healthy backend", func(t *testing.T) {
		err := backend.Health(ctx)
		assert.NoError(t, err)
	})

	t.Run("unhealthy after directory removal", func(t *testing.T) {
		backend.Close()
		os.RemoveAll(tmpDir)

		err := backend.Health(ctx)
		assert.Error(t, err)
	})
}

func TestFileBackend_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create first backend and save data
	backend1 := NewFileBackend(tmpDir)
	err := backend1.Initialize(ctx)
	require.NoError(t, err)

	credData := map[string]interface{}{
		"id":    "persist-test",
		"email": "persist@example.com",
	}
	err = backend1.SetCredential(ctx, "persist-test", credData)
	require.NoError(t, err)

	configData := map[string]interface{}{
		"key": "value",
	}
	err = backend1.SetConfig(ctx, "persist-config", configData)
	require.NoError(t, err)

	err = backend1.Close()
	require.NoError(t, err)

	// Create second backend and verify data persisted
	backend2 := NewFileBackend(tmpDir)
	err = backend2.Initialize(ctx)
	require.NoError(t, err)
	defer backend2.Close()

	retrieved, err := backend2.GetCredential(ctx, "persist-test")
	require.NoError(t, err)
	assert.Equal(t, "persist@example.com", retrieved["email"])

	retrievedConfig, err := backend2.GetConfig(ctx, "persist-config")
	require.NoError(t, err)
	retrievedConfigMap, ok := retrievedConfig.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", retrievedConfigMap["key"])
}
