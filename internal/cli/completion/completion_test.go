package completion_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/cli/completion"
)

func TestCompletionCommandGeneratesScriptsForSupportedShells(t *testing.T) {
	tests := []struct {
		name     string
		shell    string
		expected string
	}{
		{
			name:     "bash",
			shell:    "bash",
			expected: "bash completion",
		},
		{
			name:     "zsh",
			shell:    "zsh",
			expected: "zsh completion",
		},
		{
			name:     "fish",
			shell:    "fish",
			expected: "fish completion",
		},
		{
			name:     "powershell",
			shell:    "powershell",
			expected: "powershell completion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &cobra.Command{Use: "gitdex"}
			cmd := completion.NewCommand(root)
			var out bytes.Buffer

			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{tt.shell})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}

			if !strings.Contains(out.String(), tt.expected) {
				t.Fatalf("expected %q output marker, got %q", tt.expected, out.String())
			}
		})
	}
}

func TestCompletionCommandRejectsUnsupportedShell(t *testing.T) {
	root := &cobra.Command{Use: "gitdex"}
	cmd := completion.NewCommand(root)
	var out bytes.Buffer

	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"invalid-shell"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported shell")
	}

	if !strings.Contains(err.Error(), "unsupported shell") && !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletionCommandRequiresShellArgument(t *testing.T) {
	root := &cobra.Command{Use: "gitdex"}
	cmd := completion.NewCommand(root)
	var out bytes.Buffer

	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected argument validation error")
	}

	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Fatalf("unexpected error: %v", err)
	}
}
