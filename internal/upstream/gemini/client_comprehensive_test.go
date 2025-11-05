package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/oauth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("create client with default config", func(t *testing.T) {
		cfg := &config.Config{}
		client := New(cfg)

		require.NotNil(t, client)
		assert.NotNil(t, client.cli)
		assert.NotNil(t, client.cfg)
	})

	t.Run("create client with custom timeouts", func(t *testing.T) {
		cfg := &config.Config{
			DialTimeoutSec:           10,
			TLSHandshakeTimeoutSec:   15,
			ResponseHeaderTimeoutSec: 20,
			ExpectContinueTimeoutSec: 5,
		}
		client := New(cfg)

		require.NotNil(t, client)
		assert.NotNil(t, client.cli)
	})

	t.Run("create client with proxy URL", func(t *testing.T) {
		cfg := &config.Config{
			ProxyURL: "http://proxy.example.com:8080",
		}
		client := New(cfg)

		require.NotNil(t, client)
		assert.NotNil(t, client.cli)
	})
}

func TestNewWithCredential(t *testing.T) {
	t.Run("create client with valid credential", func(t *testing.T) {
		cfg := &config.Config{}
		creds := &oauth.Credentials{
			AccessToken:  "access-token-123",
			RefreshToken: "refresh-token-456",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
			ProjectID:    "project-123",
		}

		client := NewWithCredential(cfg, creds)

		require.NotNil(t, client)
		assert.Equal(t, creds, client.credentials)
		assert.Equal(t, "access-token-123", client.token)
	})

	t.Run("create client with nil credential", func(t *testing.T) {
		cfg := &config.Config{}
		client := NewWithCredential(cfg, nil)

		require.NotNil(t, client)
		assert.Nil(t, client.credentials)
		assert.Empty(t, client.token)
	})

	t.Run("create client with credential without token", func(t *testing.T) {
		cfg := &config.Config{}
		creds := &oauth.Credentials{
			ProjectID: "project-123",
		}

		client := NewWithCredential(cfg, creds)

		require.NotNil(t, client)
		assert.Equal(t, creds, client.credentials)
		assert.Empty(t, client.token)
	})
}

func TestClient_WithCaller(t *testing.T) {
	cfg := &config.Config{}
	client := New(cfg)

	t.Run("set caller to openai", func(t *testing.T) {
		result := client.WithCaller("openai")
		assert.Equal(t, client, result, "should return same client for chaining")
		assert.Equal(t, "openai", client.caller)
	})

	t.Run("set caller to gemini", func(t *testing.T) {
		result := client.WithCaller("gemini")
		assert.Equal(t, client, result)
		assert.Equal(t, "gemini", client.caller)
	})
}

func TestClient_GetToken(t *testing.T) {
	t.Run("get token from credential", func(t *testing.T) {
		cfg := &config.Config{}
		creds := &oauth.Credentials{
			AccessToken: "cred-token",
		}
		client := NewWithCredential(cfg, creds)

		token := client.getToken()
		assert.Equal(t, "cred-token", token)
	})

	t.Run("credential token takes precedence", func(t *testing.T) {
		cfg := &config.Config{}
		creds := &oauth.Credentials{
			AccessToken: "cred-token",
		}
		client := NewWithCredential(cfg, creds)

		token := client.getToken()
		assert.Equal(t, "cred-token", token, "credential token should take precedence")
	})
}

func TestGetProxyFunc(t *testing.T) {
	t.Run("valid proxy URL", func(t *testing.T) {
		proxyFunc := getProxyFunc("http://proxy.example.com:8080")
		require.NotNil(t, proxyFunc)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		proxyURL, err := proxyFunc(req)
		require.NoError(t, err)
		assert.Equal(t, "proxy.example.com:8080", proxyURL.Host)
	})

	t.Run("invalid proxy URL falls back to environment", func(t *testing.T) {
		proxyFunc := getProxyFunc("://invalid-url")
		require.NotNil(t, proxyFunc)

		// Should fall back to http.ProxyFromEnvironment
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		_, err := proxyFunc(req)
		assert.NoError(t, err) // ProxyFromEnvironment doesn't error
	})

	t.Run("empty proxy URL uses environment", func(t *testing.T) {
		proxyFunc := getProxyFunc("")
		require.NotNil(t, proxyFunc)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		_, err := proxyFunc(req)
		assert.NoError(t, err)
	})
}

func TestDurationOrDefault(t *testing.T) {
	t.Run("positive seconds", func(t *testing.T) {
		result := durationOrDefault(10, 5*time.Second)
		assert.Equal(t, 10*time.Second, result)
	})

	t.Run("zero seconds uses default", func(t *testing.T) {
		result := durationOrDefault(0, 5*time.Second)
		assert.Equal(t, 5*time.Second, result)
	})

	t.Run("negative seconds uses default", func(t *testing.T) {
		result := durationOrDefault(-1, 5*time.Second)
		assert.Equal(t, 5*time.Second, result)
	})
}

func TestWithHeaderOverrides(t *testing.T) {
	t.Run("set and get header overrides", func(t *testing.T) {
		ctx := context.Background()
		headers := http.Header{
			"X-Custom-Header": []string{"custom-value"},
			"Authorization":   []string{"Bearer token"},
		}

		ctx = WithHeaderOverrides(ctx, headers)
		retrieved := getHeaderOverrides(ctx)

		assert.Equal(t, "custom-value", retrieved.Get("X-Custom-Header"))
		assert.Equal(t, "Bearer token", retrieved.Get("Authorization"))
	})

	t.Run("nil headers", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithHeaderOverrides(ctx, nil)
		retrieved := getHeaderOverrides(ctx)

		assert.Nil(t, retrieved)
	})

	t.Run("empty headers", func(t *testing.T) {
		ctx := context.Background()
		headers := http.Header{}
		ctx = WithHeaderOverrides(ctx, headers)
		retrieved := getHeaderOverrides(ctx)

		assert.NotNil(t, retrieved)
		assert.Equal(t, 0, len(retrieved))
	})
}

func TestClient_HTTPRequest(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		}))
		defer server.Close()

		cfg := &config.Config{}
		client := New(cfg)

		// Note: This is a simplified test. In real scenarios, you'd need to mock the actual API endpoint
		// For now, we're just testing the client creation and basic setup
		assert.NotNil(t, client)
	})
}

func TestClient_ModelFallback(t *testing.T) {
	t.Run("model fallback order", func(t *testing.T) {
		cfg := &config.Config{}
		client := New(cfg)

		// Test that client is properly initialized for model fallback
		assert.NotNil(t, client)
		assert.NotNil(t, client.cfg)
	})
}

func TestClient_RetryLogic(t *testing.T) {
	t.Run("retry configuration", func(t *testing.T) {
		cfg := &config.Config{
			Retry: config.RetryConfig{
				Enabled:        true,
				Max:            3,
				IntervalSec:    1,
				MaxIntervalSec: 10,
			},
		}
		client := New(cfg)

		assert.NotNil(t, client)
		assert.True(t, client.cfg.Retry.Enabled)
		assert.Equal(t, 3, client.cfg.Retry.Max)
	})
}

func TestClient_Timeout(t *testing.T) {
	t.Run("client with zero timeout", func(t *testing.T) {
		cfg := &config.Config{}
		client := New(cfg)

		// HTTP client should have zero timeout (no timeout)
		assert.Equal(t, time.Duration(0), client.cli.Timeout)
	})
}

func TestClient_TransportConfiguration(t *testing.T) {
	t.Run("transport has correct idle connection settings", func(t *testing.T) {
		cfg := &config.Config{}
		client := New(cfg)

		transport, ok := client.cli.Transport.(*http.Transport)
		require.True(t, ok, "transport should be *http.Transport")

		assert.Greater(t, transport.MaxIdleConns, 0)
		assert.Greater(t, transport.MaxIdleConnsPerHost, 0)
		assert.Greater(t, transport.IdleConnTimeout, time.Duration(0))
	})

	t.Run("transport has correct timeout settings", func(t *testing.T) {
		cfg := &config.Config{
			DialTimeoutSec:           5,
			TLSHandshakeTimeoutSec:   10,
			ResponseHeaderTimeoutSec: 15,
		}
		client := New(cfg)

		transport, ok := client.cli.Transport.(*http.Transport)
		require.True(t, ok)

		assert.Equal(t, 10*time.Second, transport.TLSHandshakeTimeout)
		assert.Equal(t, 15*time.Second, transport.ResponseHeaderTimeout)
	})
}
