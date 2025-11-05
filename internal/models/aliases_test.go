package models

import "testing"

func TestResolveAlias_NanoBanana(t *testing.T) {
	cases := []string{
		"nano-banana",
		"nano-banana-lite",
		"nanobanana",
	}
	for _, in := range cases {
		got, ok := ResolveAlias(in)
		if !ok {
			t.Fatalf("expected alias for %q", in)
		}
		if got != nanoBananaAliasTarget {
			t.Fatalf("alias mismatch: %q -> %q", in, got)
		}
	}
}

func TestResolveAlias_NoAlias(t *testing.T) {
	in := "gemini-2.5-pro"
	got, ok := ResolveAlias(in)
	if ok {
		t.Fatalf("unexpected alias for %q", in)
	}
	if got != in {
		t.Fatalf("identity expected, got %q", got)
	}
}
