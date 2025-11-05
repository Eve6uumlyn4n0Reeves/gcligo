package models

import "strings"

// BaseDescriptor describes upstream base model capabilities and recommended toggles.
type BaseDescriptor struct {
	Base                   string   `json:"base"`
	DisplayName            string   `json:"display_name,omitempty"`
	Family                 string   `json:"family,omitempty"`
	SupportsImage          bool     `json:"supports_image"`
	SupportsStream         bool     `json:"supports_stream"`
	SupportsSearch         bool     `json:"supports_search"`
	SuggestedThinking      string   `json:"suggested_thinking,omitempty"`
	SuggestedFakeStreaming bool     `json:"suggested_fake_stream,omitempty"`
	SuggestedAntiTrunc     bool     `json:"suggested_anti_trunc,omitempty"`
	DefaultEnabled         bool     `json:"default_enabled"`
	Tags                   []string `json:"tags,omitempty"`
	Notes                  string   `json:"notes,omitempty"`
}

var baseDescriptorOverrides = map[string]BaseDescriptor{
	"gemini-2.5-pro": {
		DisplayName:       "Gemini 2.5 Pro",
		Family:            "pro",
		SupportsImage:     false,
		SupportsStream:    true,
		SupportsSearch:    true,
		SuggestedThinking: "max",
		DefaultEnabled:    true,
		Tags:              []string{"高精度", "长上下文"},
	},
	"gemini-2.5-pro-preview-06-05": {
		DisplayName:       "Gemini 2.5 Pro Preview (06-05)",
		Family:            "pro",
		SupportsStream:    true,
		SupportsSearch:    true,
		SuggestedThinking: "max",
		DefaultEnabled:    false,
		Tags:              []string{"Preview", "高精度"},
	},
	"gemini-2.5-flash": {
		DisplayName:        "Gemini 2.5 Flash",
		Family:             "flash",
		SupportsStream:     true,
		SuggestedThinking:  "auto",
		SuggestedAntiTrunc: true,
		DefaultEnabled:     true,
		Tags:               []string{"低延迟"},
	},
	"gemini-2.5-flash-preview-09-2025": {
		DisplayName:        "Gemini 2.5 Flash Preview (09-2025)",
		Family:             "flash",
		SupportsStream:     true,
		SuggestedThinking:  "auto",
		SuggestedAntiTrunc: true,
		DefaultEnabled:     false,
		Tags:               []string{"Preview", "低延迟"},
	},
	"gemini-2.5-flash-image": {
		DisplayName:        "Gemini 2.5 Flash Image",
		Family:             "flash",
		SupportsImage:      true,
		SupportsStream:     true,
		SuggestedThinking:  "auto",
		SuggestedAntiTrunc: true,
		DefaultEnabled:     true,
		Tags:               []string{"多模态", "低延迟"},
	},
	"gemini-2.5-flash-image-preview": {
		DisplayName:        "Gemini 2.5 Flash Image Preview",
		Family:             "flash",
		SupportsImage:      true,
		SupportsStream:     true,
		SuggestedThinking:  "auto",
		SuggestedAntiTrunc: true,
		DefaultEnabled:     false,
		Tags:               []string{"Preview", "多模态"},
	},
}

// DescribeBase returns capability metadata for a base model.
func DescribeBase(base string) BaseDescriptor {
	lower := strings.ToLower(strings.TrimSpace(base))
	desc, ok := baseDescriptorOverrides[lower]
	if !ok {
		desc = heuristicDescriptor(lower)
	}
	if desc.Base == "" {
		desc.Base = lower
	}
	if desc.DisplayName == "" {
		desc.DisplayName = strings.ToUpper(lower)
	}
	return desc
}

func heuristicDescriptor(base string) BaseDescriptor {
	desc := BaseDescriptor{
		Base:              base,
		SupportsStream:    true,
		DefaultEnabled:    true,
		SuggestedThinking: "auto",
	}

	if strings.Contains(base, "pro") {
		desc.Family = "pro"
		desc.SupportsSearch = true
		desc.SuggestedThinking = "max"
		desc.Tags = append(desc.Tags, "高精度")
	}
	if strings.Contains(base, "flash") {
		desc.Family = "flash"
		desc.SuggestedAntiTrunc = true
		desc.Tags = append(desc.Tags, "低延迟")
	}
	if strings.Contains(base, "image") {
		desc.SupportsImage = true
		desc.Tags = append(desc.Tags, "多模态")
	}
	if strings.Contains(base, "preview") {
		desc.DefaultEnabled = false
		desc.Tags = append(desc.Tags, "Preview")
	}
	return desc
}
