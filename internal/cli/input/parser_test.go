package input

import (
	"testing"

	"github.com/spf13/cobra"
)

func buildTestCommandTree() *cobra.Command {
	root := &cobra.Command{Use: "gitdex"}

	root.AddCommand(&cobra.Command{Use: "doctor", Short: "Run diagnostics"})
	root.AddCommand(&cobra.Command{Use: "version", Short: "Print version"})
	root.AddCommand(&cobra.Command{Use: "chat", Short: "Chat with Gitdex"})
	root.AddCommand(&cobra.Command{Use: "capabilities", Short: "List capabilities"})

	configCmd := &cobra.Command{Use: "config", Short: "Configuration"}
	configCmd.AddCommand(&cobra.Command{Use: "show", Short: "Show config"})
	root.AddCommand(configCmd)

	repoCmd := &cobra.Command{Use: "repo", Short: "Repository operations"}
	repoCmd.AddCommand(&cobra.Command{Use: "sync", Short: "Sync upstream"})
	root.AddCommand(repoCmd)

	return root
}

func TestClassify_EmptyInput(t *testing.T) {
	p := NewParser(buildTestCommandTree())

	tests := []string{"", "   ", "\t\n"}
	for _, input := range tests {
		result := p.Classify(input)
		if result.Type != InputEmpty {
			t.Errorf("Classify(%q).Type = %v, want InputEmpty", input, result.Type)
		}
	}
}

func TestClassify_KnownCommand(t *testing.T) {
	p := NewParser(buildTestCommandTree())

	tests := []struct {
		input       string
		wantCommand string
		wantArgs    int
	}{
		{"doctor", "gitdex doctor", 0},
		{"version", "gitdex version", 0},
		{"chat", "gitdex chat", 0},
		{"capabilities", "gitdex capabilities", 0},
		{"config show", "gitdex config show", 0},
		{"repo sync", "gitdex repo sync", 0},
	}

	for _, tt := range tests {
		result := p.Classify(tt.input)
		if result.Type != InputCommand {
			t.Errorf("Classify(%q).Type = %v, want InputCommand", tt.input, result.Type)
			continue
		}
		if result.Command != tt.wantCommand {
			t.Errorf("Classify(%q).Command = %q, want %q", tt.input, result.Command, tt.wantCommand)
		}
		if len(result.Args) != tt.wantArgs {
			t.Errorf("Classify(%q).Args len = %d, want %d", tt.input, len(result.Args), tt.wantArgs)
		}
	}
}

func TestClassify_CommandWithArgs(t *testing.T) {
	p := NewParser(buildTestCommandTree())
	result := p.Classify("chat hello world")
	if result.Type != InputCommand {
		t.Fatalf("Classify(chat hello world).Type = %v, want InputCommand", result.Type)
	}
	if result.Command != "gitdex chat" {
		t.Errorf("Command = %q, want 'gitdex chat'", result.Command)
	}
	if len(result.Args) != 2 || result.Args[0] != "hello" || result.Args[1] != "world" {
		t.Errorf("Args = %v, want [hello world]", result.Args)
	}
}

func TestClassify_NaturalLanguage(t *testing.T) {
	p := NewParser(buildTestCommandTree())

	tests := []string{
		"what can you help me with?",
		"tell me about this repository",
		"how do I sync upstream?",
		"please explain the diagnostics",
		"hello",
		"unknown-command --flag",
	}

	for _, input := range tests {
		result := p.Classify(input)
		if result.Type != InputNaturalLanguage {
			t.Errorf("Classify(%q).Type = %v, want InputNaturalLanguage", input, result.Type)
		}
		if result.Raw != input {
			t.Errorf("Classify(%q).Raw = %q", input, result.Raw)
		}
	}
}

func TestClassify_PreservesRaw(t *testing.T) {
	p := NewParser(buildTestCommandTree())
	raw := "  doctor  "
	result := p.Classify(raw)
	if result.Raw != raw {
		t.Errorf("Raw = %q, want %q", result.Raw, raw)
	}
}

func TestClassify_OnlyFlags(t *testing.T) {
	p := NewParser(buildTestCommandTree())
	result := p.Classify("--output json")
	if result.Type != InputNaturalLanguage {
		t.Errorf("flags-only input should classify as natural language, got %v", result.Type)
	}
}

func TestInputType_String(t *testing.T) {
	tests := []struct {
		t    InputType
		want string
	}{
		{InputCommand, "command"},
		{InputNaturalLanguage, "natural_language"},
		{InputEmpty, "empty"},
		{InputType(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("InputType(%d).String() = %q, want %q", tt.t, got, tt.want)
		}
	}
}
