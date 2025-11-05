package models

import "strings"

// VariantConfig defines configurable model variant patterns
type VariantConfig struct {
	FakeStreamingPrefix  string            `json:"fake_streaming_prefix"`
	AntiTruncationPrefix string            `json:"anti_truncation_prefix"`
	ThinkingSuffixes     map[string]string `json:"thinking_suffixes"` // level -> suffix
	SearchSuffix         string            `json:"search_suffix"`
	CustomPrefixes       []string          `json:"custom_prefixes"`
	CustomSuffixes       []string          `json:"custom_suffixes"`
}

// DefaultVariantConfig returns the default variant configuration
func DefaultVariantConfig() *VariantConfig {
	return &VariantConfig{
		FakeStreamingPrefix:  "假流式/",
		AntiTruncationPrefix: "流式抗截断/",
		ThinkingSuffixes: map[string]string{
			"max":  "-maxthinking",
			"high": "-maxthinking",
			"none": "-nothinking",
			"low":  "-lowthinking",
			"med":  "-medthinking",
			"auto": "-autothinking",
		},
		SearchSuffix:   "-search",
		CustomPrefixes: []string{},
		CustomSuffixes: []string{},
	}
}

// Generate full variant list used for exposure in /models
func AllVariants() []string {
	return AllVariantsWithConfig(DefaultVariantConfig())
}

// GenerateVariantsForModels generates all variants for the given base models
func GenerateVariantsForModels(baseModels []string) []string {
	return GenerateVariantsForModelsWithConfig(baseModels, DefaultVariantConfig())
}

// GenerateVariantsForModelsWithConfig generates variants for specific base models using custom configuration
func GenerateVariantsForModelsWithConfig(baseModels []string, config *VariantConfig) []string {
	if config == nil {
		config = DefaultVariantConfig()
	}
	if len(baseModels) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(baseModels)*30) // Estimate: ~30 variants per base model

	// Collect all suffixes
	allSuffixes := []string{""}

	// Add thinking suffixes
	for _, suffix := range config.ThinkingSuffixes {
		if suffix != "" {
			allSuffixes = append(allSuffixes, suffix)
		}
	}

	// Add search suffix
	if config.SearchSuffix != "" {
		allSuffixes = append(allSuffixes, config.SearchSuffix)
		// Also add combined thinking+search suffixes
		for _, thinkingSuffix := range config.ThinkingSuffixes {
			if thinkingSuffix != "" {
				allSuffixes = append(allSuffixes, thinkingSuffix+config.SearchSuffix)
			}
		}
	}

	// Add custom suffixes
	allSuffixes = append(allSuffixes, config.CustomSuffixes...)

	// Collect all prefixes
	allPrefixes := []string{""}
	if config.FakeStreamingPrefix != "" {
		allPrefixes = append(allPrefixes, config.FakeStreamingPrefix)
	}
	if config.AntiTruncationPrefix != "" {
		allPrefixes = append(allPrefixes, config.AntiTruncationPrefix)
	}
	allPrefixes = append(allPrefixes, config.CustomPrefixes...)

	// Generate all combinations
	for _, base := range baseModels {
		for _, suffix := range allSuffixes {
			modelWithSuffix := base + suffix
			for _, prefix := range allPrefixes {
				variant := prefix + modelWithSuffix
				out = append(out, variant)
			}
		}
	}

	return out
}

// AllVariantsWithConfig generates variants using custom configuration
func AllVariantsWithConfig(config *VariantConfig) []string {
	if config == nil {
		config = DefaultVariantConfig()
	}

	out := make([]string, 0, 256)
	bases := DefaultBaseModels()

	// Collect all suffixes
	allSuffixes := []string{""}

	// Add thinking suffixes
	for _, suffix := range config.ThinkingSuffixes {
		if suffix != "" {
			allSuffixes = append(allSuffixes, suffix)
		}
	}

	// Add search suffix
	if config.SearchSuffix != "" {
		allSuffixes = append(allSuffixes, config.SearchSuffix)
		// Also add combined thinking+search suffixes
		for _, thinkingSuffix := range config.ThinkingSuffixes {
			if thinkingSuffix != "" {
				allSuffixes = append(allSuffixes, thinkingSuffix+config.SearchSuffix)
			}
		}
	}

	// Add custom suffixes
	allSuffixes = append(allSuffixes, config.CustomSuffixes...)

	// Collect all prefixes
	allPrefixes := []string{""}
	if config.FakeStreamingPrefix != "" {
		allPrefixes = append(allPrefixes, config.FakeStreamingPrefix)
	}
	if config.AntiTruncationPrefix != "" {
		allPrefixes = append(allPrefixes, config.AntiTruncationPrefix)
	}
	allPrefixes = append(allPrefixes, config.CustomPrefixes...)

	// Generate all combinations
	for _, base := range bases {
		for _, suffix := range allSuffixes {
			modelWithSuffix := base + suffix
			for _, prefix := range allPrefixes {
				variant := prefix + modelWithSuffix
				out = append(out, variant)
			}
		}
	}

	return out
}

func IsFakeStreaming(model string) bool {
	return IsFakeStreamingWithConfig(model, DefaultVariantConfig())
}
func IsAntiTruncation(model string) bool {
	return IsAntiTruncationWithConfig(model, DefaultVariantConfig())
}

func IsFakeStreamingWithConfig(model string, config *VariantConfig) bool {
	if config == nil {
		config = DefaultVariantConfig()
	}
	return config.FakeStreamingPrefix != "" && strings.HasPrefix(model, config.FakeStreamingPrefix)
}

func IsAntiTruncationWithConfig(model string, config *VariantConfig) bool {
	if config == nil {
		config = DefaultVariantConfig()
	}
	return config.AntiTruncationPrefix != "" && strings.HasPrefix(model, config.AntiTruncationPrefix)
}

func BaseFromFeature(model string) string {
	return BaseFromFeatureWithConfig(model, DefaultVariantConfig())
}

func BaseFromFeatureWithConfig(model string, config *VariantConfig) string {
	if config == nil {
		config = DefaultVariantConfig()
	}

	result := model

	// Remove prefixes
	if config.FakeStreamingPrefix != "" && strings.HasPrefix(result, config.FakeStreamingPrefix) {
		result = strings.TrimPrefix(result, config.FakeStreamingPrefix)
	}
	if config.AntiTruncationPrefix != "" && strings.HasPrefix(result, config.AntiTruncationPrefix) {
		result = strings.TrimPrefix(result, config.AntiTruncationPrefix)
	}
	for _, prefix := range config.CustomPrefixes {
		if prefix != "" && strings.HasPrefix(result, prefix) {
			result = strings.TrimPrefix(result, prefix)
			break // Only remove first matching prefix
		}
	}

	// Remove suffixes (in reverse order of preference)
	if config.SearchSuffix != "" && strings.HasSuffix(result, config.SearchSuffix) {
		result = strings.TrimSuffix(result, config.SearchSuffix)
	}
	for _, suffix := range config.ThinkingSuffixes {
		if suffix != "" && strings.HasSuffix(result, suffix) {
			result = strings.TrimSuffix(result, suffix)
			break // Only remove first matching suffix
		}
	}
	for _, suffix := range config.CustomSuffixes {
		if suffix != "" && strings.HasSuffix(result, suffix) {
			result = strings.TrimSuffix(result, suffix)
			break // Only remove first matching suffix
		}
	}

	return result
}

func IsSearch(model string) bool      { return IsSearchWithConfig(model, DefaultVariantConfig()) }
func IsNoThinking(model string) bool  { return IsNoThinkingWithConfig(model, DefaultVariantConfig()) }
func IsMaxThinking(model string) bool { return IsMaxThinkingWithConfig(model, DefaultVariantConfig()) }

func IsSearchWithConfig(model string, config *VariantConfig) bool {
	if config == nil {
		config = DefaultVariantConfig()
	}
	return config.SearchSuffix != "" && strings.Contains(model, config.SearchSuffix)
}

func IsNoThinkingWithConfig(model string, config *VariantConfig) bool {
	if config == nil {
		config = DefaultVariantConfig()
	}
	if suffix, ok := config.ThinkingSuffixes["none"]; ok && suffix != "" {
		return strings.Contains(model, suffix)
	}
	return strings.Contains(model, "-nothinking") // fallback
}

func IsMaxThinkingWithConfig(model string, config *VariantConfig) bool {
	if config == nil {
		config = DefaultVariantConfig()
	}
	if suffix, ok := config.ThinkingSuffixes["max"]; ok && suffix != "" {
		return strings.Contains(model, suffix)
	}
	if suffix, ok := config.ThinkingSuffixes["high"]; ok && suffix != "" {
		return strings.Contains(model, suffix)
	}
	return strings.Contains(model, "-maxthinking") // fallback
}

// GetThinkingLevel extracts thinking level from model name
func GetThinkingLevel(model string) string {
	return GetThinkingLevelWithConfig(model, DefaultVariantConfig())
}

func GetThinkingLevelWithConfig(model string, config *VariantConfig) string {
	if config == nil {
		config = DefaultVariantConfig()
	}

	for _, level := range []string{"max", "high", "med", "low", "auto", "none"} {
		if suffix, ok := config.ThinkingSuffixes[level]; ok && suffix != "" && strings.Contains(model, suffix) {
			return level
		}
	}

	for level, suffix := range config.ThinkingSuffixes {
		if suffix != "" && strings.Contains(model, suffix) {
			return level
		}
	}
	return "auto" // default
}

// ParseModelFeatures extracts all features from a model name
func ParseModelFeatures(model string) ModelFeatures {
	return ParseModelFeaturesWithConfig(model, DefaultVariantConfig())
}

func ParseModelFeaturesWithConfig(model string, config *VariantConfig) ModelFeatures {
	if config == nil {
		config = DefaultVariantConfig()
	}

	return ModelFeatures{
		Base:           BaseFromFeatureWithConfig(model, config),
		FakeStreaming:  IsFakeStreamingWithConfig(model, config),
		AntiTruncation: IsAntiTruncationWithConfig(model, config),
		Search:         IsSearchWithConfig(model, config),
		ThinkingLevel:  GetThinkingLevelWithConfig(model, config),
	}
}

// ModelFeatures represents parsed model features
type ModelFeatures struct {
	Base           string `json:"base"`
	FakeStreaming  bool   `json:"fake_streaming"`
	AntiTruncation bool   `json:"anti_truncation"`
	Search         bool   `json:"search"`
	ThinkingLevel  string `json:"thinking_level"`
}
