package models

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gcli2api-go/internal/storage"
)

const capabilitiesConfigKey = "model_capabilities"

// Capability describes known model abilities used by UI和列表输出
type Capability struct {
	Modalities    []string `json:"modalities,omitempty"` // e.g., ["text"], ["image","text"]
	ContextLength int      `json:"context_length,omitempty"`
	Images        bool     `json:"images,omitempty"`
	Thinking      string   `json:"thinking,omitempty"` // none/auto/max
	// 审计字段（只读）：由服务端在写入时填充
	Source    string `json:"source,omitempty"`     // manual|upstream|probe
	UpdatedAt int64  `json:"updated_at,omitempty"` // unix seconds
}

// GetCapability returns a capability record for a given base or id from storage if available.
func GetCapability(st storage.Backend, idOrBase string) (Capability, bool) {
	var zero Capability
	if st == nil || strings.TrimSpace(idOrBase) == "" {
		return zero, false
	}
	v, err := st.GetConfig(context.Background(), capabilitiesConfigKey)
	if err != nil || v == nil {
		return zero, false
	}
	b, _ := json.Marshal(v)
	var m map[string]Capability
	if json.Unmarshal(b, &m) != nil || len(m) == 0 {
		return zero, false
	}
	id := strings.ToLower(strings.TrimSpace(idOrBase))
	if cap, ok := m[id]; ok {
		return cap, true
	}
	// try base extracted from feature id
	base := strings.ToLower(BaseFromFeature(idOrBase))
	if cap, ok := m[base]; ok {
		return cap, true
	}
	return zero, false
}

// UpsertCapabilities stores/merges capability map to storage (admin API使用)
func UpsertCapabilities(st storage.Backend, caps map[string]Capability) error {
	return UpsertCapabilitiesWithSource(st, caps, "manual")
}

// UpsertCapabilitiesWithSource merges capabilities and stamps source/updated_at.
func UpsertCapabilitiesWithSource(st storage.Backend, caps map[string]Capability, source string) error {
	if st == nil || len(caps) == 0 {
		return nil
	}
	// merge with existing
	existing := map[string]Capability{}
	if v, err := st.GetConfig(context.Background(), capabilitiesConfigKey); err == nil && v != nil {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &existing)
	}
	// normalize source
	src := strings.ToLower(strings.TrimSpace(source))
	if src == "" {
		src = "manual"
	}
	now := time.Now().Unix()
	for k, v := range caps {
		key := strings.ToLower(strings.TrimSpace(k))
		v.Source = src
		v.UpdatedAt = now
		existing[key] = v
	}
	return st.SetConfig(context.Background(), capabilitiesConfigKey, existing)
}

// DefaultCapabilities builds a coarse capability map from base descriptors.
func DefaultCapabilities() map[string]Capability {
	out := make(map[string]Capability)
	now := time.Now().Unix()
	for _, base := range DefaultBaseModels() {
		b := BaseFromFeature(base)
		if _, ok := out[b]; ok {
			continue
		}
		desc := DescribeBase(b)
		mods := []string{"text"}
		if desc.SupportsImage {
			mods = []string{"text", "image"}
		}
		think := desc.SuggestedThinking
		if think == "" {
			think = "auto"
		}
		out[b] = Capability{Modalities: mods, ContextLength: 1000000, Images: desc.SupportsImage, Thinking: think, Source: "upstream", UpdatedAt: now}
	}
	return out
}
