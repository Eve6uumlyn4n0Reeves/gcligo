package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Log successful request", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestLogger())
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

	t.Run("Log request with API key", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.Set("api_key", "test-key")
			c.Set("model", "test-model")
			c.Set("base_model", "base-model")
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Log failed request", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.String(500, "Error")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 500 {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("Log request with user agent", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestLogger())
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Test-Agent/1.0")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}
