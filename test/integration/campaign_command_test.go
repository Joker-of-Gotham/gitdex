package integration

import (
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestCampaignCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	campCmd, _, err := root.Find([]string{"campaign"})
	if err != nil {
		t.Fatalf("campaign command not found: %v", err)
	}
	subs := []string{"create", "show", "list", "add-repo", "remove-repo", "matrix", "status", "approve", "exclude", "retry", "intervene"}
	for _, name := range subs {
		found := false
		for _, c := range campCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'campaign'", name)
		}
	}
}

func TestCampaignCreateRequiresName(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "create"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --name is missing")
	}
}

func TestCampaignCreateRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetOut(nil)
	root.SetErr(nil)
	root.SetArgs([]string{"campaign", "create", "--name", "My Campaign", "--description", "Test desc", "--output", "json"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("campaign create failed: %v", err)
	}
}

func TestCampaignShowRequiresCampaignID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "show"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when campaign_id is missing")
	}
}

func TestCampaignListRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "list", "--output", "json"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("campaign list failed: %v", err)
	}
}

func TestCampaignAddRepoRequiresRepo(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "add-repo", "camp_abc"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --repo is missing")
	}
}
