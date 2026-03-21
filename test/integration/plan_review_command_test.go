package integration

import (
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestPlanReviewCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	planCmd, _, err := root.Find([]string{"plan"})
	if err != nil {
		t.Fatalf("plan command not found: %v", err)
	}

	subs := []string{"review", "approve", "reject", "defer", "edit"}
	for _, name := range subs {
		found := false
		for _, c := range planCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'plan'", name)
		}
	}
}

func TestPlanRejectRequiresReason(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"plan", "reject", "plan_fake123"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when reason flag is missing")
	}
}

func TestPlanApproveAcceptsMode(t *testing.T) {
	root := command.NewRootCommand()
	planCmd, _, _ := root.Find([]string{"plan", "approve"})
	if planCmd == nil {
		t.Fatal("plan approve command not found")
	}

	modeFlag := planCmd.Flags().Lookup("mode")
	if modeFlag == nil {
		t.Error("--mode flag not found on plan approve command")
	}
}

func TestPlanEditAcceptsBranchAndMode(t *testing.T) {
	root := command.NewRootCommand()
	editCmd, _, _ := root.Find([]string{"plan", "edit"})
	if editCmd == nil {
		t.Fatal("plan edit command not found")
	}

	branchFlag := editCmd.Flags().Lookup("branch")
	if branchFlag == nil {
		t.Error("--branch flag not found on plan edit command")
	}

	modeFlag := editCmd.Flags().Lookup("mode")
	if modeFlag == nil {
		t.Error("--mode flag not found on plan edit command")
	}
}
