package gemini

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"gcli2api-go/internal/config"
)

func TestApplyDefaultHeaders_PassthroughRestricted(t *testing.T) {
	cfg := &config.Config{HeaderPassThrough: true}
	c := New(cfg) // base client, no credentials

	req, err := http.NewRequest(http.MethodPost, "http://example.test", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	// Attempt to override UA and X-Goog-Api-Client (should be ignored)
	// Allow X-Goog-User-Project (should be accepted)
	hdr := make(http.Header)
	hdr.Set("User-Agent", "malicious-UA/0.0.1")
	hdr.Set("X-Goog-Api-Client", "gl-node/99.99.99")
	hdr.Set("X-Goog-User-Project", "abc-project")

	ctx := WithHeaderOverrides(context.Background(), hdr)
	c.applyDefaultHeaders(ctx, req, "")

	// UA must be gemini-cli fingerprint
	wantUA := generateGeminiCLIUserAgent()
	if got := req.Header.Get("User-Agent"); got != wantUA {
		t.Fatalf("User-Agent not enforced, got=%q want=%q", got, wantUA)
	}
	// X-Goog-Api-Client must be gl-go/*
	if got := req.Header.Get("X-Goog-Api-Client"); !strings.HasPrefix(got, "gl-go/") {
		t.Fatalf("X-Goog-Api-Client not enforced, got=%q", got)
	}
	// Client-Metadata should be set to pluginType=GEMINI
	if got := req.Header.Get("Client-Metadata"); !strings.Contains(got, "pluginType=GEMINI") {
		t.Fatalf("Client-Metadata not set, got=%q", got)
	}
	// X-Goog-User-Project should pass through
	if got := req.Header.Get("X-Goog-User-Project"); got != "abc-project" {
		t.Fatalf("X-Goog-User-Project not passed through, got=%q", got)
	}
}

func TestApplyDefaultHeaders_ProjectFallbackFromConfig(t *testing.T) {
	cfg := &config.Config{HeaderPassThrough: true, GoogleProjID: "proj-xyz"}
	c := New(cfg)
	req, _ := http.NewRequest(http.MethodPost, "http://example.test", nil)
	c.applyDefaultHeaders(context.Background(), req, "")
	if got := req.Header.Get("X-Goog-User-Project"); got != "proj-xyz" {
		t.Fatalf("X-Goog-User-Project fallback not applied, got=%q", got)
	}
}

func TestApplyDefaultHeaders_RequestIDPassthrough(t *testing.T) {
	cfg := &config.Config{HeaderPassThrough: true}
	c := New(cfg)
	req, _ := http.NewRequest(http.MethodPost, "http://example.test", nil)
	hdr := make(http.Header)
	hdr.Set("X-Request-ID", "req-123")
	ctx := WithHeaderOverrides(context.Background(), hdr)
	c.applyDefaultHeaders(ctx, req, "")
	if got := req.Header.Get("X-Request-ID"); got != "req-123" {
		t.Fatalf("X-Request-ID passthrough failed, got=%q", got)
	}
	if got := req.Header.Get("X-Client-Request-ID"); got != "req-123" {
		t.Fatalf("X-Client-Request-ID not set, got=%q", got)
	}
}
