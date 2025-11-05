package antitrunc

import (
	"bytes"
	"encoding/json"
	"strings"

	"gcli2api-go/internal/common"
	"gcli2api-go/internal/translator"
)

var defaultIndicators = []string{
	"...",
	"[truncated]",
	"[continued]",
	"[incomplete]",
	"<truncated>",
	"[to be continued]",
	"[继续]",
	"[continue]",
	"[未完]",
}

// Config captures shared anti-truncation heuristics.
type Config struct {
	MinCompletionLen     int
	TruncationIndicators []string
}

// DefaultConfig returns the baseline detector configuration.
func DefaultConfig() Config {
	return Config{
		MinCompletionLen:     50,
		TruncationIndicators: append([]string(nil), defaultIndicators...),
	}
}

func (c Config) ensure() Config {
	if c.MinCompletionLen <= 0 {
		c.MinCompletionLen = 50
	}
	if len(c.TruncationIndicators) == 0 {
		c.TruncationIndicators = append([]string(nil), defaultIndicators...)
	}
	c.TruncationIndicators = normalizeIndicators(c.TruncationIndicators)
	return c
}

// ResponseComplete reports whether the accumulated text looks complete.
func (c Config) ResponseComplete(text string) bool {
	cfg := c.ensure()
	if common.HasDoneMarker(text) {
		return true
	}

	trimmed := strings.TrimSpace(text)
	if len(trimmed) < cfg.MinCompletionLen {
		return false
	}

	lower := strings.ToLower(trimmed)
	for _, indicator := range cfg.TruncationIndicators {
		if indicator == "" {
			continue
		}
		if strings.Contains(lower, indicator) {
			return false
		}
	}

	if len(trimmed) == 0 {
		return false
	}

	last := trimmed[len(trimmed)-1]
	switch last {
	case '.', '!', '?', '"', ')':
		return true
	}

	return len(trimmed) > cfg.MinCompletionLen*2
}

// AppearsTruncated heuristically checks if text likely ended prematurely.
func (c Config) AppearsTruncated(text string) bool {
	cfg := c.ensure()

	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}

	if common.HasDoneMarker(trimmed) {
		return false
	}

	lower := strings.ToLower(trimmed)
	for _, indicator := range cfg.TruncationIndicators {
		if indicator == "" {
			continue
		}
		if strings.HasSuffix(lower, indicator) {
			return true
		}
	}

	rs := []rune(trimmed)
	last := rs[len(rs)-1]
	switch last {
	case '.', '!', '?', '。', '！', '？', '\n', '"', '\'':
		return false
	}

	return len(rs) > 1000
}

// CleanContinuationText removes done markers and applies sanitizer before reuse.
func CleanContinuationText(text string) string {
	cleaned := common.StripDoneMarker(text)
	return translator.SanitizeOutputText(cleaned)
}

// BuildContinuationPayload clones the original payload and appends continuation instructions.
// Expected shape:
//
//	{ "model": "...", "project": "...", "request": { "contents": [ ... ] } }
func BuildContinuationPayload(orig []byte, soFar string, contText string) []byte {
	if len(orig) == 0 {
		return orig
	}

	var root map[string]any
	if err := json.Unmarshal(orig, &root); err != nil {
		return orig
	}

	b, _ := json.Marshal(root)
	var cloned map[string]any
	_ = json.Unmarshal(b, &cloned)

	req, _ := cloned["request"].(map[string]any)
	if req == nil {
		req = map[string]any{}
		cloned["request"] = req
	}

	carr, _ := req["contents"].([]any)
	cleanSoFar := CleanContinuationText(soFar)
	if cleanSoFar != "" {
		carr = append(carr, map[string]any{
			"role":  "model",
			"parts": []any{map[string]any{"text": cleanSoFar}},
		})
	}

	if strings.TrimSpace(contText) == "" {
		contText = "continue"
	}

	carr = append(carr, map[string]any{
		"role":  "user",
		"parts": []any{map[string]any{"text": contText}},
	})
	req["contents"] = carr

	out, err := json.Marshal(cloned)
	if err != nil {
		return orig
	}

	return bytes.Clone(out)
}

func normalizeIndicators(indicators []string) []string {
	out := make([]string, 0, len(indicators))
	for _, indicator := range indicators {
		indicator = strings.ToLower(strings.TrimSpace(indicator))
		out = append(out, indicator)
	}
	return out
}
