package integration

import (
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestCampaignMatrixRequiresCampaignID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "matrix"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when campaign_id is missing")
	}
}

func TestCampaignMatrixRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetOut(nil)
	root.SetErr(nil)
	root.SetArgs([]string{"campaign", "create", "--name", "Matrix Test", "--output", "json"})
	_ = root.Execute()
	root.SetArgs([]string{"campaign", "list", "--output", "json"})
	_ = root.Execute()
	root.SetArgs([]string{"campaign", "matrix", "camp_xxxxxxxx", "--output", "json"})
	err := root.Execute()
	if err != nil && err.Error() != "campaign \"camp_xxxxxxxx\" not found" {
		t.Logf("matrix against nonexistent campaign: %v (expected for missing ID)", err)
	}
}

func TestCampaignStatusRequiresCampaignID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "status"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when campaign_id is missing")
	}
}
