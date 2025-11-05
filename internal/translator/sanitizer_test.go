package translator

import (
	"testing"
)

func TestSanitizeText_RemovesAgePattern(t *testing.T) {
	ConfigureSanitizer(true, nil)
	t.Cleanup(func() { ConfigureSanitizer(false, nil) })
	in := "这是一个 16岁 的学生，内容。"
	out := sanitizeText(in)
	if out == in || out == "" {
		t.Fatalf("expected age pattern removed, got: %q", out)
	}
}

func TestEnsureDoneInstruction_OnlyOnce(t *testing.T) {
	var parts []interface{}
	ensureDoneInstruction(&parts)
	if len(parts) == 0 {
		t.Fatal("expected instruction appended")
	}
	// run twice; should not duplicate
	ensureDoneInstruction(&parts)
	if len(parts) != 1 {
		t.Fatalf("expected single instruction, got %d", len(parts))
	}
}
