package gemini

import (
	"gcli2api-go/internal/models"
	"testing"
)

func TestFallbackBasesOrder(t *testing.T) {
	cases := map[string][]string{
		"gemini-2.5-pro":         {"gemini-2.5-pro", "gemini-2.5-pro-preview-06-05", "gemini-2.5-pro-preview-05-06", "gemini-2.5-flash"},
		"gemini-2.5-flash-image": {"gemini-2.5-flash-image", "gemini-2.5-flash-image-preview"},
		"gemini-2.5-flash":       {"gemini-2.5-flash", "gemini-2.5-flash-preview-09-2025"},
	}
	for base, want := range cases {
		got := models.FallbackBases(base)
		if len(got) < len(want) {
			t.Fatalf("%s fallback too short: %v", base, got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s[%d]: want %s, got %s", base, i, want[i], got[i])
			}
		}
	}
}
