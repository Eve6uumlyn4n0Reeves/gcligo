package gemini

import (
	"math"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (c *Client) nextBackoff(attempt int) time.Duration {
	base := float64(time.Duration(c.cfg.RetryIntervalSec) * time.Second)
	max := float64(time.Duration(c.cfg.RetryMaxIntervalSec) * time.Second)
	if base <= 0 {
		base = float64(time.Second)
	}
	if max <= 0 {
		max = float64(8 * time.Second)
	}
	dur := base * math.Pow(2, float64(attempt))
	if dur > max {
		dur = max
	}
	jitter := 0.5 + rand.Float64()
	return time.Duration(dur * jitter)
}

func parseRetryAfter(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			secs = 0
		}
		return time.Duration(secs) * time.Second, true
	}
	layouts := []string{time.RFC1123, time.RFC1123Z, time.RFC850, time.ANSIC}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			d := time.Until(t)
			if d < 0 {
				d = 0
			}
			return d, true
		}
	}
	return 0, false
}

func classifyErr(err error) string {
	if err == nil {
		return ""
	}
	if ue, ok := err.(*url.Error); ok {
		if ue.Timeout() {
			return "timeout"
		}
		if ue.Err != nil {
			s := ue.Err.Error()
			if strings.Contains(s, "no such host") {
				return "dns"
			}
			if strings.Contains(s, "connection reset") {
				return "conn_reset"
			}
			if strings.Contains(s, "broken pipe") {
				return "conn_broken_pipe"
			}
			if strings.Contains(s, "i/o timeout") {
				return "timeout"
			}
		}
	}
	s := err.Error()
	if strings.Contains(s, "deadline exceeded") {
		return "deadline"
	}
	if strings.Contains(s, "context canceled") {
		return "canceled"
	}
	if strings.Contains(s, "no such host") {
		return "dns"
	}
	if strings.Contains(s, "connection reset") {
		return "conn_reset"
	}
	if strings.Contains(s, "broken pipe") {
		return "conn_broken_pipe"
	}
	if strings.Contains(s, "timeout") {
		return "timeout"
	}
	return "other"
}
