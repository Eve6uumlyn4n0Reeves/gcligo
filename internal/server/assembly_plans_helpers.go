package server

import (
	"encoding/json"
	"fmt"

	"gcli2api-go/internal/models"
)

func coerceRegistryEntries(v any) ([]models.RegistryEntry, error) {
	if v == nil {
		return nil, nil
	}
	switch typed := v.(type) {
	case []models.RegistryEntry:
		return typed, nil
	case []map[string]any:
		return mapEntriesToRegistry(typed)
	case []any:
		b, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		var entries []models.RegistryEntry
		if err := json.Unmarshal(b, &entries); err != nil {
			return nil, err
		}
		return entries, nil
	default:
		return nil, fmt.Errorf("unsupported registry payload type %T", v)
	}
}

func mapEntriesToRegistry(src []map[string]any) ([]models.RegistryEntry, error) {
	b, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var entries []models.RegistryEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

type configUpdate struct {
	key   string
	value interface{}
}

type priorSnapshot struct {
	key    string
	value  interface{}
	exists bool
}

func updatesFromPlan(plan map[string]any) ([]configUpdate, error) {
	var updates []configUpdate
	if modelsAny, ok := plan["models"].(map[string]any); ok {
		if entries, err := coerceRegistryEntries(modelsAny["openai"]); err != nil {
			return nil, err
		} else if entries != nil {
			updates = append(updates, configUpdate{key: "model_registry_openai", value: entries})
		}
		if entries, err := coerceRegistryEntries(modelsAny["gemini"]); err != nil {
			return nil, err
		} else if entries != nil {
			updates = append(updates, configUpdate{key: "model_registry_gemini", value: entries})
		}
	}
	if vc, ok := plan["variant_config"].(map[string]any); ok {
		updates = append(updates, configUpdate{key: "model_variant_config", value: vc})
	}
	return updates, nil
}

func updatesFromBackup(plan map[string]any) ([]configUpdate, error) {
	var updates []configUpdate
	if entries, err := coerceRegistryEntries(plan["models_openai"]); err != nil {
		return nil, err
	} else if entries != nil {
		updates = append(updates, configUpdate{key: "model_registry_openai", value: entries})
	}
	if entries, err := coerceRegistryEntries(plan["models_gemini"]); err != nil {
		return nil, err
	} else if entries != nil {
		updates = append(updates, configUpdate{key: "model_registry_gemini", value: entries})
	}
	if vc, ok := plan["variant_config"].(map[string]any); ok {
		updates = append(updates, configUpdate{key: "model_variant_config", value: vc})
	}
	return updates, nil
}
