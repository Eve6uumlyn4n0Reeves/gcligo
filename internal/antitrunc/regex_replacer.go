package antitrunc

import (
	"encoding/json"
	"regexp"
	"sync"

	log "github.com/sirupsen/logrus"
)

// RegexRule represents a single regex replacement rule
type RegexRule struct {
	Name        string
	Pattern     string
	Replacement string
	Enabled     bool
	compiled    *regexp.Regexp
}

// RegexReplacer handles regex-based text replacements
type RegexReplacer struct {
	rules []RegexRule
	mu    sync.RWMutex
}

// NewRegexReplacer creates a new regex replacer with the given rules
func NewRegexReplacer(rules []RegexRule) (*RegexReplacer, error) {
	replacer := &RegexReplacer{
		rules: make([]RegexRule, 0, len(rules)),
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		compiled, err := regexp.Compile(rule.Pattern)
		if err != nil {
			log.WithFields(log.Fields{
				"rule":    rule.Name,
				"pattern": rule.Pattern,
				"error":   err,
			}).Warn("Failed to compile regex pattern, skipping rule")
			continue
		}

		rule.compiled = compiled
		replacer.rules = append(replacer.rules, rule)
	}

	log.WithField("count", len(replacer.rules)).Info("Regex replacer initialized")
	return replacer, nil
}

// ApplyToText applies all regex replacements to the given text
func (r *RegexReplacer) ApplyToText(text string) string {
	if text == "" {
		return text
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := text
	totalReplacements := 0

	for _, rule := range r.rules {
		if rule.compiled == nil {
			continue
		}

		// Count matches before replacement
		matches := rule.compiled.FindAllString(result, -1)
		if len(matches) > 0 {
			result = rule.compiled.ReplaceAllString(result, rule.Replacement)
			totalReplacements += len(matches)

			log.WithFields(log.Fields{
				"rule":    rule.Name,
				"matches": len(matches),
			}).Debug("Applied regex replacement")
		}
	}

	if totalReplacements > 0 {
		log.WithField("total_replacements", totalReplacements).Info("Applied regex replacements to text")
	}

	return result
}

// ApplyToPayload applies regex replacements to text content in a request payload
// Expected payload structure: { "model": "...", "project": "...", "request": { "contents": [...] } }
func (r *RegexReplacer) ApplyToPayload(payload []byte) []byte {
	if len(payload) == 0 {
		return payload
	}

	r.mu.RLock()
	if len(r.rules) == 0 {
		r.mu.RUnlock()
		return payload
	}
	r.mu.RUnlock()

	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		log.WithError(err).Warn("Failed to unmarshal payload for regex replacement")
		return payload
	}

	modified := false

	// Navigate to request.contents
	req, ok := root["request"].(map[string]any)
	if !ok {
		return payload
	}

	contents, ok := req["contents"].([]any)
	if !ok {
		return payload
	}

	// Process each content item
	for i, content := range contents {
		contentMap, ok := content.(map[string]any)
		if !ok {
			continue
		}

		parts, ok := contentMap["parts"].([]any)
		if !ok {
			continue
		}

		// Process each part
		for j, part := range parts {
			partMap, ok := part.(map[string]any)
			if !ok {
				continue
			}

			// Apply replacements to text field
			if text, ok := partMap["text"].(string); ok && text != "" {
				newText := r.ApplyToText(text)
				if newText != text {
					partMap["text"] = newText
					parts[j] = partMap
					modified = true
				}
			}
		}

		if modified {
			contentMap["parts"] = parts
			contents[i] = contentMap
		}
	}

	if !modified {
		return payload
	}

	req["contents"] = contents
	root["request"] = req

	result, err := json.Marshal(root)
	if err != nil {
		log.WithError(err).Warn("Failed to marshal payload after regex replacement")
		return payload
	}

	return result
}

// UpdateRules updates the regex rules dynamically
func (r *RegexReplacer) UpdateRules(rules []RegexRule) error {
	newRules := make([]RegexRule, 0, len(rules))

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		compiled, err := regexp.Compile(rule.Pattern)
		if err != nil {
			log.WithFields(log.Fields{
				"rule":    rule.Name,
				"pattern": rule.Pattern,
				"error":   err,
			}).Warn("Failed to compile regex pattern, skipping rule")
			continue
		}

		rule.compiled = compiled
		newRules = append(newRules, rule)
	}

	r.mu.Lock()
	r.rules = newRules
	r.mu.Unlock()

	log.WithField("count", len(newRules)).Info("Regex rules updated")
	return nil
}

// GetRules returns a copy of the current rules (without compiled regex)
func (r *RegexReplacer) GetRules() []RegexRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules := make([]RegexRule, len(r.rules))
	for i, rule := range r.rules {
		rules[i] = RegexRule{
			Name:        rule.Name,
			Pattern:     rule.Pattern,
			Replacement: rule.Replacement,
			Enabled:     rule.Enabled,
		}
	}
	return rules
}

// RuleCount returns the number of active rules
func (r *RegexReplacer) RuleCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.rules)
}

