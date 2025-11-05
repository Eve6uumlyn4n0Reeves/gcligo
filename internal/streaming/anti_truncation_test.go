package streaming

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"gcli2api-go/internal/antitrunc"
)

func TestCleanContinuationText(t *testing.T) {
	cases := map[string]string{
		"uppercase":    "hello\n[DONE]\nworld",
		"lowercase":    "hello\n[done]\nworld",
		"mixed-spaces": "hello\n  [DoNe]  \nworld",
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			out := antitrunc.CleanContinuationText(input)
			if out == input || out == "" {
				t.Fatalf("clean continuation failed: %q", out)
			}
			if out != "hello\nworld" {
				t.Fatalf("unexpected cleaned: %q", out)
			}
		})
	}
}

func TestDefaultAntiTruncationConfig(t *testing.T) {
	cfg := DefaultAntiTruncationConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts=3, got %d", cfg.MaxAttempts)
	}
	if cfg.MinCompletionLen != 50 {
		t.Errorf("Expected MinCompletionLen=50, got %d", cfg.MinCompletionLen)
	}
	if cfg.RetryDelay != 1*time.Second {
		t.Errorf("Expected RetryDelay=1s, got %v", cfg.RetryDelay)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", cfg.MaxRetries)
	}
	if cfg.InitialDelay != 1*time.Second {
		t.Errorf("Expected InitialDelay=1s, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 10*time.Second {
		t.Errorf("Expected MaxDelay=10s, got %v", cfg.MaxDelay)
	}
	if cfg.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor=2.0, got %f", cfg.BackoffFactor)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		retryableErrors []string
		expected        bool
	}{
		{"Nil error", nil, []string{"timeout"}, false},
		{"Timeout error", errors.New("connection timeout"), []string{"timeout"}, true},
		{"429 error", errors.New("HTTP 429 Too Many Requests"), []string{"429"}, true},
		{"503 error", errors.New("Service Unavailable 503"), []string{"503"}, true},
		{"Non-retryable error", errors.New("invalid request"), []string{"timeout", "429"}, false},
		{"Case insensitive", errors.New("CONNECTION TIMEOUT"), []string{"timeout"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err, tt.retryableErrors)
			if result != tt.expected {
				t.Errorf("isRetryableError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWithRetry(t *testing.T) {
	t.Run("Success on first attempt", func(t *testing.T) {
		callCount := 0
		reqFunc := func(ctx context.Context, body []byte) (io.Reader, error) {
			callCount++
			return strings.NewReader("success"), nil
		}

		ctx := context.Background()
		cfg := RetryConfig{
			MaxRetries:      3,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"timeout"},
		}

		result, err := WithRetry(ctx, reqFunc, []byte("test"), cfg)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("Expected result, got nil")
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call, got %d", callCount)
		}
	})

	t.Run("Success after retries", func(t *testing.T) {
		callCount := 0
		reqFunc := func(ctx context.Context, body []byte) (io.Reader, error) {
			callCount++
			if callCount < 3 {
				return nil, errors.New("timeout error")
			}
			return strings.NewReader("success"), nil
		}

		ctx := context.Background()
		cfg := RetryConfig{
			MaxRetries:      3,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"timeout"},
		}

		result, err := WithRetry(ctx, reqFunc, []byte("test"), cfg)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("Expected result, got nil")
		}
		if callCount != 3 {
			t.Errorf("Expected 3 calls, got %d", callCount)
		}
	})

	t.Run("Max retries exceeded", func(t *testing.T) {
		callCount := 0
		reqFunc := func(ctx context.Context, body []byte) (io.Reader, error) {
			callCount++
			return nil, errors.New("timeout error")
		}

		ctx := context.Background()
		cfg := RetryConfig{
			MaxRetries:      2,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"timeout"},
		}

		result, err := WithRetry(ctx, reqFunc, []byte("test"), cfg)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if result != nil {
			t.Errorf("Expected nil result, got %v", result)
		}
		if callCount != 3 { // Initial + 2 retries
			t.Errorf("Expected 3 calls, got %d", callCount)
		}
	})

	t.Run("Non-retryable error", func(t *testing.T) {
		callCount := 0
		reqFunc := func(ctx context.Context, body []byte) (io.Reader, error) {
			callCount++
			return nil, errors.New("invalid request")
		}

		ctx := context.Background()
		cfg := RetryConfig{
			MaxRetries:      3,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"timeout"},
		}

		result, err := WithRetry(ctx, reqFunc, []byte("test"), cfg)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if result != nil {
			t.Errorf("Expected nil result, got %v", result)
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call (no retries), got %d", callCount)
		}
	})
}

func TestCombineStreamReaders(t *testing.T) {
	t.Run("Single reader", func(t *testing.T) {
		reader1 := strings.NewReader("Hello")
		combined := CombineStreamReaders(reader1)

		output, err := io.ReadAll(combined)
		if err != nil {
			t.Fatalf("Failed to read: %v", err)
		}

		if string(output) != "Hello" {
			t.Errorf("Expected 'Hello', got %q", string(output))
		}
	})

	t.Run("Multiple readers", func(t *testing.T) {
		reader1 := strings.NewReader("Hello")
		reader2 := strings.NewReader("World")
		reader3 := strings.NewReader("!")

		combined := CombineStreamReaders(reader1, reader2, reader3)

		output, err := io.ReadAll(combined)
		if err != nil {
			t.Fatalf("Failed to read: %v", err)
		}

		expected := "Hello\nWorld\n!"
		if string(output) != expected {
			t.Errorf("Expected %q, got %q", expected, string(output))
		}
	})

	t.Run("Empty readers", func(t *testing.T) {
		combined := CombineStreamReaders()

		output, err := io.ReadAll(combined)
		if err != nil {
			t.Fatalf("Failed to read: %v", err)
		}

		if len(output) != 0 {
			t.Errorf("Expected empty output, got %q", string(output))
		}
	})
}

func TestWithAntiTruncation(t *testing.T) {
	t.Run("Complete response on first attempt", func(t *testing.T) {
		callCount := 0
		reqFunc := func(ctx context.Context, body []byte) (io.Reader, error) {
			callCount++
			// Return a complete response
			stream := `data: {"choices":[{"delta":{"content":"This is a complete response with sufficient length to pass the check."}}]}

data: [DONE]

`
			return strings.NewReader(stream), nil
		}

		ctx := context.Background()
		cfg := DefaultAntiTruncationConfig()
		cfg.MinCompletionLen = 20

		result, err := WithAntiTruncation(ctx, reqFunc, []byte("test"), cfg)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("Expected result, got nil")
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call, got %d", callCount)
		}
	})

	t.Run("Request error", func(t *testing.T) {
		reqFunc := func(ctx context.Context, body []byte) (io.Reader, error) {
			return nil, errors.New("request failed")
		}

		ctx := context.Background()
		cfg := DefaultAntiTruncationConfig()

		result, err := WithAntiTruncation(ctx, reqFunc, []byte("test"), cfg)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if result != nil {
			t.Errorf("Expected nil result, got %v", result)
		}
	})
}
