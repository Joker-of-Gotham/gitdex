package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestPlanCommand_Registration(t *testing.T) {
	root := command.NewRootCommand()
	found := false
	for _, cmd := range root.Commands() {
		if cmd.Name() == "plan" {
			found = true
			break
		}
	}
	if !found {
		t.Error("plan command not registered in root")
	}
}

func TestPlanCommand_Help(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"plan", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("plan --help failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "compile") {
		t.Error("help should mention compile subcommand")
	}
	if !strings.Contains(output, "show") {
		t.Error("help should mention show subcommand")
	}
	if !strings.Contains(output, "list") {
		t.Error("help should mention list subcommand")
	}
}

func TestPlanCompile_Help(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"plan", "compile", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("plan compile --help failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "goal") {
		t.Error("compile help should mention goal")
	}
}
