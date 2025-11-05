package openai

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"gcli2api-go/internal/config"
	credpkg "gcli2api-go/internal/credential"
)

func setupTestOpenAIHandler(t *testing.T) *Handler {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	handler := New(cfg, nil, nil, nil, nil)
	return handler
}

func TestNewWithStrategy(t *testing.T) {
	cfg := &config.Config{}

	t.Run("create handler with nil strategy", func(t *testing.T) {
		handler := NewWithStrategy(cfg, nil, nil, nil, nil, nil)
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.router)
	})

	t.Run("create handler with existing strategy", func(t *testing.T) {
		// Create a strategy first
		handler1 := New(cfg, nil, nil, nil, nil)
		strategy := handler1.router

		// Create handler with existing strategy
		handler2 := NewWithStrategy(cfg, nil, nil, nil, nil, strategy)
		assert.NotNil(t, handler2)
		assert.Equal(t, strategy, handler2.router)
	})
}

func TestInvalidateCachesFor(t *testing.T) {
	handler := setupTestOpenAIHandler(t)

	t.Run("invalidate caches for empty credential ID", func(t *testing.T) {
		// Should not panic
		handler.InvalidateCachesFor("")
	})

	t.Run("invalidate caches for non-existent credential", func(t *testing.T) {
		// Should not panic
		handler.InvalidateCachesFor("non-existent-cred")
	})

	t.Run("invalidate caches for credential", func(t *testing.T) {
		// Add a credential to cache
		handler.clientCache["test-cred-1"] = handler.baseClient

		// Verify it's in cache
		handler.cacheMu.RLock()
		_, exists := handler.clientCache["test-cred-1"]
		handler.cacheMu.RUnlock()
		assert.True(t, exists)

		// Invalidate cache
		handler.InvalidateCachesFor("test-cred-1")

		// Verify it's removed
		handler.cacheMu.RLock()
		_, exists = handler.clientCache["test-cred-1"]
		handler.cacheMu.RUnlock()
		assert.False(t, exists)
	})
}

func TestGetClientFor(t *testing.T) {
	handler := setupTestOpenAIHandler(t)

	t.Run("get client for nil credential returns base client", func(t *testing.T) {
		client := handler.getClientFor(nil)
		assert.NotNil(t, client)
		assert.Equal(t, handler.baseClient, client)
	})

	t.Run("get client for empty credential ID returns base client", func(t *testing.T) {
		cred := &credpkg.Credential{ID: ""}
		client := handler.getClientFor(cred)
		assert.NotNil(t, client)
		assert.Equal(t, handler.baseClient, client)
	})

	t.Run("get client for valid credential creates and caches", func(t *testing.T) {
		cred := &credpkg.Credential{
			ID:          "test-cred-2",
			AccessToken: "test-token",
			ProjectID:   "test-project",
		}

		// First call should create and cache
		client1 := handler.getClientFor(cred)
		assert.NotNil(t, client1)

		// Verify it's cached
		handler.cacheMu.RLock()
		cached, exists := handler.clientCache["test-cred-2"]
		handler.cacheMu.RUnlock()
		assert.True(t, exists)
		assert.Equal(t, client1, cached)

		// Second call should return cached client
		client2 := handler.getClientFor(cred)
		assert.Equal(t, client1, client2)
	})
}

func TestGetUpstreamClient(t *testing.T) {
	handler := setupTestOpenAIHandler(t)

	t.Run("get upstream client without credential manager", func(t *testing.T) {
		handler.credMgr = nil

		ctx := context.Background()
		client, cred := handler.getUpstreamClient(ctx)

		assert.NotNil(t, client)
		assert.Nil(t, cred)
		assert.Equal(t, handler.baseClient, client)
	})

	t.Run("get upstream client with credential manager but no router", func(t *testing.T) {
		handler.credMgr = credpkg.NewManager(credpkg.Options{})
		handler.router = nil

		ctx := context.Background()
		client, _ := handler.getUpstreamClient(ctx)

		assert.NotNil(t, client)
	})
}

func TestShouldRefreshAhead(t *testing.T) {
	handler := setupTestOpenAIHandler(t)

	t.Run("should refresh ahead for credential", func(t *testing.T) {
		cred := &credpkg.Credential{
			ID:          "test-cred",
			AccessToken: "test-token",
		}

		// Should not panic
		result := handler.shouldRefreshAhead(cred)
		assert.IsType(t, false, result)
	})
}

func TestInvalidateClientCache(t *testing.T) {
	handler := setupTestOpenAIHandler(t)

	t.Run("invalidate empty credential ID does nothing", func(t *testing.T) {
		handler.invalidateClientCache("")
		// Should not panic
	})

	t.Run("invalidate existing credential", func(t *testing.T) {
		handler.clientCache["test-cred"] = handler.baseClient
		handler.invalidateClientCache("test-cred")

		handler.cacheMu.RLock()
		_, exists := handler.clientCache["test-cred"]
		handler.cacheMu.RUnlock()
		assert.False(t, exists)
	})
}

func TestInvalidateProviderCache(t *testing.T) {
	handler := setupTestOpenAIHandler(t)

	t.Run("invalidate empty credential ID does nothing", func(t *testing.T) {
		handler.invalidateProviderCache("")
		// Should not panic
	})

	t.Run("invalidate with nil providers does nothing", func(t *testing.T) {
		handler.providers = nil
		handler.invalidateProviderCache("test-cred")
		// Should not panic
	})
}

func TestAcquireCredential(t *testing.T) {
	handler := setupTestOpenAIHandler(t)

	t.Run("acquire credential without credential manager", func(t *testing.T) {
		handler.credMgr = nil
		ctx := context.Background()

		cred, err := handler.acquireCredential(ctx)
		assert.NoError(t, err)
		assert.Nil(t, cred)
	})

	t.Run("acquire credential with credential manager", func(t *testing.T) {
		handler.credMgr = credpkg.NewManager(credpkg.Options{})
		ctx := context.Background()

		cred, err := handler.acquireCredential(ctx)
		// May return error if no credentials available
		_ = err
		_ = cred
	})
}

func TestChunkText(t *testing.T) {
	t.Run("chunk empty string", func(t *testing.T) {
		result := chunkText("", 10)
		assert.Len(t, result, 1)
		assert.Equal(t, "", result[0])
	})

	t.Run("chunk short string", func(t *testing.T) {
		result := chunkText("hello", 10)
		assert.Len(t, result, 1)
		assert.Equal(t, "hello", result[0])
	})

	t.Run("chunk long string", func(t *testing.T) {
		result := chunkText("hello world this is a test", 5)
		assert.Greater(t, len(result), 1)

		// Verify all chunks combined equal original
		combined := ""
		for _, chunk := range result {
			combined += chunk
		}
		assert.Equal(t, "hello world this is a test", combined)
	})

	t.Run("chunk with zero size uses default", func(t *testing.T) {
		result := chunkText("hello world", 0)
		assert.Greater(t, len(result), 0)
	})

	t.Run("chunk with negative size uses default", func(t *testing.T) {
		result := chunkText("hello world", -1)
		assert.Greater(t, len(result), 0)
	})

	t.Run("chunk unicode string", func(t *testing.T) {
		result := chunkText("你好世界", 2)
		assert.Greater(t, len(result), 1)

		// Verify all chunks combined equal original
		combined := ""
		for _, chunk := range result {
			combined += chunk
		}
		assert.Equal(t, "你好世界", combined)
	})
}
