package gemini

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/constants"
	mw "gcli2api-go/internal/middleware"
	"gcli2api-go/internal/models"
	"gcli2api-go/internal/monitoring/tracing"
	"gcli2api-go/internal/oauth"
	"gcli2api-go/internal/upstream"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	cfg         *config.Config
	cli         *http.Client
	caller      string             // optional: which server is using this client ("openai"/"gemini")
	credentials *oauth.Credentials // credential for this client
	token       string             // cached access token
}

func WithHeaderOverrides(ctx context.Context, hdr http.Header) context.Context {
	return upstream.WithHeaderOverrides(ctx, hdr)
}

func getHeaderOverrides(ctx context.Context) http.Header {
	return upstream.HeaderOverrides(ctx)
}

func durationOrDefault(seconds int, fallback time.Duration) time.Duration {
	if seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func New(cfg *config.Config) *Client {
	// Timeouts and proxy from environment/config
	dialTO := durationOrDefault(cfg.DialTimeoutSec, constants.DefaultDialTimeout)
	tlsTO := durationOrDefault(cfg.TLSHandshakeTimeoutSec, constants.DefaultTLSHandshakeTimeout)
	hdrTO := durationOrDefault(cfg.ResponseHeaderTimeoutSec, constants.DefaultResponseHeaderTimeout)
	expTO := durationOrDefault(cfg.ExpectContinueTimeoutSec, constants.DefaultExpectContinueTimeout)

	tr := &http.Transport{
		Proxy: getProxyFunc(cfg.ProxyURL),
		DialContext: (&net.Dialer{
			Timeout:   dialTO,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   tlsTO,
		ResponseHeaderTimeout: hdrTO,
		ExpectContinueTimeout: expTO,
		MaxIdleConns:          constants.BaseMaxIdleConns,
		MaxIdleConnsPerHost:   constants.BaseMaxIdleConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
	}
	return &Client{cfg: cfg, cli: &http.Client{Transport: tr, Timeout: 0}}
}

// getProxyFunc returns appropriate proxy function based on configuration
func getProxyFunc(proxyURL string) func(*http.Request) (*url.URL, error) {
	if proxyURL != "" {
		// Parse proxy URL
		if parsedURL, err := url.Parse(proxyURL); err == nil {
			return http.ProxyURL(parsedURL)
		}
	}
	// Fall back to environment proxy
	return http.ProxyFromEnvironment
}

// NewWithCredential creates a client with a specific credential
func NewWithCredential(cfg *config.Config, creds *oauth.Credentials) *Client {
	client := New(cfg)
	client.credentials = creds
	if creds != nil && creds.AccessToken != "" {
		client.token = creds.AccessToken
	}
	return client
}

// WithCaller sets which server layer is using this client (e.g., "openai" or "gemini").
func (c *Client) WithCaller(server string) *Client { c.caller = server; return c }

// getToken returns the access token (from credential or config fallback)
func (c *Client) getToken() string {
	if c.token != "" {
		return c.token
	}
	if c.credentials != nil && c.credentials.AccessToken != "" {
		return c.credentials.AccessToken
	}
	return c.cfg.GoogleToken
}

// generateGeminiCLIUserAgent creates a User-Agent string that mimics Gemini CLI client
// moved to client_headers.go
// func generateGeminiCLIUserAgent() string { return "" }

// postJSON sends a POST request with JSON body to the specified URL.
// It implements automatic model fallback on 404 errors.
//
// IMPORTANT: Caller is responsible for closing resp.Body if resp is non-nil and err is nil.
// On error, the response body (if any) is already closed by this function.
func (c *Client) postJSON(ctx context.Context, url string, body []byte, bearer string) (*http.Response, error) {
	// Determine requested base model and construct fallback order
	origModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if origModel == "" {
		origModel = "gemini-2.5-pro"
	}
	// Use full-feature fallback order (preserves suffixes like -maxthinking/-search)
	candidates := models.FallbackOrder(origModel)
	if len(candidates) == 0 {
		candidates = []string{origModel}
	}

	spanCtx, span := tracing.StartSpan(ctx, "upstream/gemini", "Gemini.PostJSON",
		trace.WithAttributes(
			attribute.String("http.method", http.MethodPost),
			attribute.String("http.url", url),
			attribute.String("upstream.caller", c.caller),
		))
	defer span.End()
	span.SetAttributes(attribute.String("upstream.original_model", origModel))
	ctx = spanCtx

	totalRetries := 0
	finishSpan := func(status int, err error) {
		span.SetAttributes(
			attribute.Int("http.status_code", status),
			attribute.Int("upstream.retry_total", totalRetries),
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else if status >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("http_status=%d", status))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}

	// Iterate candidate models with thinkingConfig safety and preview fallback on 404
	for i, m := range candidates {
		trial, _ := sjson.SetBytes(body, "model", m)
		// Apply lightweight image hints for flash-image variants
		trial = fixGeminiCLIImageHints(m, trial)
		// Remove thinking config for models that disallow it
		if geminiModelDisallowsThinking(m) {
			trial = deleteJSONField(trial, "request.generationConfig.thinkingConfig")
			trial = deleteJSONField(trial, "generationConfig.thinkingConfig")
			mw.RecordThinkingRemoved(c.caller, "code_assist", m)
		}
		resp, err, _, status, retries := c.doAttempt(ctx, url, trial, bearer)
		totalRetries += retries
		// record per-model upstream counter
		mw.RecordUpstreamModel("gemini", m, status, err != nil)
		span.AddEvent("attempt", trace.WithAttributes(
			attribute.String("upstream.model", m),
			attribute.Int("http.status_code", status),
			attribute.Int("retry.count", retries),
		))
		if status == 404 && i < len(candidates)-1 {
			if resp != nil {
				_ = resp.Body.Close()
			}
			next := candidates[i+1]
			mw.RecordFallback(c.caller, "code_assist", m, next)
			continue
		}
		finishSpan(status, err)
		return resp, err
	}
	// As a final fallback, run original body
	resp, err, _, status, retries := c.doAttempt(ctx, url, body, bearer)
	totalRetries += retries
	mw.RecordUpstreamModel("gemini", origModel, status, err != nil)
	span.AddEvent("attempt", trace.WithAttributes(
		attribute.String("upstream.model", origModel),
		attribute.Int("http.status_code", status),
		attribute.Int("retry.count", retries),
	))
	finishSpan(status, err)
	return resp, err
}

func getStatus(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

func (c *Client) shouldRetry(resp *http.Response, err error, attempt int) (bool, time.Duration) {
	// Do not retry on context cancellation/deadline
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return false, 0
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return false, 0
		}
		if c.cfg.RetryOnNetworkError {
			return true, c.nextBackoff(attempt)
		}
		return false, 0
	}
	if resp == nil {
		return false, 0
	}
	code := resp.StatusCode
	if code == 429 {
		if d, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
			return true, d
		}
		return true, c.nextBackoff(attempt)
	}
	if c.cfg.RetryOn5xx && code >= 500 && code <= 599 {
		if code == 503 {
			if d, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
				return true, d
			}
		}
		return true, c.nextBackoff(attempt)
	}
	if code == 408 || code == 425 { // request timeout/too early
		return true, c.nextBackoff(attempt)
	}
	return false, 0
}

// moved to client_retry.go

// moved to client_retry.go

// moved to client_retry.go

// moved to client_headers.go

// Generate sends a non-stream request to Code Assist v1internal:generateContent.
//
// IMPORTANT: Caller MUST close resp.Body if resp is non-nil and err is nil.
// Example:
//   resp, err := client.Generate(ctx, payload)
//   if err != nil { return err }
//   defer resp.Body.Close()
func (c *Client) Generate(ctx context.Context, payload []byte) (*http.Response, error) {
	useURL := c.cfg.CodeAssist + "/v1internal:generateContent"
	return c.postJSON(ctx, useURL, payload, c.getToken())
}

// Stream sends a stream request to Code Assist v1internal:streamGenerateContent.
//
// IMPORTANT: Caller MUST close resp.Body if resp is non-nil and err is nil.
// Example:
//   resp, err := client.Stream(ctx, payload)
//   if err != nil { return err }
//   defer resp.Body.Close()
func (c *Client) Stream(ctx context.Context, payload []byte) (*http.Response, error) {
	useURL := c.cfg.CodeAssist + "/v1internal:streamGenerateContent?alt=sse"
	return c.postJSON(ctx, useURL, payload, c.getToken())
}

// CountTokens sends a request to Code Assist v1internal:countTokens.
//
// IMPORTANT: Caller MUST close resp.Body if resp is non-nil and err is nil.
// Example:
//   resp, err := client.CountTokens(ctx, payload)
//   if err != nil { return err }
//   defer resp.Body.Close()
func (c *Client) CountTokens(ctx context.Context, payload []byte) (*http.Response, error) {
	useURL := c.cfg.CodeAssist + "/v1internal:countTokens"
	return c.postJSON(ctx, useURL, payload, c.getToken())
}

func (c *Client) Action(ctx context.Context, action string, payload []byte) (*http.Response, error) {
	url := c.cfg.CodeAssist + "/v1internal:" + action
	return c.postJSON(ctx, url, payload, c.getToken())
}
