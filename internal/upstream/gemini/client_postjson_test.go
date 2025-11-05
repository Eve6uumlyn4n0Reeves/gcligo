package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"gcli2api-go/internal/config"
)

func TestClientPostJSONFallbackOn404(t *testing.T) {
	t.Parallel()

	var attempts []string
	cfg := &config.Config{
		CodeAssist:   "https://stub",
		RetryEnabled: false,
	}
	client := New(cfg)
	client.cli = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			req.Body.Close()
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			if m, ok := payload["model"].(string); ok {
				attempts = append(attempts, m)
			}
			if len(attempts) == 1 {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewBufferString(`{"error":"missing model"}`)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"response":{"ok":true}}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, err := client.Generate(context.Background(), []byte(`{"model":"gemini-2.5-pro","request":{}}`))
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, data)
	}
	data, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(data), `"ok":true`) {
		t.Fatalf("unexpected body: %s", data)
	}
	if len(attempts) < 2 {
		t.Fatalf("expected fallback attempts, got %v", attempts)
	}
	if attempts[0] == attempts[1] {
		t.Fatalf("expected different model on fallback, got %v", attempts)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientDoAttemptSetsAcceptHeaderForSSE(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	client := New(cfg)
	var seen string
	client.cli = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = req.Header.Get("Accept")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, err, _, status, tries := client.doAttempt(context.Background(), "https://example.com/v1?alt=sse", []byte("{}"), "")
	if err != nil {
		t.Fatalf("doAttempt returned err: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response")
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if tries != 0 {
		t.Fatalf("expected 0 retries, got %d", tries)
	}
	if seen != "text/event-stream" {
		t.Fatalf("expected Accept text/event-stream, got %q", seen)
	}
}

func TestClientDoAttemptSetsAcceptHeaderForJSON(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	client := New(cfg)
	var seen string
	client.cli = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = req.Header.Get("Accept")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	if _, err, _, _, _ := client.doAttempt(context.Background(), "https://example.com/v1", []byte("{}"), ""); err != nil {
		t.Fatalf("doAttempt returned err: %v", err)
	}
	if seen != "application/json" {
		t.Fatalf("expected Accept application/json, got %q", seen)
	}
}

func TestClientDoAttemptRetriesOn5xx(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		RetryEnabled:        true,
		RetryMax:            2,
		RetryIntervalSec:    0,
		RetryMaxIntervalSec: 0,
		RetryOn5xx:          true,
		RetryOnNetworkError: false,
	}
	client := New(cfg)
	var calls int32
	client.cli = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if atomic.AddInt32(&calls, 1) == 1 {
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Header:     http.Header{"Retry-After": []string{"0"}},
					Body:       io.NopCloser(bytes.NewBufferString(`{"error":"retry"}`)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, err, _, status, tries := client.doAttempt(context.Background(), "https://example.com/v1", []byte("{}"), "")
	if err != nil {
		t.Fatalf("doAttempt returned err: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response")
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if tries == 0 {
		t.Fatalf("expected at least one retry, got %d", tries)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected multiple calls, got %d", calls)
	}
}

func TestClientDoAttemptStopsOnContextCancel(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	client := New(cfg)
	client.cli = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("should not be called when context cancelled")
		}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resp, err, _, status, tries := client.doAttempt(ctx, "https://example.com/v1", []byte("{}"), "")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response")
	}
	if status != 0 {
		t.Fatalf("expected status 0, got %d", status)
	}
	if tries != 0 {
		t.Fatalf("expected 0 retries, got %d", tries)
	}
}
