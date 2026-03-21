package integration

import (
	"encoding/json"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestAPICommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	apiCmd, _, err := root.Find([]string{"api"})
	if err != nil {
		t.Fatalf("api command not found: %v", err)
	}

	subs := []string{"submit", "endpoints", "query", "get", "exchange"}
	for _, name := range subs {
		found := false
		for _, c := range apiCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'api'", name)
		}
	}
}

func TestAPISubmitRuns(t *testing.T) {
	root := command.NewRootCommand()
	payload := `{"intent":"add feature"}`
	root.SetArgs([]string{"api", "submit", "--endpoint", "intents", "--payload", payload})
	if err := root.Execute(); err != nil {
		t.Fatalf("api submit failed: %v", err)
	}
}

func TestAPISubmitRequiresEndpoint(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "submit"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when endpoint is missing")
	}
}

func TestAPIEndpointsRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "endpoints"})
	if err := root.Execute(); err != nil {
		t.Fatalf("api endpoints failed: %v", err)
	}
}

func TestAPIQueryRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "query", "--type", "tasks"})
	if err := root.Execute(); err != nil {
		t.Fatalf("api query failed: %v", err)
	}
}

func TestAPIGetRequiresEndpointAndID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "get"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when endpoint and id are missing")
	}
}

func TestAPISubmitPlans(t *testing.T) {
	root := command.NewRootCommand()
	payload, _ := json.Marshal(map[string]string{"plan_id": "plan_001", "goal": "deploy"})
	root.SetArgs([]string{"api", "submit", "--endpoint", "plans", "--payload", string(payload)})
	if err := root.Execute(); err != nil {
		t.Fatalf("api submit plans failed: %v", err)
	}
}

func TestAPISubmitTasks(t *testing.T) {
	root := command.NewRootCommand()
	payload, _ := json.Marshal(map[string]string{"task_id": "task_001", "action": "run"})
	root.SetArgs([]string{"api", "submit", "--endpoint", "tasks", "--payload", string(payload)})
	if err := root.Execute(); err != nil {
		t.Fatalf("api submit tasks failed: %v", err)
	}
}
