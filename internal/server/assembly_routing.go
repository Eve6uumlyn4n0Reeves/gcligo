package server

import (
	"context"
	"time"

	store "gcli2api-go/internal/storage"
)

// SaveRoutingState persists strategy cooldown state if storage/backend are available.
func (s *AssemblyService) SaveRoutingState(ctx context.Context) (int, error) {
	st := s.strategy
	if st == nil || s.st == nil {
		return 0, &store.ErrNotSupported{Operation: "routing_state"}
	}
	_, cds := st.Snapshot()
	payload := map[string]any{"cooldowns": cds, "saved_at": time.Now().Format(time.RFC3339Nano)}
	if err := s.st.SetConfig(ctx, "routing_state", payload); err != nil {
		return 0, err
	}
	return len(cds), nil
}

// RestoreRoutingState reloads persisted cooldown entries into the routing strategy.
func (s *AssemblyService) RestoreRoutingState(ctx context.Context) (int, error) {
	st := s.strategy
	if st == nil || s.st == nil {
		return 0, &store.ErrNotSupported{Operation: "routing_state"}
	}
	v, err := s.st.GetConfig(ctx, "routing_state")
	if err != nil || v == nil {
		return 0, err
	}
	raw, ok := v.(map[string]any)
	if !ok {
		return 0, &store.ErrNotSupported{Operation: "routing_state_format"}
	}
	arr, _ := raw["cooldowns"].([]any)
	_, existing := st.Snapshot()
	for _, cd := range existing {
		st.ClearCooldown(cd.CredID)
	}
	applied := 0
	for _, it := range arr {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		id, _ := m["credential_id"].(string)
		strikes := int(toInt64(m["strikes"]))
		if id != "" && strikes > 0 {
			st.SetCooldown(id, strikes, time.Now().Add(5*time.Second))
			applied++
		}
	}
	return applied, nil
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	case string:
		if t == "" {
			return 0
		}
		var out int64
		for _, r := range t {
			if r < '0' || r > '9' {
				return 0
			}
		}
		for i := 0; i < len(t); i++ {
			out = out*10 + int64(t[i]-'0')
		}
		return out
	default:
		return 0
	}
}
