package cli

import (
	"strings"
	"testing"
)

func TestNewCLIExecutorAndVersion(t *testing.T) {
	exec, err := NewCLIExecutor()
	if err != nil {
		t.Fatalf("expected git executor: %v", err)
	}
	version, err := exec.Version()
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}
	if !strings.Contains(strings.ToLower(version), "git version") {
		t.Fatalf("unexpected version output: %s", version)
	}
}
