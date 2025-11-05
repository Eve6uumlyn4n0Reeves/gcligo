package gemini

import (
	"testing"
)

func TestGeminiModelDisallowsThinking(t *testing.T) {
	cases := map[string]bool{
		"gemini-2.5-flash-image":         true,
		"gemini-2.5-flash-image-preview": true,
		"gemini-2.5-pro":                 false,
		"gemini-2.5-flash":               false,
	}
	for m, want := range cases {
		if got := geminiModelDisallowsThinking(m); got != want {
			t.Fatalf("model %s: want %v, got %v", m, want, got)
		}
	}
}

func TestDeleteJSONField(t *testing.T) {
	payload := []byte(`{"generationConfig":{"thinkingConfig":{"thinkingBudget":1024}}}`)
	out := deleteJSONField(payload, "generationConfig.thinkingConfig")
	if string(out) == string(payload) {
		t.Fatalf("field was not deleted")
	}
}
