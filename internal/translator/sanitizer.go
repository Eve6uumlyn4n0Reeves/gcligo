package translator

import (
	"os"
	"regexp"
	"strings"
	"sync"

	"gcli2api-go/internal/common"
	log "github.com/sirupsen/logrus"
)

var (
	defaultAgePattern = `(?i)(?:[1-9]|1[0-8])岁(?:的)?|(?:十一|十二|十三|十四|十五|十六|十七|十八|十|一|二|三|四|五|六|七|八|九)岁(?:的)?`
	sanitizeOnce      sync.Once
	sanitizerMu       sync.RWMutex
	compiledPatterns  []*regexp.Regexp
	sanitizerEnabled  = false
	doneInstrEnabled  = true
)

func initSanitizer() {
	sanitizeOnce.Do(func() {
		enabled := sanitizerEnabled
		if v := strings.ToLower(strings.TrimSpace(os.Getenv("SANITIZER_ENABLED"))); v != "" {
			enabled = v == "true" || v == "1" || v == "yes" || v == "on"
		}
		if v := strings.ToLower(strings.TrimSpace(os.Getenv("DONE_INSTRUCTION_ENABLED"))); v != "" {
			doneInstrEnabled = v == "true" || v == "1" || v == "yes" || v == "on"
		}

		patterns := []string{defaultAgePattern}
		if raw := strings.TrimSpace(os.Getenv("SANITIZER_PATTERNS")); raw != "" {
			if strings.Contains(raw, "|") {
				patterns = strings.Split(raw, "|")
			} else {
				patterns = strings.Split(raw, ",")
			}
		}
		configureSanitizer(enabled, patterns)
	})
}

// ConfigureSanitizer updates runtime sanitizer settings overriding environment defaults.
func ConfigureSanitizer(enabled bool, patterns []string) {
	if len(patterns) == 0 {
		patterns = []string{defaultAgePattern}
	}
	configureSanitizer(enabled, patterns)
}

func configureSanitizer(enabled bool, patterns []string) {
	sanitizerMu.Lock()
	defer sanitizerMu.Unlock()

	sanitizerEnabled = enabled
	compiledPatterns = compiledPatterns[:0]
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if re, err := regexp.Compile(p); err == nil {
			compiledPatterns = append(compiledPatterns, re)
		} else {
			log.Warnf("invalid sanitizer pattern ignored: %q, err=%v", p, err)
		}
	}
	if len(compiledPatterns) == 0 {
		compiledPatterns = []*regexp.Regexp{regexp.MustCompile(defaultAgePattern)}
	}
}

func sanitizeText(text string) string {
	if text == "" {
		return text
	}
	initSanitizer()
	sanitizerMu.RLock()
	enabled := sanitizerEnabled
	patterns := compiledPatterns
	sanitizerMu.RUnlock()
	if !enabled {
		return text
	}
	out := text
	for _, re := range patterns {
		out = re.ReplaceAllString(out, "")
	}
	return out
}

func sanitizeParts(parts []interface{}) []interface{} {
	for _, part := range parts {
		if mp, ok := part.(map[string]interface{}); ok {
			if text, ok := mp["text"].(string); ok {
				mp["text"] = sanitizeText(text)
			}
		}
	}
	return parts
}

// SanitizeOutputText applies configured sanitizer patterns to a single text blob.
func SanitizeOutputText(text string) string {
	return sanitizeText(text)
}

func sanitizeMessages(messages []interface{}) []interface{} {
	for _, item := range messages {
		msg, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if parts, ok := msg["parts"].([]interface{}); ok {
			msg["parts"] = sanitizeParts(parts)
		}
	}
	return messages
}

func ensureDoneInstruction(parts *[]interface{}) {
	if parts == nil {
		return
	}
	initSanitizer()
	sanitizerMu.RLock()
	enabled := doneInstrEnabled
	sanitizerMu.RUnlock()
	if !enabled {
		return
	}
	for _, part := range *parts {
		mp, ok := part.(map[string]interface{})
		if !ok {
			continue
		}
		if text, ok := mp["text"].(string); ok {
			if strings.Contains(text, common.DoneMarker) {
				return
			}
		}
	}
	*parts = append(*parts, map[string]interface{}{
		"text": common.DoneInstruction,
	})
}
