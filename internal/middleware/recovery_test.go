package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Recover from panic", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())
		router.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 500 {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("Normal request without panic", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())
		router.GET("/normal", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/normal", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestRecoveryWithWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Custom recovery writer", func(t *testing.T) {
		called := false
		customWriter := func(c *gin.Context, err any) {
			called = true
		}

		router := gin.New()
		router.Use(RecoveryWithWriter(customWriter))
		router.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if !called {
			t.Error("Expected custom writer to be called")
		}

		if w.Code != 500 {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

func TestSafeGo(t *testing.T) {
	t.Run("Recover from goroutine panic", func(t *testing.T) {
		done := make(chan bool)

		SafeGo(func() {
			defer func() {
				done <- true
			}()
			panic("goroutine panic")
		})

		<-done
		// If we reach here, panic was recovered
	})

	t.Run("Normal goroutine execution", func(t *testing.T) {
		done := make(chan bool)

		SafeGo(func() {
			done <- true
		})

		<-done
	})
}

func TestSafeGoWithContext(t *testing.T) {
	t.Run("Recover from named goroutine panic", func(t *testing.T) {
		done := make(chan bool)

		SafeGoWithContext("test-goroutine", func() {
			defer func() {
				done <- true
			}()
			panic("named goroutine panic")
		})

		<-done
	})

	t.Run("Normal named goroutine execution", func(t *testing.T) {
		done := make(chan bool)

		SafeGoWithContext("test-goroutine", func() {
			done <- true
		})

		<-done
	})
}

func TestRecoverToError(t *testing.T) {
	t.Run("Convert panic to error", func(t *testing.T) {
		// RecoverToError is meant to be called inside a defer after recover()
		// It's not a standalone panic recovery mechanism
		// This test verifies it returns nil when no panic is active
		err := RecoverToError()
		if err != nil {
			t.Errorf("Expected nil when no panic, got %v", err)
		}
	})
}

func TestSafeCall(t *testing.T) {
	t.Run("Catch panic in function", func(t *testing.T) {
		err := SafeCall(func() error {
			panic("test panic")
		})

		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})

	t.Run("Return function error", func(t *testing.T) {
		expectedErr := errors.New("function error")
		err := SafeCall(func() error {
			return expectedErr
		})

		if err != expectedErr {
			t.Errorf("Expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("Successful function call", func(t *testing.T) {
		err := SafeCall(func() error {
			return nil
		})

		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})
}

func TestSafeCallWithValue(t *testing.T) {
	t.Run("Catch panic and return zero value", func(t *testing.T) {
		result, err := SafeCallWithValue(func() (int, error) {
			panic("test panic")
		})

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if result != 0 {
			t.Errorf("Expected zero value 0, got %d", result)
		}
	})

	t.Run("Return function result", func(t *testing.T) {
		result, err := SafeCallWithValue(func() (string, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}

		if result != "success" {
			t.Errorf("Expected 'success', got %q", result)
		}
	})

	t.Run("Return function error", func(t *testing.T) {
		expectedErr := errors.New("function error")
		result, err := SafeCallWithValue(func() (int, error) {
			return 0, expectedErr
		})

		if err != expectedErr {
			t.Errorf("Expected %v, got %v", expectedErr, err)
		}

		if result != 0 {
			t.Errorf("Expected 0, got %d", result)
		}
	})
}
