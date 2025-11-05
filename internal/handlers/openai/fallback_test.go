package openai

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"
)

func canBind() bool {
	if l, err := net.Listen("tcp", "127.0.0.1:0"); err == nil {
		_ = l.Close()
		return true
	}
	if l6, err := net.Listen("tcp6", "[::1]:0"); err == nil {
		_ = l6.Close()
		return true
	}
	return false
}

func TestTryStreamWithFallback(t *testing.T) {
	if !canBind() {
		t.Skip("sandbox does not allow binding ports for httptest")
	}
	var attempts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		model, _ := payload["model"].(string)
		attempts = append(attempts, model)
		if model == "gemini-2.5-pro" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"fail"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{
		CodeAssist:          srv.URL,
		RetryEnabled:        false,
		GoogleProjID:        "test-project",
		PreferredBaseModels: []string{"gemini-2.5-pro"},
	}
	handler := New(cfg, nil, nil, nil, nil)

	resp, usedModel, err := handler.tryStreamWithFallback(context.Background(), nil, "gemini-2.5-pro", cfg.GoogleProjID, map[string]any{})
	if err != nil {
		t.Fatalf("tryStreamWithFallback error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if usedModel == "gemini-2.5-pro" {
		t.Fatalf("expected fallback model, got %s", usedModel)
	}
	if len(attempts) < 2 {
		t.Fatalf("expected at least two attempts, got %v", attempts)
	}
}

func TestTryGenerateWithFallback(t *testing.T) {
	if !canBind() {
		t.Skip("sandbox does not allow binding ports for httptest")
	}
	var attempts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		model, _ := payload["model"].(string)
		attempts = append(attempts, model)
		if model == "gemini-2.5-pro" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"fail"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{
		CodeAssist:          srv.URL,
		RetryEnabled:        false,
		GoogleProjID:        "test-project",
		PreferredBaseModels: []string{"gemini-2.5-pro"},
	}
	handler := New(cfg, nil, nil, nil, nil)

	resp, usedModel, err := handler.tryGenerateWithFallback(context.Background(), nil, "gemini-2.5-pro", cfg.GoogleProjID, map[string]any{})
	if err != nil {
		t.Fatalf("tryGenerateWithFallback error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if usedModel == "gemini-2.5-pro" {
		t.Fatalf("expected fallback model, got %s", usedModel)
	}
	if len(attempts) < 2 {
		t.Fatalf("expected at least two attempts, got %v", attempts)
	}
}
