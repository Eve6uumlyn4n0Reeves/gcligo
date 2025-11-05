package models

import (
	"context"
	"encoding/json"
	"strings"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/storage"
)

// RegistryEntry describes a model exposure entry managed by admin UI.
type RegistryEntry struct {
	ID            string `json:"id"`   // final exposed id (with prefix/suffix)
	Base          string `json:"base"` // base model, e.g., gemini-2.5-pro
	FakeStreaming bool   `json:"fake_streaming"`
	AntiTrunc     bool   `json:"anti_truncation"`
	Thinking      string `json:"thinking"` // "auto","none","low","medium","high","max"
	Search        bool   `json:"search"`
	Image         bool   `json:"image"`  // hint for UI; id still determines behavior
	Stream        bool   `json:"stream"` // prefer streaming when available
	Enabled       bool   `json:"enabled"`
	Upstream      string `json:"upstream"`        // expected "code_assist"
	Group         string `json:"group,omitempty"` // optional group id/name
	// 可选：显示禁用原因（由管理端叠加，不参与路由逻辑）
	DisabledReason string `json:"disabled_reason,omitempty"`
}

const registryConfigKey = "model_registry" // legacy
const registryOpenAIKey = "model_registry_openai"
const registryGeminiKey = "model_registry_gemini"
const groupsConfigKey = "model_groups"

// GroupEntry represents a simple group for organizing models in UI.
type GroupEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Order       int    `json:"order"`
	Enabled     bool   `json:"enabled"`
}

// GroupsConfigKey returns the storage key for groups
func GroupsConfigKey() string { return groupsConfigKey }

// BuildVariantID composes the exposed model id from options using existing naming scheme.
func BuildVariantID(base string, fake, anti bool, thinking string, search bool) string {
	id := base
	// suffixes
	switch strings.ToLower(strings.TrimSpace(thinking)) {
	case "max", "high":
		id += "-maxthinking"
	case "none":
		id += "-nothinking"
	}
	if search {
		id += "-search"
	}
	// prefixes
	if fake {
		id = "假流式/" + id
	}
	if anti {
		id = "流式抗截断/" + id
	}
	return id
}

// ExposedModelIDs returns the list of models to expose to /v1/models.
// It tries reading the dynamic registry from storage; falls back to AllVariants with disabled filters.
func ExposedModelIDs(cfg *config.Config, st storage.Backend) []string {
	// default to OpenAI channel for backward compatibility
	return ExposedModelIDsByChannel(cfg, st, "openai")
}

// ExposedModelIDsByChannel returns the list of models for a specific channel ("openai" or "gemini").
func ExposedModelIDsByChannel(cfg *config.Config, st storage.Backend, channel string) []string {
	key := registryOpenAIKey
	if strings.ToLower(channel) == "gemini" {
		key = registryGeminiKey
	}
	// Try registry from storage
	if st != nil {
		// prefer channel-specific key
		if v, err := st.GetConfig(context.Background(), key); err == nil && v != nil {
			// Expect a JSON-serializable array of entries
			b, _ := json.Marshal(v)
			var entries []RegistryEntry
			if json.Unmarshal(b, &entries) == nil {
				out := make([]string, 0, len(entries))
				for _, e := range entries {
					if !e.Enabled {
						continue
					}
					id := e.ID
					if strings.TrimSpace(id) == "" {
						id = BuildVariantID(e.Base, e.FakeStreaming, e.AntiTrunc, e.Thinking, e.Search)
					}
					out = append(out, id)
				}
				if len(out) > 0 {
					return filterDisabled(out, cfg.DisabledModels)
				}
			}
		}
		// fallback to legacy common key when channel-specific not set
		if v, err := st.GetConfig(context.Background(), registryConfigKey); err == nil && v != nil {
			b, _ := json.Marshal(v)
			var entries []RegistryEntry
			if json.Unmarshal(b, &entries) == nil {
				out := make([]string, 0, len(entries))
				for _, e := range entries {
					if !e.Enabled {
						continue
					}
					id := e.ID
					if strings.TrimSpace(id) == "" {
						id = BuildVariantID(e.Base, e.FakeStreaming, e.AntiTrunc, e.Thinking, e.Search)
					}
					out = append(out, id)
				}
				if len(out) > 0 {
					return filterDisabled(out, cfg.DisabledModels)
				}
			}
		}
	}
	// Fallback to curated defaults
	defs := DefaultRegistry()
	out := make([]string, 0, len(defs))
	for _, e := range defs {
		id := e.ID
		if strings.TrimSpace(id) == "" {
			id = BuildVariantID(e.Base, e.FakeStreaming, e.AntiTrunc, e.Thinking, e.Search)
		}
		out = append(out, id)
	}
	return filterDisabled(out, cfg.DisabledModels)
}

// ActiveEntriesByChannel returns enabled registry entries with computed IDs for a channel.
func ActiveEntriesByChannel(cfg *config.Config, st storage.Backend, channel string) []RegistryEntry {
	key := registryOpenAIKey
	if strings.ToLower(channel) == "gemini" {
		key = registryGeminiKey
	}
	var entries []RegistryEntry
	if st != nil {
		if v, err := st.GetConfig(context.Background(), key); err == nil && v != nil {
			b, _ := json.Marshal(v)
			_ = json.Unmarshal(b, &entries)
		} else if v, err := st.GetConfig(context.Background(), registryConfigKey); err == nil && v != nil {
			b, _ := json.Marshal(v)
			_ = json.Unmarshal(b, &entries)
		}
	}
	if len(entries) == 0 {
		entries = DefaultRegistry()
	}
	out := make([]RegistryEntry, 0, len(entries))
	for _, e := range entries {
		if !e.Enabled {
			continue
		}
		id := strings.TrimSpace(e.ID)
		if id == "" {
			id = BuildVariantID(e.Base, e.FakeStreaming, e.AntiTrunc, e.Thinking, e.Search)
		}
		if strings.TrimSpace(e.Base) == "" {
			e.Base = BaseFromFeature(id)
		}
		e.ID = id
		out = append(out, e)
	}
	return out
}

func filterDisabled(ids []string, disabled []string) []string {
	if len(disabled) == 0 {
		return ids
	}
	off := map[string]struct{}{}
	for _, d := range disabled {
		if d != "" {
			off[d] = struct{}{}
		}
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := off[id]; ok {
			continue
		}
		out = append(out, id)
	}
	return out
}

// DefaultRegistry returns a small curated set of sensible defaults.
func DefaultRegistry() []RegistryEntry {
	return []RegistryEntry{
		{Base: "gemini-2.5-pro", Thinking: "auto", Stream: true, Enabled: true, Upstream: "code_assist"},
		{Base: "gemini-2.5-pro", AntiTrunc: true, Thinking: "auto", Stream: true, Enabled: true, Upstream: "code_assist"},
		{Base: "gemini-2.5-pro", Thinking: "max", Stream: true, Enabled: true, Upstream: "code_assist"},
		{Base: "gemini-2.5-flash", Thinking: "auto", Stream: true, Enabled: true, Upstream: "code_assist"},
		{Base: "gemini-2.5-flash-image", Image: true, Stream: false, Enabled: true, Upstream: "code_assist"},
		{Base: "gemini-2.5-flash-image-preview", Image: true, Stream: false, Enabled: true, Upstream: "code_assist"},
	}
}
