package parser

import "testing"

func TestParseSubmodules(t *testing.T) {
	out := ParseSubmodules("+cb5918aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa libs/foo (v1.2.3)")
	if len(out) != 1 {
		t.Fatalf("expected one submodule, got %d", len(out))
	}
	if out[0].Name != "foo" || out[0].Status != "different commit" {
		t.Fatalf("unexpected submodule parse: %+v", out[0])
	}
}
