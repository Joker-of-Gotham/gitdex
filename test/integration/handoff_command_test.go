package integration

import (
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/cli/command"
)

func TestHandoffCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	handCmd, _, err := root.Find([]string{"handoff"})
	if err != nil {
		t.Fatalf("handoff command not found: %v", err)
	}

	subs := []string{"generate", "show", "list"}
	for _, name := range subs {
		found := false
		for _, c := range handCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'handoff'", name)
		}
	}
}

func TestHandoffGenerateRequiresTaskID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"handoff", "generate"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when task_id is missing")
	}
}

func TestHandoffShowRequiresPackageID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"handoff", "show"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when package_id is missing")
	}
}

func TestHandoffGenerateThenShow(t *testing.T) {
	restore := command.SetHandoffGeneratorForTest(func(app bootstrap.App, taskID string) (*autonomy.HandoffPackage, error) {
		pkg := &autonomy.HandoffPackage{
			TaskID:         taskID,
			TaskSummary:    "test handoff",
			CurrentState:   "paused",
			CompletedSteps: []string{"step 1"},
			PendingSteps:   []string{"step 2"},
			CreatedAt:      time.Now().UTC(),
		}
		if err := app.StorageProvider.HandoffStore().SavePackage(pkg); err != nil {
			return nil, err
		}
		return pkg, nil
	})
	defer restore()

	root := command.NewRootCommand()
	root.SetOut(nil)
	root.SetErr(nil)
	root.SetArgs([]string{"handoff", "generate", "task_gen_1", "--output", "json"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("handoff generate failed: %v", err)
	}
}
