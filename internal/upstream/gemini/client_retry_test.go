package gemini

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"
)

func TestParseRetryAfter(t *testing.T) {
	if d, ok := parseRetryAfter("15"); !ok || d != 15*time.Second {
		t.Fatalf("expected 15s, got %v ok=%v", d, ok)
	}
	now := time.Now().Add(30 * time.Second).Format(time.RFC1123)
	if d, ok := parseRetryAfter(now); !ok || d < 29*time.Second || d > 31*time.Second {
		t.Fatalf("unexpected duration for date header: %v ok=%v", d, ok)
	}
	if _, ok := parseRetryAfter(""); ok {
		t.Fatalf("expected empty string to fail")
	}
}

func TestClassifyErr(t *testing.T) {
	timeoutErr := &url.Error{Err: context.DeadlineExceeded, Op: "Post", URL: "http://example.com"}
	if got := classifyErr(timeoutErr); got != "timeout" {
		t.Fatalf("expected timeout, got %s", got)
	}
	if got := classifyErr(context.DeadlineExceeded); got != "deadline" {
		t.Fatalf("expected deadline for bare error, got %s", got)
	}
	hostErr := &url.Error{Err: errors.New("lookup fail: no such host")}
	if got := classifyErr(hostErr); got != "dns" {
		t.Fatalf("expected dns, got %s", got)
	}
	if got := classifyErr(errors.New("connection reset by peer")); got != "conn_reset" {
		t.Fatalf("expected conn_reset, got %s", got)
	}
	if got := classifyErr(nil); got != "" {
		t.Fatalf("expected empty classification, got %s", got)
	}
}
