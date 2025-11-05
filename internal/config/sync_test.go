package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncToDomains verifies that top-level fields are correctly synced to domain structures
func TestSyncToDomains(t *testing.T) {
	cfg := &Config{
		// Server
		OpenAIPort:      "8317",
		GeminiPort:      "8318",
		BasePath:        "/api",
		WebAdminEnabled: true,
		RunProfile:      "production",

		// Upstream
		OpenAIKey:        "sk-test",
		GeminiKey:        "gk-test",
		CodeAssist:       "https://code-assist.example.com",
		GoogleToken:      "token-123",
		GoogleProjID:     "proj-456",
		UpstreamProvider: "gemini",

		// Security
		ManagementKey:            "mgmt-key",
		ManagementKeyHash:        "hash-123",
		ManagementReadOnly:       true,
		ManagementAllowRemote:    true,
		ManagementRemoteTTlHours: 24,
		ManagementRemoteAllowIPs: []string{"192.168.1.1", "10.0.0.1"},
		AuthDir:                  "/auth",
		HeaderPassThrough:        true,
		Debug:                    true,
		LogFile:                  "/var/log/app.log",

		// Execution
		CallsPerRotation:           100,
		MaxConcurrentPerCredential: 5,
		AutoLoadEnvCreds:           true,

		// Storage
		StorageBackend: "redis",
		StorageBaseDir: "/data",
		RedisAddr:      "localhost:6379",
		RedisPassword:  "secret",
		RedisDB:        1,
		RedisPrefix:    "gcli:",
		MongoURI:       "mongodb://localhost",
		MongoDatabase:  "gcli",
		PostgresDSN:    "postgres://user:pass@localhost/db",
		GitRemoteURL:   "https://github.com/user/repo",
		GitBranch:      "main",
		GitUsername:    "git-user",
		GitPassword:    "git-pass",
		GitAuthorName:  "Author",
		GitAuthorEmail: "author@example.com",

		// Retry
		RetryEnabled:             true,
		RetryMax:                 3,
		RetryIntervalSec:         1,
		RetryMaxIntervalSec:      10,
		RetryOn5xx:               true,
		RetryOnNetworkError:      true,
		DialTimeoutSec:           30,
		TLSHandshakeTimeoutSec:   10,
		ResponseHeaderTimeoutSec: 30,
		ExpectContinueTimeoutSec: 1,

		// RateLimit
		RateLimitEnabled:        true,
		RateLimitRPS:            100,
		RateLimitBurst:          200,
		UsageResetIntervalHours: 24,
		UsageResetTimezone:      "UTC",
		UsageResetHourLocal:     0,

		// APICompat
		OpenAIImagesIncludeMIME: true,
		ToolArgsDeltaChunk:      512,
		PreferredBaseModels:     []string{"gemini-2.5-pro", "gemini-2.5-flash"},
		DisabledModels:          []string{"old-model"},
		DisableModelVariants:    false,

		// ResponseShaping
		AntiTruncationMax:      3,
		AntiTruncationEnabled:  true,
		FakeStreamingEnabled:   true,
		FakeStreamingChunkSize: 100,
		FakeStreamingDelayMs:   50,
		AutoImagePlaceholder:   true,
		RequestLogEnabled:      true,
		PprofEnabled:           false,
		ProxyURL:               "http://proxy:8080",
		SanitizerEnabled:       true,
		SanitizerPatterns:      []string{"pattern1", "pattern2"},

		// OAuth
		OAuthClientID:     "client-id",
		OAuthClientSecret: "client-secret",
		OAuthRedirectURL:  "https://example.com/callback",

		// AutoBan
		AutoBanEnabled:          true,
		AutoBan429Threshold:     10,
		AutoBan403Threshold:     5,
		AutoBan401Threshold:     3,
		AutoBan5xxThreshold:     20,
		AutoBanConsecutiveFails: 5,
		AutoRecoveryEnabled:     true,
		AutoRecoveryIntervalMin: 60,

		// AutoProbe
		AutoProbeEnabled:             true,
		AutoProbeHourUTC:             2,
		AutoProbeModel:               "gemini-2.5-flash",
		AutoProbeTimeoutSec:          30,
		AutoProbeDisableThresholdPct: 50,

		// Routing
		StickyTTLSeconds:          3600,
		RouterCooldownBaseMS:      1000,
		RouterCooldownMaxMS:       60000,
		PersistRoutingState:       true,
		RoutingPersistIntervalSec: 300,
		RoutingDebugHeaders:       true,
	}

	cfg.SyncToDomains()

	// Verify Server
	assert.Equal(t, "8317", cfg.Server.OpenAIPort)
	assert.Equal(t, "8318", cfg.Server.GeminiPort)
	assert.Equal(t, "/api", cfg.Server.BasePath)
	assert.True(t, cfg.Server.WebAdminEnabled)
	assert.Equal(t, "production", cfg.Server.RunProfile)

	// Verify Upstream
	assert.Equal(t, "sk-test", cfg.Upstream.OpenAIKey)
	assert.Equal(t, "gk-test", cfg.Upstream.GeminiKey)
	assert.Equal(t, "https://code-assist.example.com", cfg.Upstream.CodeAssist)
	assert.Equal(t, "token-123", cfg.Upstream.GoogleToken)
	assert.Equal(t, "proj-456", cfg.Upstream.GoogleProjID)
	assert.Equal(t, "gemini", cfg.Upstream.UpstreamProvider)

	// Verify Security
	assert.Equal(t, "mgmt-key", cfg.Security.ManagementKey)
	assert.Equal(t, "hash-123", cfg.Security.ManagementKeyHash)
	assert.True(t, cfg.Security.ManagementReadOnly)
	assert.True(t, cfg.Security.ManagementAllowRemote)
	assert.Equal(t, 24, cfg.Security.ManagementRemoteTTlHours)
	assert.Equal(t, []string{"192.168.1.1", "10.0.0.1"}, cfg.Security.ManagementRemoteAllowIPs)
	assert.Equal(t, "/auth", cfg.Security.AuthDir)
	assert.True(t, cfg.Security.HeaderPassThrough)
	assert.True(t, cfg.Security.Debug)
	assert.Equal(t, "/var/log/app.log", cfg.Security.LogFile)

	// Verify Execution
	assert.Equal(t, 100, cfg.Execution.CallsPerRotation)
	assert.Equal(t, 5, cfg.Execution.MaxConcurrentPerCredential)
	assert.True(t, cfg.Execution.AutoLoadEnvCreds)

	// Verify Storage
	assert.Equal(t, "redis", cfg.Storage.Backend)
	assert.Equal(t, "/data", cfg.Storage.BaseDir)
	assert.Equal(t, "localhost:6379", cfg.Storage.RedisAddr)
	assert.Equal(t, "secret", cfg.Storage.RedisPassword)
	assert.Equal(t, 1, cfg.Storage.RedisDB)
	assert.Equal(t, "gcli:", cfg.Storage.RedisPrefix)

	// Verify Retry
	assert.True(t, cfg.Retry.Enabled)
	assert.Equal(t, 3, cfg.Retry.Max)
	assert.Equal(t, 1, cfg.Retry.IntervalSec)
	assert.Equal(t, 10, cfg.Retry.MaxIntervalSec)
	assert.True(t, cfg.Retry.On5xx)
	assert.True(t, cfg.Retry.OnNetworkError)

	// Verify RateLimit
	assert.True(t, cfg.RateLimit.Enabled)
	assert.Equal(t, 100, cfg.RateLimit.RPS)
	assert.Equal(t, 200, cfg.RateLimit.Burst)
	assert.Equal(t, 24, cfg.RateLimit.UsageResetIntervalHours)

	// Verify APICompat
	assert.True(t, cfg.APICompat.OpenAIImagesIncludeMIME)
	assert.Equal(t, 512, cfg.APICompat.ToolArgsDeltaChunk)
	assert.Equal(t, []string{"gemini-2.5-pro", "gemini-2.5-flash"}, cfg.APICompat.PreferredBaseModels)

	// Verify ResponseShaping
	assert.Equal(t, 3, cfg.ResponseShaping.AntiTruncationMax)
	assert.True(t, cfg.ResponseShaping.AntiTruncationEnabled)
	assert.True(t, cfg.ResponseShaping.FakeStreamingEnabled)

	// Verify OAuth
	assert.Equal(t, "client-id", cfg.OAuth.ClientID)
	assert.Equal(t, "client-secret", cfg.OAuth.ClientSecret)

	// Verify AutoBan
	assert.True(t, cfg.AutoBan.Enabled)
	assert.Equal(t, 10, cfg.AutoBan.Ban429Threshold)

	// Verify AutoProbe
	assert.True(t, cfg.AutoProbe.Enabled)
	assert.Equal(t, 2, cfg.AutoProbe.HourUTC)

	// Verify Routing
	assert.Equal(t, 3600, cfg.Routing.StickyTTLSeconds)
	assert.Equal(t, 1000, cfg.Routing.CooldownBaseMS)
}

// TestSyncFromDomains verifies that domain structures are correctly synced to top-level fields
func TestSyncFromDomains(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			OpenAIPort:      "9317",
			GeminiPort:      "9318",
			BasePath:        "/v2",
			WebAdminEnabled: false,
			RunProfile:      "dev",
		},
		Upstream: UpstreamConfig{
			OpenAIKey:        "sk-new",
			GeminiKey:        "gk-new",
			CodeAssist:       "https://new-endpoint.com",
			GoogleToken:      "new-token",
			GoogleProjID:     "new-proj",
			UpstreamProvider: "code_assist",
		},
		Security: SecurityConfig{
			ManagementKey:            "new-mgmt",
			ManagementKeyHash:        "new-hash",
			ManagementReadOnly:       false,
			ManagementAllowRemote:    false,
			ManagementRemoteTTlHours: 48,
			ManagementRemoteAllowIPs: []string{"172.16.0.1"},
			AuthDir:                  "/new-auth",
			HeaderPassThrough:        false,
			Debug:                    false,
			LogFile:                  "/new/log.log",
		},
		RateLimit: RateLimitConfig{
			Enabled:                 false,
			RPS:                     50,
			Burst:                   100,
			UsageResetIntervalHours: 12,
			UsageResetTimezone:      "America/New_York",
			UsageResetHourLocal:     6,
		},
	}

	cfg.SyncFromDomains()

	// Verify Server fields
	assert.Equal(t, "9317", cfg.OpenAIPort)
	assert.Equal(t, "9318", cfg.GeminiPort)
	assert.Equal(t, "/v2", cfg.BasePath)
	assert.False(t, cfg.WebAdminEnabled)
	assert.Equal(t, "dev", cfg.RunProfile)

	// Verify Upstream fields
	assert.Equal(t, "sk-new", cfg.OpenAIKey)
	assert.Equal(t, "gk-new", cfg.GeminiKey)
	assert.Equal(t, "https://new-endpoint.com", cfg.CodeAssist)
	assert.Equal(t, "new-token", cfg.GoogleToken)
	assert.Equal(t, "new-proj", cfg.GoogleProjID)
	assert.Equal(t, "code_assist", cfg.UpstreamProvider)

	// Verify Security fields
	assert.Equal(t, "new-mgmt", cfg.ManagementKey)
	assert.Equal(t, "new-hash", cfg.ManagementKeyHash)
	assert.False(t, cfg.ManagementReadOnly)
	assert.False(t, cfg.ManagementAllowRemote)
	assert.Equal(t, 48, cfg.ManagementRemoteTTlHours)
	assert.Equal(t, []string{"172.16.0.1"}, cfg.ManagementRemoteAllowIPs)

	// Verify RateLimit fields
	assert.False(t, cfg.RateLimitEnabled)
	assert.Equal(t, 50, cfg.RateLimitRPS)
	assert.Equal(t, 100, cfg.RateLimitBurst)
	assert.Equal(t, 12, cfg.UsageResetIntervalHours)
}

// TestBidirectionalSyncConsistency verifies that SyncToDomains followed by SyncFromDomains preserves values
func TestBidirectionalSyncConsistency(t *testing.T) {
	original := &Config{
		OpenAIPort:               "8080",
		GeminiPort:               "8081",
		BasePath:                 "/test",
		ManagementKey:            "test-key",
		RetryEnabled:             true,
		RetryMax:                 5,
		RateLimitRPS:             200,
		AntiTruncationEnabled:    true,
		FakeStreamingChunkSize:   256,
		AutoBanEnabled:           true,
		AutoProbeEnabled:         true,
		StickyTTLSeconds:         7200,
		UsageResetIntervalHours:  48,
		PreferredBaseModels:      []string{"model-a", "model-b"},
		ManagementRemoteAllowIPs: []string{"1.2.3.4", "5.6.7.8"},
	}

	// Sync to domains
	original.SyncToDomains()

	// Create a new config and sync from domains
	result := &Config{
		Server:          original.Server,
		Upstream:        original.Upstream,
		Security:        original.Security,
		Execution:       original.Execution,
		Storage:         original.Storage,
		Retry:           original.Retry,
		RateLimit:       original.RateLimit,
		APICompat:       original.APICompat,
		ResponseShaping: original.ResponseShaping,
		OAuth:           original.OAuth,
		AutoBan:         original.AutoBan,
		AutoProbe:       original.AutoProbe,
		Routing:         original.Routing,
	}
	result.SyncFromDomains()

	// Verify critical fields are preserved
	assert.Equal(t, original.OpenAIPort, result.OpenAIPort, "OpenAIPort should be preserved")
	assert.Equal(t, original.GeminiPort, result.GeminiPort, "GeminiPort should be preserved")
	assert.Equal(t, original.BasePath, result.BasePath, "BasePath should be preserved")
	assert.Equal(t, original.ManagementKey, result.ManagementKey, "ManagementKey should be preserved")
	assert.Equal(t, original.RetryEnabled, result.RetryEnabled, "RetryEnabled should be preserved")
	assert.Equal(t, original.RetryMax, result.RetryMax, "RetryMax should be preserved")
	assert.Equal(t, original.RateLimitRPS, result.RateLimitRPS, "RateLimitRPS should be preserved")
	assert.Equal(t, original.AntiTruncationEnabled, result.AntiTruncationEnabled, "AntiTruncationEnabled should be preserved")
	assert.Equal(t, original.FakeStreamingChunkSize, result.FakeStreamingChunkSize, "FakeStreamingChunkSize should be preserved")
	assert.Equal(t, original.AutoBanEnabled, result.AutoBanEnabled, "AutoBanEnabled should be preserved")
	assert.Equal(t, original.AutoProbeEnabled, result.AutoProbeEnabled, "AutoProbeEnabled should be preserved")
	assert.Equal(t, original.StickyTTLSeconds, result.StickyTTLSeconds, "StickyTTLSeconds should be preserved")
	assert.Equal(t, original.UsageResetIntervalHours, result.UsageResetIntervalHours, "UsageResetIntervalHours should be preserved")
	assert.Equal(t, original.PreferredBaseModels, result.PreferredBaseModels, "PreferredBaseModels should be preserved")
	assert.Equal(t, original.ManagementRemoteAllowIPs, result.ManagementRemoteAllowIPs, "ManagementRemoteAllowIPs should be preserved")
}

// TestSyncDomainsIdempotency verifies that multiple sync operations don't corrupt data
func TestSyncDomainsIdempotency(t *testing.T) {
	cfg := &Config{
		OpenAIPort:       "8000",
		RetryMax:         10,
		RateLimitBurst:   500,
		AutoBanEnabled:   true,
		StickyTTLSeconds: 1800,
	}

	// Perform multiple sync cycles
	for i := 0; i < 5; i++ {
		cfg.SyncToDomains()
		cfg.SyncFromDomains()
	}

	// Values should remain stable
	assert.Equal(t, "8000", cfg.OpenAIPort)
	assert.Equal(t, 10, cfg.RetryMax)
	assert.Equal(t, 500, cfg.RateLimitBurst)
	assert.True(t, cfg.AutoBanEnabled)
	assert.Equal(t, 1800, cfg.StickyTTLSeconds)

	// Domain structures should also be stable
	assert.Equal(t, "8000", cfg.Server.OpenAIPort)
	assert.Equal(t, 10, cfg.Retry.Max)
	assert.Equal(t, 500, cfg.RateLimit.Burst)
	assert.True(t, cfg.AutoBan.Enabled)
	assert.Equal(t, 1800, cfg.Routing.StickyTTLSeconds)
}

// TestSyncWithNilSlices verifies that nil slices are handled correctly
func TestSyncWithNilSlices(t *testing.T) {
	cfg := &Config{
		PreferredBaseModels:      nil,
		DisabledModels:           nil,
		SanitizerPatterns:        nil,
		ManagementRemoteAllowIPs: nil,
	}

	// Should not panic
	require.NotPanics(t, func() {
		cfg.SyncToDomains()
		cfg.SyncFromDomains()
	})

	// Nil slices should remain nil (or become empty slices, both are acceptable)
	// The important thing is no panic and consistent behavior
	assert.NotPanics(t, func() {
		_ = len(cfg.PreferredBaseModels)
		_ = len(cfg.APICompat.PreferredBaseModels)
	})
}

// TestSyncWithEmptyStrings verifies that empty strings are preserved
func TestSyncWithEmptyStrings(t *testing.T) {
	cfg := &Config{
		OpenAIPort:    "",
		BasePath:      "",
		ManagementKey: "",
		CodeAssist:    "",
	}

	cfg.SyncToDomains()

	assert.Equal(t, "", cfg.Server.OpenAIPort)
	assert.Equal(t, "", cfg.Server.BasePath)
	assert.Equal(t, "", cfg.Security.ManagementKey)
	assert.Equal(t, "", cfg.Upstream.CodeAssist)

	cfg.SyncFromDomains()

	assert.Equal(t, "", cfg.OpenAIPort)
	assert.Equal(t, "", cfg.BasePath)
	assert.Equal(t, "", cfg.ManagementKey)
	assert.Equal(t, "", cfg.CodeAssist)
}
