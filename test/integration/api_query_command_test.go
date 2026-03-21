package integration

import (
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestAPIQueryWithFilter(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "query", "--type", "tasks", "--filter", "status=running"})
	if err := root.Execute(); err != nil {
		t.Fatalf("api query with filter failed: %v", err)
	}
}

func TestAPIQueryCampaigns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "query", "--type", "campaigns"})
	if err := root.Execute(); err != nil {
		t.Fatalf("api query campaigns failed: %v", err)
	}
}

func TestAPIQueryAudit(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "query", "--type", "audit"})
	if err := root.Execute(); err != nil {
		t.Fatalf("api query audit failed: %v", err)
	}
}

func TestAPIGetTask(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "get", "--endpoint", "tasks", "--id", "task_001"})
	if err := root.Execute(); err != nil {
		t.Fatalf("api get task failed: %v", err)
	}
}

func TestAPIGetCampaign(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "get", "--endpoint", "campaigns", "--id", "camp_001"})
	if err := root.Execute(); err != nil {
		t.Fatalf("api get campaign failed: %v", err)
	}
}
