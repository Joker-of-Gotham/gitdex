package app

import "testing"

func TestNew(t *testing.T) {
	a := New(Config{Version: "test"})
	if a == nil {
		t.Fatal("expected application")
	}
	if a.config.Version != "test" {
		t.Fatalf("unexpected version: %s", a.config.Version)
	}
}
