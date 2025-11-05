package openai

import (
	"encoding/json"
	"strings"
)

// jsonString converts a value to string, trimming empty strings and encoding objects.
func jsonString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return ""
		}
		return s
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
