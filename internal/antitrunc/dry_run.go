package antitrunc

import (
	"encoding/json"
	"fmt"
)

// DryRunRequest represents a dry-run request for anti-truncation debugging
type DryRunRequest struct {
	Text    string          `json:"text"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Rules   []RegexRule     `json:"rules,omitempty"`
}

// DryRunResponse represents the response from a dry-run request
type DryRunResponse struct {
	OriginalText     string                  `json:"original_text,omitempty"`
	ProcessedText    string                  `json:"processed_text,omitempty"`
	OriginalPayload  json.RawMessage         `json:"original_payload,omitempty"`
	ProcessedPayload json.RawMessage         `json:"processed_payload,omitempty"`
	RulesApplied     []RuleApplicationResult `json:"rules_applied"`
	Summary          DryRunSummary           `json:"summary"`
}

// RuleApplicationResult represents the result of applying a single rule
type RuleApplicationResult struct {
	RuleIndex   int      `json:"rule_index"`
	Pattern     string   `json:"pattern"`
	Replacement string   `json:"replacement"`
	Matches     int      `json:"matches"`
	Examples    []string `json:"examples,omitempty"` // Sample matches (up to 3)
}

// DryRunSummary provides a summary of the dry-run operation
type DryRunSummary struct {
	TotalRules      int  `json:"total_rules"`
	RulesMatched    int  `json:"rules_matched"`
	TotalMatches    int  `json:"total_matches"`
	TextModified    bool `json:"text_modified"`
	PayloadModified bool `json:"payload_modified"`
}

// DryRunText performs a dry-run of regex replacement on text
func DryRunText(text string, rules []RegexRule) (*DryRunResponse, error) {
	if len(rules) == 0 {
		return &DryRunResponse{
			OriginalText:  text,
			ProcessedText: text,
			RulesApplied:  []RuleApplicationResult{},
			Summary: DryRunSummary{
				TotalRules:   0,
				RulesMatched: 0,
				TotalMatches: 0,
				TextModified: false,
			},
		}, nil
	}

	replacer, err := NewRegexReplacer(rules)
	if err != nil {
		return nil, fmt.Errorf("failed to create regex replacer: %w", err)
	}
	processedText, matchCounts := replacer.ApplyToText(text)

	rulesApplied := make([]RuleApplicationResult, 0, len(rules))
	totalMatches := 0
	rulesMatched := 0

	for i, rule := range rules {
		matches := matchCounts[i]
		if matches > 0 {
			rulesMatched++
			totalMatches += matches
		}

		result := RuleApplicationResult{
			RuleIndex:   i,
			Pattern:     rule.Pattern,
			Replacement: rule.Replacement,
			Matches:     matches,
		}

		// Extract sample matches (up to 3)
		if matches > 0 && rule.compiled != nil {
			examples := rule.compiled.FindAllString(text, 3)
			result.Examples = examples
		}

		rulesApplied = append(rulesApplied, result)
	}

	return &DryRunResponse{
		OriginalText:  text,
		ProcessedText: processedText,
		RulesApplied:  rulesApplied,
		Summary: DryRunSummary{
			TotalRules:   len(rules),
			RulesMatched: rulesMatched,
			TotalMatches: totalMatches,
			TextModified: text != processedText,
		},
	}, nil
}

// DryRunPayload performs a dry-run of regex replacement on a JSON payload
func DryRunPayload(payload json.RawMessage, rules []RegexRule) (*DryRunResponse, error) {
	if len(rules) == 0 {
		return &DryRunResponse{
			OriginalPayload:  payload,
			ProcessedPayload: payload,
			RulesApplied:     []RuleApplicationResult{},
			Summary: DryRunSummary{
				TotalRules:      0,
				RulesMatched:    0,
				TotalMatches:    0,
				PayloadModified: false,
			},
		}, nil
	}

	// Parse payload
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}

	replacer, err := NewRegexReplacer(rules)
	if err != nil {
		return nil, fmt.Errorf("failed to create regex replacer: %w", err)
	}
	processedMap, matchCounts := replacer.ApplyToPayload(payloadMap)

	// Marshal back to JSON
	processedPayload, err := json.MarshalIndent(processedMap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal processed payload: %w", err)
	}

	rulesApplied := make([]RuleApplicationResult, 0, len(rules))
	totalMatches := 0
	rulesMatched := 0

	for i, rule := range rules {
		matches := matchCounts[i]
		if matches > 0 {
			rulesMatched++
			totalMatches += matches
		}

		result := RuleApplicationResult{
			RuleIndex:   i,
			Pattern:     rule.Pattern,
			Replacement: rule.Replacement,
			Matches:     matches,
		}

		rulesApplied = append(rulesApplied, result)
	}

	return &DryRunResponse{
		OriginalPayload:  payload,
		ProcessedPayload: json.RawMessage(processedPayload),
		RulesApplied:     rulesApplied,
		Summary: DryRunSummary{
			TotalRules:      len(rules),
			RulesMatched:    rulesMatched,
			TotalMatches:    totalMatches,
			PayloadModified: string(payload) != string(processedPayload),
		},
	}, nil
}

// DryRun performs a dry-run of regex replacement on either text or payload
func DryRun(req *DryRunRequest) (*DryRunResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// If both text and payload are provided, prefer payload
	if len(req.Payload) > 0 {
		return DryRunPayload(req.Payload, req.Rules)
	}

	if req.Text != "" {
		return DryRunText(req.Text, req.Rules)
	}

	return nil, fmt.Errorf("either text or payload must be provided")
}
