package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestCockpitCommand_Registration(t *testing.T) {
	root := command.NewRootCommand()
	found := false
	for _, cmd := range root.Commands() {
		if cmd.Name() == "cockpit" {
			found = true
			break
		}
	}
	if !found {
		t.Error("cockpit command not registered in root")
	}
}

func TestCockpitCommand_Help(t *testing.T) {
	root := command.NewRootCommand()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"cockpit", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("cockpit --help failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "cockpit") {
		t.Error("help output should mention cockpit")
	}
	if !strings.Contains(output, "--no-tui") {
		t.Error("help output should mention --no-tui flag")
	}
	if !strings.Contains(output, "--repo") {
		t.Error("help output should mention --repo flag")
	}
}

func TestCockpitCommand_NoTUIFlag(t *testing.T) {
	root := command.NewRootCommand()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"cockpit", "--no-tui"})

	_ = root.Execute()
}

func TestCockpitCommand_JSONOutput(t *testing.T) {
	root := command.NewRootCommand()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"cockpit", "--output", "json", "--no-tui"})

	_ = root.Execute()
}
