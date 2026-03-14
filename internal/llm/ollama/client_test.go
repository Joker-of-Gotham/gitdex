package ollama

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/llm"
)

func TestParseContextLengthFromModelfile(t *testing.T) {
	if got := parseContextLengthFromModelfile("temperature 0.1\nnum_ctx 16384\n"); got != 16384 {
		t.Fatalf("unexpected context length: %d", got)
	}
}

func TestResolveModelPrefersSecondary(t *testing.T) {
	client := NewClient("http://localhost:11434", "primary")
	client.SetModelForRole(llm.RoleSecondary, "secondary")
	if got := client.resolveModel(llm.GenerateRequest{Role: llm.RoleSecondary}); got != "secondary" {
		t.Fatalf("unexpected model: %s", got)
	}
}
