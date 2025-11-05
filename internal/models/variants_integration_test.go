package models

import (
	"testing"
)

func TestGenerateVariantsForModels(t *testing.T) {
	baseModels := []string{"gemini-2.5-pro", "gemini-2.5-flash"}
	variants := GenerateVariantsForModels(baseModels)

	if len(variants) == 0 {
		t.Fatal("No variants generated")
	}

	t.Logf("Generated %d variants for %d base models", len(variants), len(baseModels))

	// 检查必须存在的变体
	requiredVariants := []string{
		"gemini-2.5-pro",
		"假流式/gemini-2.5-pro",
		"流式抗截断/gemini-2.5-flash",
		"gemini-2.5-pro-maxthinking",
		"gemini-2.5-flash-nothinking",
		"gemini-2.5-pro-search",
		"假流式/gemini-2.5-pro-maxthinking",
		"流式抗截断/gemini-2.5-flash-search",
	}

	variantMap := make(map[string]bool)
	for _, v := range variants {
		variantMap[v] = true
	}

	for _, required := range requiredVariants {
		if !variantMap[required] {
			t.Errorf("Required variant not found: %s", required)
		} else {
			t.Logf("✓ Found: %s", required)
		}
	}

	// 统计各类变体
	stats := map[string]int{
		"base":        0,
		"fake_stream": 0,
		"anti_trunc":  0,
		"thinking":    0,
		"search":      0,
		"combined":    0,
	}

	for _, v := range variants {
		features := ParseModelFeatures(v)

		if !features.FakeStreaming && !features.AntiTruncation &&
			features.ThinkingLevel == "auto" && !features.Search {
			stats["base"]++
		} else if features.FakeStreaming && !features.AntiTruncation &&
			features.ThinkingLevel == "auto" && !features.Search {
			stats["fake_stream"]++
		} else if !features.FakeStreaming && features.AntiTruncation &&
			features.ThinkingLevel == "auto" && !features.Search {
			stats["anti_trunc"]++
		} else if !features.FakeStreaming && !features.AntiTruncation &&
			features.ThinkingLevel != "auto" && !features.Search {
			stats["thinking"]++
		} else if !features.FakeStreaming && !features.AntiTruncation &&
			features.ThinkingLevel == "auto" && features.Search {
			stats["search"]++
		} else {
			stats["combined"]++
		}
	}

	t.Logf("Variant statistics:")
	t.Logf("  Base models: %d", stats["base"])
	t.Logf("  Fake streaming: %d", stats["fake_stream"])
	t.Logf("  Anti-truncation: %d", stats["anti_trunc"])
	t.Logf("  Thinking suffixes: %d", stats["thinking"])
	t.Logf("  Search suffix: %d", stats["search"])
	t.Logf("  Combined variants: %d", stats["combined"])
}

func TestModelFeatureParsing(t *testing.T) {
	tests := []struct {
		model              string
		expectedBase       string
		expectedFakeStream bool
		expectedAntiTrunc  bool
		expectedSearch     bool
		expectedThinking   string
	}{
		{
			model:              "gemini-2.5-pro",
			expectedBase:       "gemini-2.5-pro",
			expectedFakeStream: false,
			expectedAntiTrunc:  false,
			expectedSearch:     false,
			expectedThinking:   "auto",
		},
		{
			model:              "假流式/gemini-2.5-pro",
			expectedBase:       "gemini-2.5-pro",
			expectedFakeStream: true,
			expectedAntiTrunc:  false,
			expectedSearch:     false,
			expectedThinking:   "auto",
		},
		{
			model:              "流式抗截断/gemini-2.5-flash",
			expectedBase:       "gemini-2.5-flash",
			expectedFakeStream: false,
			expectedAntiTrunc:  true,
			expectedSearch:     false,
			expectedThinking:   "auto",
		},
		{
			model:              "gemini-2.5-pro-maxthinking",
			expectedBase:       "gemini-2.5-pro",
			expectedFakeStream: false,
			expectedAntiTrunc:  false,
			expectedSearch:     false,
			expectedThinking:   "max",
		},
		{
			model:              "gemini-2.5-flash-nothinking",
			expectedBase:       "gemini-2.5-flash",
			expectedFakeStream: false,
			expectedAntiTrunc:  false,
			expectedSearch:     false,
			expectedThinking:   "none",
		},
		{
			model:              "gemini-2.5-pro-search",
			expectedBase:       "gemini-2.5-pro",
			expectedFakeStream: false,
			expectedAntiTrunc:  false,
			expectedSearch:     true,
			expectedThinking:   "auto",
		},
		{
			model:              "假流式/gemini-2.5-pro-maxthinking",
			expectedBase:       "gemini-2.5-pro",
			expectedFakeStream: true,
			expectedAntiTrunc:  false,
			expectedSearch:     false,
			expectedThinking:   "max",
		},
		{
			model:              "流式抗截断/gemini-2.5-flash-search",
			expectedBase:       "gemini-2.5-flash",
			expectedFakeStream: false,
			expectedAntiTrunc:  true,
			expectedSearch:     true,
			expectedThinking:   "auto",
		},
		{
			model:              "假流式/gemini-2.5-pro-maxthinking-search",
			expectedBase:       "gemini-2.5-pro",
			expectedFakeStream: true,
			expectedAntiTrunc:  false,
			expectedSearch:     true,
			expectedThinking:   "max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			features := ParseModelFeatures(tt.model)

			if features.Base != tt.expectedBase {
				t.Errorf("Base mismatch: got %s, want %s", features.Base, tt.expectedBase)
			}
			if features.FakeStreaming != tt.expectedFakeStream {
				t.Errorf("FakeStreaming mismatch: got %v, want %v", features.FakeStreaming, tt.expectedFakeStream)
			}
			if features.AntiTruncation != tt.expectedAntiTrunc {
				t.Errorf("AntiTruncation mismatch: got %v, want %v", features.AntiTruncation, tt.expectedAntiTrunc)
			}
			if features.Search != tt.expectedSearch {
				t.Errorf("Search mismatch: got %v, want %v", features.Search, tt.expectedSearch)
			}
			if features.ThinkingLevel != tt.expectedThinking {
				t.Errorf("ThinkingLevel mismatch: got %s, want %s", features.ThinkingLevel, tt.expectedThinking)
			}

			t.Logf("✓ %s -> base=%s, fake=%v, anti=%v, search=%v, thinking=%s",
				tt.model, features.Base, features.FakeStreaming, features.AntiTruncation,
				features.Search, features.ThinkingLevel)
		})
	}
}
