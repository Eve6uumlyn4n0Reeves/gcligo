package server

import (
	"context"

	"gcli2api-go/internal/models"
)

// Snapshot returns current exposed models and variant config.
func (s *AssemblyService) Snapshot(ctx context.Context) map[string]any {
	oa := models.ActiveEntriesByChannel(s.cfg, s.st, "openai")
	gm := models.ActiveEntriesByChannel(s.cfg, s.st, "gemini")
	out := map[string]any{
		"models": map[string]any{
			"openai": oa,
			"gemini": gm,
		},
	}
	if s.st != nil {
		if v, err := s.st.GetConfig(ctx, "model_variant_config"); err == nil && v != nil {
			if m, ok := v.(map[string]any); ok {
				out["variant_config"] = m
			}
		}
	}
	return out
}
