package integration

import (
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestExportCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	expCmd, _, err := root.Find([]string{"export"})
	if err != nil {
		t.Fatalf("export command not found: %v", err)
	}

	subs := []string{"generate", "list"}
	for _, name := range subs {
		found := false
		for _, c := range expCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'export'", name)
		}
	}
}

func TestExportGenerateRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"export", "generate", "--type", "plan_report"})
	if err := root.Execute(); err != nil {
		t.Fatalf("export generate failed: %v", err)
	}
}

func TestExportListRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"export", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("export list failed: %v", err)
	}
}

func TestExportGenerateWithFormat(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"export", "generate", "--type", "audit_report", "--format", "yaml"})
	if err := root.Execute(); err != nil {
		t.Fatalf("export generate with format failed: %v", err)
	}
}
