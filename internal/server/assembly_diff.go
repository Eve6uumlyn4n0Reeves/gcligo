package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gcli2api-go/internal/models"
	store "gcli2api-go/internal/storage"
)

// PlanDiff summarizes differences between current registry and a plan.
type PlanDiff struct {
	OpenAI struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	} `json:"openai"`
	Gemini struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	} `json:"gemini"`
	VariantChanged bool `json:"variant_changed"`
}

func (s *AssemblyService) DiffApply(ctx context.Context, name string) (*PlanDiff, error) {
	plan, err := s.GetPlan(ctx, name)
	if err != nil {
		return nil, err
	}
	curOA := models.ExposedModelIDsByChannel(s.cfg, s.st, "openai")
	curGM := models.ExposedModelIDsByChannel(s.cfg, s.st, "gemini")
	wantOA, wantGM := idsFromPlan(plan)
	return diffIDs(curOA, wantOA, curGM, wantGM, s.st, plan), nil
}

func (s *AssemblyService) DiffRollback(ctx context.Context, name string) (*PlanDiff, error) {
	if s.st == nil {
		return nil, &store.ErrNotSupported{Operation: "rollback diff"}
	}
	key := "assembly_plan_backup:" + sanitizePlanName(name)
	v, err := s.st.GetConfig(ctx, key)
	if err != nil {
		return nil, err
	}
	plan, _ := v.(map[string]any)
	curOA := models.ExposedModelIDsByChannel(s.cfg, s.st, "openai")
	curGM := models.ExposedModelIDsByChannel(s.cfg, s.st, "gemini")
	wantOA, wantGM := idsFromBackup(plan)
	return diffIDs(curOA, wantOA, curGM, wantGM, s.st, plan), nil
}

func (s *AssemblyService) DiffPlan(ctx context.Context, plan map[string]any) (*PlanDiff, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan payload required")
	}
	sanitized := sanitizePlanPayload(plan)
	curOA := models.ExposedModelIDsByChannel(s.cfg, s.st, "openai")
	curGM := models.ExposedModelIDsByChannel(s.cfg, s.st, "gemini")
	wantOA, wantGM := idsFromPlan(sanitized)
	return diffIDs(curOA, wantOA, curGM, wantGM, s.st, sanitized), nil
}

func idsFromPlan(plan map[string]any) ([]string, []string) {
	var oa, gm []string
	if ms, ok := plan["models"].(map[string]any); ok {
		if arr, ok2 := ms["openai"].([]any); ok2 {
			oa = collectIDs(arr)
		} else if typed, ok2 := ms["openai"].([]map[string]any); ok2 {
			oa = collectIDs(fromMapSlice(typed))
		} else if typed, ok2 := ms["openai"].([]models.RegistryEntry); ok2 {
			oa = collectRegistryIDs(typed)
		}
		if arr, ok2 := ms["gemini"].([]any); ok2 {
			gm = collectIDs(arr)
		} else if typed, ok2 := ms["gemini"].([]map[string]any); ok2 {
			gm = collectIDs(fromMapSlice(typed))
		} else if typed, ok2 := ms["gemini"].([]models.RegistryEntry); ok2 {
			gm = collectRegistryIDs(typed)
		}
	}
	return oa, gm
}

func idsFromBackup(plan map[string]any) ([]string, []string) {
	var oa, gm []string
	if arr, ok := plan["models_openai"].([]any); ok {
		oa = collectIDs(arr)
	} else if typed, ok := plan["models_openai"].([]map[string]any); ok {
		oa = collectIDs(fromMapSlice(typed))
	} else if typed, ok := plan["models_openai"].([]models.RegistryEntry); ok {
		oa = collectRegistryIDs(typed)
	}
	if arr, ok := plan["models_gemini"].([]any); ok {
		gm = collectIDs(arr)
	} else if typed, ok := plan["models_gemini"].([]map[string]any); ok {
		gm = collectIDs(fromMapSlice(typed))
	} else if typed, ok := plan["models_gemini"].([]models.RegistryEntry); ok {
		gm = collectRegistryIDs(typed)
	}
	return oa, gm
}

func sanitizePlanPayload(plan map[string]any) map[string]any {
	if plan == nil {
		return map[string]any{}
	}
	sanitized := make(map[string]any)
	if modelsAny, ok := plan["models"].(map[string]any); ok {
		sanitized["models"] = modelsAny
	}
	if vc, ok := plan["variant_config"].(map[string]any); ok {
		sanitized["variant_config"] = vc
	}
	return sanitized
}

func collectIDs(arr []any) []string {
	out := make([]string, 0, len(arr))
	for _, it := range arr {
		if m, ok := it.(map[string]any); ok {
			if id, _ := m["id"].(string); id != "" {
				out = append(out, id)
			}
			if id, _ := m["ID"].(string); id != "" {
				out = append(out, id)
			}
		} else if str, ok := it.(string); ok && strings.TrimSpace(str) != "" {
			out = append(out, strings.TrimSpace(str))
		}
	}
	return out
}

func fromMapSlice(src []map[string]any) []any {
	out := make([]any, 0, len(src))
	for _, item := range src {
		out = append(out, item)
	}
	return out
}

func collectRegistryIDs(src []models.RegistryEntry) []string {
	out := make([]string, 0, len(src))
	for _, entry := range src {
		id := entry.ID
		if strings.TrimSpace(id) == "" {
			id = models.BuildVariantID(entry.Base, entry.FakeStreaming, entry.AntiTrunc, entry.Thinking, entry.Search)
		}
		if strings.TrimSpace(id) != "" {
			out = append(out, strings.TrimSpace(id))
		}
	}
	return out
}

func diffIDs(curOA, wantOA, curGM, wantGM []string, st store.Backend, plan map[string]any) *PlanDiff {
	res := &PlanDiff{}
	res.OpenAI.Add, res.OpenAI.Remove = setDiff(curOA, wantOA)
	res.Gemini.Add, res.Gemini.Remove = setDiff(curGM, wantGM)
	// variant_changed
	if st != nil {
		if v, err := st.GetConfig(context.Background(), "model_variant_config"); err == nil && v != nil {
			cur := stringify(v)
			want := stringify(plan["variant_config"])
			res.VariantChanged = (cur != want)
		}
	}
	return res
}

func setDiff(cur, want []string) (add, remove []string) {
	mcur := map[string]struct{}{}
	for _, s := range cur {
		mcur[s] = struct{}{}
	}
	mw := map[string]struct{}{}
	for _, s := range want {
		mw[s] = struct{}{}
	}
	for s := range mw {
		if _, ok := mcur[s]; !ok {
			add = append(add, s)
		}
	}
	for s := range mcur {
		if _, ok := mw[s]; !ok {
			remove = append(remove, s)
		}
	}
	return
}

func stringify(v any) string { b, _ := json.Marshal(v); return string(b) }
