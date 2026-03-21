package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestTriggerCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	triggerCmd, _, err := root.Find([]string{"trigger"})
	if err != nil {
		t.Fatalf("trigger command not found: %v", err)
	}

	subs := []string{"add", "list", "enable", "disable", "events", "fire"}
	for _, name := range subs {
		found := false
		for _, c := range triggerCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'trigger'", name)
		}
	}
}

func TestTriggerCommandHelp(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"trigger", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("trigger --help failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "add") {
		t.Error("help should mention add subcommand")
	}
	if !strings.Contains(output, "list") {
		t.Error("help should mention list subcommand")
	}
	if !strings.Contains(output, "events") {
		t.Error("help should mention events subcommand")
	}
	if !strings.Contains(output, "fire") {
		t.Error("help should mention fire subcommand")
	}
}

func TestTriggerAddRuns(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"trigger", "add", "--type", "schedule", "--name", "nightly-sync", "--pattern", "0 0 * * *", "--action", "repo sync"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("trigger add failed: %v", err)
	}
	if !strings.Contains(out.String(), "nightly-sync") && !strings.Contains(out.String(), "tr_") {
		t.Error("output should contain trigger name or id")
	}
}

func TestTriggerListEmpty(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"trigger", "list"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("trigger list failed: %v", err)
	}
}
