package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestStatusClass(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected string
	}{
		{"2xx success", 200, "2xx"},
		{"2xx created", 201, "2xx"},
		{"3xx redirect", 301, "3xx"},
		{"3xx not modified", 304, "3xx"},
		{"4xx bad request", 400, "4xx"},
		{"4xx not found", 404, "4xx"},
		{"5xx server error", 500, "5xx"},
		{"5xx gateway error", 502, "5xx"},
		{"1xx informational", 100, "1xx"},
		{"unknown", 600, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statusClass(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Metrics())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
	})

	t.Run("successful request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Metrics are recorded (no panic)
	})

	t.Run("error request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/error", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		// Metrics are recorded (no panic)
	})

	t.Run("POST request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", nil)

		router.ServeHTTP(w, req)

		// Metrics are recorded (no panic)
	})
}

func TestRecordSSEMetrics(t *testing.T) {
	t.Run("record SSE lines", func(t *testing.T) {
		RecordSSELines("server1", "/api/chat", 10)
		RecordSSELines("server2", "/api/completions", 5)
		// Should not panic
	})

	t.Run("record tool calls", func(t *testing.T) {
		RecordToolCalls("server1", "/api/chat", 3)
		RecordToolCalls("server2", "/api/chat", 2)
		// Should not panic
	})

	t.Run("record SSE close", func(t *testing.T) {
		RecordSSEClose("server1", "/api/chat", "client_disconnect")
		RecordSSEClose("server2", "/api/chat", "timeout")
		RecordSSEClose("server3", "/api/chat", "")
		// Should not panic
	})

	t.Run("record fallback", func(t *testing.T) {
		RecordFallback("server1", "/api/chat", "gemini-2.0-flash-exp", "gemini-1.5-pro")
		RecordFallback("server2", "/api/chat", "model-a", "model-b")
		// Should not panic
	})

	t.Run("record thinking removed", func(t *testing.T) {
		RecordThinkingRemoved("server1", "/api/chat", "gemini-2.0-flash-exp")
		RecordThinkingRemoved("server2", "/api/chat", "gemini-1.5-pro")
		// Should not panic
	})

	t.Run("record anti-truncation attempt", func(t *testing.T) {
		RecordAntiTruncAttempt("server1", "/api/chat", 3)
		RecordAntiTruncAttempt("server2", "/api/chat", 5)
		// Should not panic
	})

	t.Run("record management access", func(t *testing.T) {
		RecordManagementAccess("/api/credentials", "allow", "local")
		RecordManagementAccess("/api/config", "deny", "remote")
		// Should not panic
	})
}

func TestRecordUpstreamMetrics(t *testing.T) {
	t.Run("record upstream", func(t *testing.T) {
		RecordUpstream("gemini", 100*time.Millisecond, 200, false)
		RecordUpstream("gemini", 500*time.Millisecond, 500, true)
		// Should not panic
	})

	t.Run("record upstream with server", func(t *testing.T) {
		RecordUpstreamWithServer("gemini", "server1", 100*time.Millisecond, 200, false)
		RecordUpstreamWithServer("gemini", "server2", 500*time.Millisecond, 500, true)
		// Should not panic
	})

	t.Run("record upstream retry", func(t *testing.T) {
		RecordUpstreamRetry("gemini", 3, true)
		RecordUpstreamRetry("gemini", 5, false)
		// Should not panic
	})

	t.Run("record upstream error", func(t *testing.T) {
		RecordUpstreamError("gemini", "timeout")
		RecordUpstreamError("gemini", "connection_refused")
		// Should not panic
	})

	t.Run("record upstream model", func(t *testing.T) {
		RecordUpstreamModel("gemini", "gemini-2.0-flash-exp", 200, false)
		RecordUpstreamModel("gemini", "gemini-1.5-pro", 500, false)
		RecordUpstreamModel("gemini", "gemini-2.0-flash-exp", 0, true)
		// Should not panic
	})
}

func TestMetricsIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Metrics())

	router.POST("/api/chat", func(c *gin.Context) {
		// Simulate some processing
		time.Sleep(10 * time.Millisecond)

		// Record some metrics
		RecordSSELines("server1", "/api/chat", 5)
		RecordToolCalls("server1", "/api/chat", 2)
		RecordUpstream("gemini", 50*time.Millisecond, 200, false)

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	t.Run("full request with metrics", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/chat", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestMetricsEdgeCases(t *testing.T) {
	t.Run("record with empty server/path", func(t *testing.T) {
		RecordSSELines("", "", 10)
		RecordToolCalls("", "", 5)
		RecordUpstreamModel("gemini", "", 200, false)
		// Should not panic
	})

	t.Run("record with zero duration", func(t *testing.T) {
		RecordUpstream("gemini", 0, 200, false)
		RecordSSEClose("server1", "/api/chat", "timeout")
		// Should not panic
	})

	t.Run("record with negative values", func(t *testing.T) {
		RecordSSELines("server1", "/api/chat", -1)
		RecordToolCalls("server1", "/api/chat", -1)
		RecordAntiTruncAttempt("server1", "/api/chat", -1)
		// Should not panic (metrics library handles this)
	})

	t.Run("record with very large values", func(t *testing.T) {
		RecordSSELines("server1", "/api/chat", 1000000)
		RecordUpstream("gemini", 1*time.Hour, 200, false)
		// Should not panic
	})
}

func TestMetricsConcurrency(t *testing.T) {
	t.Run("concurrent metric recording", func(t *testing.T) {
		done := make(chan bool)

		// Simulate concurrent requests
		for i := 0; i < 10; i++ {
			go func(id int) {
				RecordSSELines("server1", "/api/chat", id)
				RecordToolCalls("server1", "/api/chat", id)
				RecordUpstream("gemini", time.Duration(id)*time.Millisecond, 200, false)
				RecordUpstreamRetry("gemini", id, true)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
