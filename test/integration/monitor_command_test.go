package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/cli/command"
)

func TestMonitorCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	monCmd, _, err := root.Find([]string{"monitor"})
	if err != nil {
		t.Fatalf("monitor command not found: %v", err)
	}

	subs := []string{"add", "list", "events", "remove", "check"}
	for _, name := range subs {
		found := false
		for _, c := range monCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'monitor'", name)
		}
	}
}

func TestMonitorCommandHelp(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"monitor", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("monitor --help failed: %v", err)
	}
	output := out.String()
	for _, s := range []string{"add", "list", "events", "remove", "check"} {
		if !strings.Contains(output, s) {
			t.Errorf("help should mention %q", s)
		}
	}
}

func TestMonitorAddRequiresRepo(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"monitor", "add"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when add without --repo")
	}
}

func TestMonitorAddAndList(t *testing.T) {
	store := autonomy.NewMemoryMonitorStore()
	restore := command.SetMonitorStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"monitor", "add", "--repo", "owner/repo", "--interval", "5m"})

	if err := root.Execute(); err != nil {
		t.Fatalf("monitor add failed: %v", err)
	}
	if !strings.Contains(out.String(), "Monitor added") {
		t.Error("output should indicate monitor added")
	}

	out.Reset()
	root.SetArgs([]string{"monitor", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("monitor list failed: %v", err)
	}
	if !strings.Contains(out.String(), "owner/repo") {
		t.Error("list should show added monitor")
	}
}

func TestMonitorListEmpty(t *testing.T) {
	store := autonomy.NewMemoryMonitorStore()
	restore := command.SetMonitorStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"monitor", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("monitor list failed: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "No monitors") && !strings.Contains(s, "Monitor ID") {
		t.Errorf("output should indicate monitor state (empty or list): %s", s)
	}
}
