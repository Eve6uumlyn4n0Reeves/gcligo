package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Allow requests within limit", func(t *testing.T) {
		router := gin.New()
		router.Use(RateLimiter(10, 10))
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		// Should allow first request
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Block requests exceeding limit", func(t *testing.T) {
		router := gin.New()
		router.Use(RateLimiter(1, 1)) // Very low limit
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		// First request should succeed
		req1 := httptest.NewRequest("GET", "/test", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		if w1.Code != 200 {
			t.Errorf("First request: expected status 200, got %d", w1.Code)
		}

		// Second request should be rate limited
		req2 := httptest.NewRequest("GET", "/test", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("Second request: expected status 429, got %d", w2.Code)
		}
	})
}

func TestRateLimiterAutoKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Use API key for rate limiting", func(t *testing.T) {
		router := gin.New()
		router.Use(RateLimiterAutoKey(10, 10))
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer test-key-123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Fallback to IP when no API key", func(t *testing.T) {
		router := gin.New()
		router.Use(RateLimiterAutoKey(10, 10))
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Global rate limit", func(t *testing.T) {
		router := gin.New()
		router.Use(RateLimiterAutoKey(1, 1)) // Very low limit
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		// Make many requests quickly
		successCount := 0
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer key-"+string(rune(i)))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == 200 {
				successCount++
			}
		}

		// Should have some rate limited requests
		if successCount >= 10 {
			t.Error("Expected some requests to be rate limited")
		}
	})
}

func TestExtractAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		setup    func(*gin.Context)
		expected string
	}{
		{
			name: "From context",
			setup: func(c *gin.Context) {
				c.Set("api_key", "context-key")
			},
			expected: "context-key",
		},
		{
			name: "From Authorization header",
			setup: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "Bearer header-key")
			},
			expected: "header-key",
		},
		{
			name: "From x-api-key header",
			setup: func(c *gin.Context) {
				c.Request.Header.Set("x-api-key", "x-api-key-value")
			},
			expected: "x-api-key-value",
		},
		{
			name: "From x-goog-api-key header",
			setup: func(c *gin.Context) {
				c.Request.Header.Set("x-goog-api-key", "goog-key")
			},
			expected: "goog-key",
		},
		{
			name:     "No API key",
			setup:    func(c *gin.Context) {},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			tt.setup(c)

			result := extractAPIKey(c)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTTLLimiterCache(t *testing.T) {
	t.Run("Get or create limiter", func(t *testing.T) {
		cache := newTTLLimiterCache(1 * time.Minute)

		lim1 := cache.get("key1", func() *rate.Limiter {
			return rate.NewLimiter(10, 10)
		})

		if lim1 == nil {
			t.Fatal("Expected limiter, got nil")
		}

		// Getting same key should return same limiter
		lim2 := cache.get("key1", func() *rate.Limiter {
			return rate.NewLimiter(20, 20)
		})

		if lim1 != lim2 {
			t.Error("Expected same limiter instance")
		}
	})

	t.Run("Sweep expired entries", func(t *testing.T) {
		cache := newTTLLimiterCache(100 * time.Millisecond)

		// Add entry
		cache.get("key1", func() *rate.Limiter {
			return rate.NewLimiter(10, 10)
		})

		if len(cache.items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(cache.items))
		}

		// Wait for expiry
		time.Sleep(150 * time.Millisecond)

		// Trigger sweep by adding new entry
		cache.lastSweep = time.Time{} // Force sweep
		cache.get("key2", func() *rate.Limiter {
			return rate.NewLimiter(10, 10)
		})

		// key1 should be swept
		cache.mu.RLock()
		_, exists := cache.items["key1"]
		cache.mu.RUnlock()

		if exists {
			t.Error("Expected key1 to be swept")
		}
	})
}

func TestRateLimiterAutoKeyDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Use defaults for invalid values", func(t *testing.T) {
		router := gin.New()
		router.Use(RateLimiterAutoKey(0, 0)) // Invalid values
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}
