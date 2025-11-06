package gemini

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"time"

	mw "gcli2api-go/internal/middleware"
)

// doAttempt executes a single HTTP attempt with retry policy applied to the payload.
// It returns the final response, error, total duration, HTTP status code, and retry count.
//
// IMPORTANT: Caller is responsible for closing resp.Body if resp is non-nil.
// The response body is NOT automatically closed by this function.
func (c *Client) doAttempt(ctx context.Context, url string, payload []byte, bearer string) (*http.Response, error, time.Duration, int, int) {
	makeReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if strings.Contains(url, "alt=sse") || strings.Contains(url, "$alt=sse") {
			req.Header.Set("Accept", "text/event-stream")
		} else {
			req.Header.Set("Accept", "application/json")
		}
		c.applyDefaultHeaders(ctx, req, bearer)
		return req, nil
	}

	doOnce := func() (*http.Response, error, time.Duration) {
		// Check if context is already cancelled before making request
		if err := ctx.Err(); err != nil {
			return nil, err, 0
		}
		req, err := makeReq()
		if err != nil {
			return nil, err, 0
		}
		start := time.Now()
		resp, err := c.cli.Do(req)
		return resp, err, time.Since(start)
	}

	resp, err, dur := doOnce()
	tries := 0
	if c.cfg.RetryEnabled && c.cfg.RetryMax > 0 {
		for should, wait := c.shouldRetry(resp, err, tries); should && tries < c.cfg.RetryMax; tries++ {
			if resp != nil {
				_ = resp.Body.Close()
			}
			time.Sleep(wait)
			resp, err, dur = doOnce()
		}
	}

	status := getStatus(resp)
	if c.caller != "" {
		mw.RecordUpstreamWithServer("gemini", c.caller, dur, status, err != nil)
	} else {
		mw.RecordUpstream("gemini", dur, status, err != nil)
	}
	if tries > 0 {
		mw.RecordUpstreamRetry("gemini", tries, err == nil)
	}
	if err != nil {
		mw.RecordUpstreamError("gemini", classifyErr(err))
	}

	return resp, err, dur, status, tries
}
