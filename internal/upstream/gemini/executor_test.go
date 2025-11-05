package gemini

import (
	"gcli2api-go/internal/config"
	"testing"
)

func TestExecutorPreparePayload_ImageAndThinking(t *testing.T) {
	e := NewExecutor(&config.Config{})
	// minimal payload with empty generationConfig
	raw := []byte(`{"generationConfig":{}}`)
	out := e.preparePayload("gemini-2.5-flash-image-preview", raw)
	if string(out) == string(raw) {
		t.Fatalf("expected payload to be modified for image hints")
	}
}
