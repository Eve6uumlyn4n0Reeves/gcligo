package openai

import "testing"

func TestValidateAndNormalizeOpenAI_Chat_Minimal(t *testing.T) {
	raw := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": map[string]any{"type": "text", "text": "hi"}},
		},
	}
	out, status, msg := validateAndNormalizeOpenAI(raw, true)
	if status != 0 {
		t.Fatalf("unexpected invalid: %d %s", status, msg)
	}
	if out["model"].(string) == "" {
		t.Fatalf("model should be defaulted")
	}
	msgs := out["messages"].([]any)
	content, _ := msgs[0].(map[string]any)["content"].(string)
	if content != "hi" {
		t.Fatalf("content not normalized: %v", msgs[0])
	}
}

func TestValidateAndNormalizeOpenAI_Chat_MessagesRequired(t *testing.T) {
	_, status, _ := validateAndNormalizeOpenAI(map[string]any{}, true)
	if status == 0 {
		t.Fatalf("expected invalid when messages missing")
	}
}

func TestValidateAndNormalizeOpenAI_Completions_PromptRequired(t *testing.T) {
	_, status, _ := validateAndNormalizeOpenAI(map[string]any{}, false)
	if status == 0 {
		t.Fatalf("expected invalid when prompt missing")
	}
}
