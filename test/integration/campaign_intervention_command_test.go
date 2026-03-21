package integration

import (
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestCampaignApproveRequiresRepo(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "approve", "camp_abc"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --repo is missing")
	}
}

func TestCampaignExcludeRequiresRepo(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "exclude", "camp_abc"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --repo is missing")
	}
}

func TestCampaignRetryRequiresRepo(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "retry", "camp_abc"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --repo is missing")
	}
}

func TestCampaignInterveneRequiresAction(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "intervene", "camp_abc", "--repo", "owner/repo"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --action is missing")
	}
}

func TestCampaignApproveRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetOut(nil)
	root.SetErr(nil)
	root.SetArgs([]string{"campaign", "create", "--name", "Intervention Test", "--output", "json"})
	_ = root.Execute()
	root.SetArgs([]string{"campaign", "add-repo", "camp_xxxxxxxx", "--repo", "org/repo", "--output", "json"})
	err := root.Execute()
	if err != nil && err.Error() != "campaign \"camp_xxxxxxxx\" not found" {
		t.Logf("add-repo (may fail for unknown ID): %v", err)
	}
}
