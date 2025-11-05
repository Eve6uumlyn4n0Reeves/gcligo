package streaming

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"gcli2api-go/internal/antitrunc"
	"gcli2api-go/internal/common"

	log "github.com/sirupsen/logrus"
)

// AntiTruncationConfig holds configuration for anti-truncation
type AntiTruncationConfig struct {
	MaxAttempts      int           // Maximum retry attempts
	MinCompletionLen int           // Minimum expected completion length
	ContinuePrompt   string        // Prompt to continue generation
	RetryDelay       time.Duration // Delay between retries
}

// DefaultAntiTruncationConfig returns default configuration
func DefaultAntiTruncationConfig() AntiTruncationConfig {
	return AntiTruncationConfig{
		MaxAttempts:      3,
		MinCompletionLen: 50,
		ContinuePrompt:   common.ContinuationPrompt,
		RetryDelay:       1 * time.Second,
	}
}

// RequestFunc is a function that sends a request and returns a streaming response
type RequestFunc func(ctx context.Context, requestBody []byte) (io.Reader, error)

// WithAntiTruncation wraps a streaming request with anti-truncation logic
func WithAntiTruncation(ctx context.Context, req RequestFunc, initialRequest []byte, cfg AntiTruncationConfig) (io.Reader, error) {
	var allContent strings.Builder
	attempts := 0
	detectorCfg := antitrunc.DefaultConfig()
	detectorCfg.MinCompletionLen = cfg.MinCompletionLen

	for attempts < cfg.MaxAttempts {
		attempts++

		// Send request
		stream, err := req(ctx, initialRequest)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		// Extract text from stream
		text, err := ExtractTextFromStream(stream)
		if err != nil {
			return nil, fmt.Errorf("failed to extract text: %w", err)
		}

		allContent.WriteString(text)

		// Check if response looks complete
		if detectorCfg.ResponseComplete(text) {
			log.Infof("Anti-truncation: Response complete after %d attempts", attempts)
			break
		}

		// Check if we should retry
		if attempts >= cfg.MaxAttempts {
			log.Warnf("Anti-truncation: Max attempts reached (%d)", cfg.MaxAttempts)
			break
		}

		log.Infof("Anti-truncation: Response truncated, retrying (attempt %d/%d)", attempts+1, cfg.MaxAttempts)

		// Wait before retry
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(cfg.RetryDelay):
		}

		// Modify request to continue from where we left off
		initialRequest = antitrunc.BuildContinuationPayload(initialRequest, allContent.String(), cfg.ContinuePrompt)
	}

	// Convert accumulated text back to a streaming response
	finalText := allContent.String()
	return strings.NewReader(finalText), nil
}

// CombineStreamReaders combines multiple streaming readers into one
func CombineStreamReaders(readers ...io.Reader) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		for i, reader := range readers {
			if i > 0 {
				// Add a separator between streams
				pw.Write([]byte("\n"))
			}

			_, err := io.Copy(pw, reader)
			if err != nil {
				log.Errorf("Error copying stream %d: %v", i, err)
				return
			}
		}
	}()

	return pr
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []string
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []string{
			"timeout",
			"connection refused",
			"429",
			"503",
		},
	}
}

// WithRetry wraps a request with exponential backoff retry logic
func WithRetry(ctx context.Context, req RequestFunc, requestBody []byte, cfg RetryConfig) (io.Reader, error) {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Infof("Retry attempt %d/%d after %v", attempt, cfg.MaxRetries, delay)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			// Exponential backoff
			delay = time.Duration(float64(delay) * cfg.BackoffFactor)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}

		result, err := req(ctx, requestBody)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err, cfg.RetryableErrors) {
			return nil, err
		}

		log.Warnf("Request failed with retryable error: %v", err)
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetryableError(err error, retryableErrors []string) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, strings.ToLower(retryable)) {
			return true
		}
	}

	return false
}
