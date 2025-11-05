package server

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRespondError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Error without details", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		respondError(c, 400, "test error", nil)

		if w.Code != 400 {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "test error") {
			t.Errorf("Expected error message in body, got %s", body)
		}
	})

	t.Run("Error with details", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		respondError(c, 500, "server error", "detailed info")

		if w.Code != 500 {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "server error") {
			t.Errorf("Expected error message in body, got %s", body)
		}
		if !strings.Contains(body, "detailed info") {
			t.Errorf("Expected details in body, got %s", body)
		}
	})
}

func TestRespondValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Nil error does nothing", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		respondValidationError(c, nil)

		if w.Code != 200 {
			t.Errorf("Expected status 200 (default), got %d", w.Code)
		}
	})

	t.Run("Non-nil error responds", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		respondValidationError(c, &testError{"validation failed"})

		if w.Code != 400 {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestBindJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Valid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"test"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		var dest map[string]string
		result := bindJSON(c, &dest)

		if !result {
			t.Error("Expected bindJSON to return true for valid JSON")
		}

		if dest["name"] != "test" {
			t.Errorf("Expected name=test, got %s", dest["name"])
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{invalid}`))
		c.Request.Header.Set("Content-Type", "application/json")

		var dest map[string]string
		result := bindJSON(c, &dest)

		if result {
			t.Error("Expected bindJSON to return false for invalid JSON")
		}

		if w.Code != 400 {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestSetNoCacheHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	setNoCacheHeaders(c)

	headers := w.Header()

	if headers.Get("Cache-Control") != "no-store, no-cache, must-revalidate" {
		t.Errorf("Unexpected Cache-Control header: %s", headers.Get("Cache-Control"))
	}

	if headers.Get("Pragma") != "no-cache" {
		t.Errorf("Unexpected Pragma header: %s", headers.Get("Pragma"))
	}

	if headers.Get("Expires") != "0" {
		t.Errorf("Unexpected Expires header: %s", headers.Get("Expires"))
	}
}

func TestRegisterPprof(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	registerPprof(router)

	// Test that pprof routes are registered
	routes := router.Routes()

	pprofRoutes := []string{
		"/debug/pprof/",
		"/debug/pprof/cmdline",
		"/debug/pprof/profile",
		"/debug/pprof/symbol",
		"/debug/pprof/trace",
		"/debug/pprof/allocs",
		"/debug/pprof/block",
		"/debug/pprof/goroutine",
		"/debug/pprof/heap",
		"/debug/pprof/mutex",
		"/debug/pprof/threadcreate",
	}

	routeMap := make(map[string]bool)
	for _, route := range routes {
		routeMap[route.Path] = true
	}

	for _, expectedRoute := range pprofRoutes {
		if !routeMap[expectedRoute] {
			t.Errorf("Expected pprof route %s to be registered", expectedRoute)
		}
	}
}
