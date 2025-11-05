package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Generate request ID when not provided", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestID())
		router.GET("/test", func(c *gin.Context) {
			rid, exists := c.Get("request_id")
			if !exists {
				t.Error("Expected request_id to be set in context")
			}
			if rid == "" {
				t.Error("Expected non-empty request ID")
			}
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response header
		responseID := w.Header().Get("X-Request-ID")
		if responseID == "" {
			t.Error("Expected X-Request-ID header in response")
		}

		if len(responseID) != 32 { // hex encoded 16 bytes = 32 chars
			t.Errorf("Expected request ID length 32, got %d", len(responseID))
		}
	})

	t.Run("Use provided request ID", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestID())
		router.GET("/test", func(c *gin.Context) {
			rid, _ := c.Get("request_id")
			if rid != "custom-request-id" {
				t.Errorf("Expected 'custom-request-id', got %v", rid)
			}
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "custom-request-id")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response header
		responseID := w.Header().Get("X-Request-ID")
		if responseID != "custom-request-id" {
			t.Errorf("Expected 'custom-request-id', got %q", responseID)
		}
	})

	t.Run("Generate unique IDs for different requests", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestID())
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req1 := httptest.NewRequest("GET", "/test", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		req2 := httptest.NewRequest("GET", "/test", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		id1 := w1.Header().Get("X-Request-ID")
		id2 := w2.Header().Get("X-Request-ID")

		if id1 == id2 {
			t.Error("Expected different request IDs for different requests")
		}
	})
}
