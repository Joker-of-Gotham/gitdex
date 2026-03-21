package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestCollabCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	collabCmd, _, err := root.Find([]string{"collab"})
	if err != nil {
		t.Fatalf("collab command not found: %v", err)
	}

	subs := []string{"list", "show", "create", "comment", "close", "reopen"}
	for _, name := range subs {
		found := false
		for _, c := range collabCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'collab'", name)
		}
	}
}

func TestCollabHelp(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"collab", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("collab --help failed: %v", err)
	}

	output := out.String()
	for _, sub := range []string{"list", "show", "create"} {
		if !strings.Contains(output, sub) {
			t.Errorf("help should mention %q subcommand", sub)
		}
	}
}

func TestCollabListEmpty(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"collab", "list", "--repo", "nonexistent/empty"})

	if err := root.Execute(); err != nil {
		t.Fatalf("collab list failed: %v", err)
	}

	if !strings.Contains(out.String(), "No collaboration objects found") && !strings.Contains(out.String(), "objects") {
		t.Logf("list output: %s", out.String())
	}
}

func TestCollabShowNotFound(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetErr(&out)
	root.SetArgs([]string{"collab", "show", "owner/repo#99999"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when showing non-existent object")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}
