package integration

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestAutonomyCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	autonomyCmd, _, err := root.Find([]string{"autonomy"})
	if err != nil {
		t.Fatalf("autonomy command not found: %v", err)
	}

	subs := []string{"show", "list", "set", "run-once"}
	for _, name := range subs {
		found := false
		for _, c := range autonomyCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'autonomy'", name)
		}
	}
}

func TestAutonomyCommandHelp(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"autonomy", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("autonomy --help failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "show") {
		t.Error("help should mention show subcommand")
	}
	if !strings.Contains(output, "list") {
		t.Error("help should mention list subcommand")
	}
	if !strings.Contains(output, "set") {
		t.Error("help should mention set subcommand")
	}
	if !strings.Contains(output, "run-once") {
		t.Error("help should mention run-once subcommand")
	}
}

func TestAutonomyShowNoConfig(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"autonomy", "show"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("autonomy show failed: %v", err)
	}
	if !strings.Contains(out.String(), "No") && !strings.Contains(out.String(), "no") {
		t.Error("output should indicate no config when none configured")
	}
}

func TestAutonomySetRequiresCapability(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"autonomy", "set", "--level", "supervised"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when set without --capability")
	}
}

func TestAutonomySetRequiresLevel(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"autonomy", "set", "--capability", "repo_sync"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when set without --level")
	}
}

func TestAutonomyShowJSON(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"autonomy", "show", "--output", "json"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("autonomy show --output json failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Errorf("output should be valid JSON: %v", err)
	}
}
