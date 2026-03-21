package integration

import (
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestTaskCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	taskCmd, _, err := root.Find([]string{"task"})
	if err != nil {
		t.Fatalf("task command not found: %v", err)
	}

	subs := []string{"start", "status", "list"}
	for _, name := range subs {
		found := false
		for _, c := range taskCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'task'", name)
		}
	}
}

func TestTaskStartRequiresPlanID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"task", "start"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when plan_id is missing")
	}
}

func TestTaskStatusRequiresTaskID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"task", "status"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when task_id is missing")
	}
}
