package server

import (
	"context"
	"strings"

	store "gcli2api-go/internal/storage"
)

func (s *AssemblyService) ListPlans(ctx context.Context) ([]map[string]any, error) {
	if s.st == nil {
		return []map[string]any{}, nil
	}
	cfgs, err := s.st.ListConfigs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0)
	for k, v := range cfgs {
		if !strings.HasPrefix(k, "assembly_plan:") {
			continue
		}
		name := strings.TrimPrefix(k, "assembly_plan:")
		item := map[string]any{"name": name}
		if mv, ok := v.(map[string]any); ok {
			item["plan"] = mv
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *AssemblyService) GetPlan(ctx context.Context, name string) (map[string]any, error) {
	if s.st == nil {
		return nil, &store.ErrNotSupported{Operation: "get plan"}
	}
	key := "assembly_plan:" + sanitizePlanName(name)
	v, err := s.st.GetConfig(ctx, key)
	if err != nil {
		return nil, err
	}
	if m, ok := v.(map[string]any); ok {
		return m, nil
	}
	return map[string]any{"raw": v}, nil
}

func (s *AssemblyService) SavePlan(ctx context.Context, name string, include map[string]bool) (map[string]any, error) {
	if s.st == nil {
		return nil, &store.ErrNotSupported{Operation: "save plan"}
	}
	snap := s.Snapshot(ctx)
	plan := map[string]any{"name": sanitizePlanName(name)}
	if include == nil || include["models"] {
		plan["models"] = snap["models"]
	}
	if include == nil || include["variants"] {
		if vc, ok := snap["variant_config"]; ok {
			plan["variant_config"] = vc
		}
	}
	if err := s.st.SetConfig(ctx, "assembly_plan:"+sanitizePlanName(name), plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *AssemblyService) DeletePlan(ctx context.Context, name string) error {
	if s.st == nil {
		return &store.ErrNotSupported{Operation: "delete plan"}
	}
	return s.st.DeleteConfig(ctx, "assembly_plan:"+sanitizePlanName(name))
}
