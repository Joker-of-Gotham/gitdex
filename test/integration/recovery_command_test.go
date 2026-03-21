package integration

import (
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/cli/command"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
)

func TestRecoveryCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	recCmd, _, err := root.Find([]string{"recovery"})
	if err != nil {
		t.Fatalf("recovery command not found: %v", err)
	}

	subs := []string{"assess", "execute", "history"}
	for _, name := range subs {
		found := false
		for _, c := range recCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'recovery'", name)
		}
	}
}

func TestRecoveryAssessRequiresTaskID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"recovery", "assess"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when task_id is missing")
	}
}

func TestRecoveryExecuteRuns(t *testing.T) {
	taskStore := orchestrator.NewMemoryTaskStore()
	_ = taskStore.SaveTask(&orchestrator.Task{
		TaskID:    "task_123",
		PlanID:    "plan_1",
		Status:    orchestrator.TaskQueued,
		UpdatedAt: time.Now().UTC(),
	})
	eng := autonomy.NewRecoveryEngine(taskStore, planning.NewMemoryPlanStore(), audit.NewMemoryAuditLedger(), autonomy.NewMemoryHandoffStore())
	restore := command.SetRecoveryEngineForTest(eng)
	defer restore()

	root := command.NewRootCommand()
	root.SetArgs([]string{"recovery", "execute", "task_123", "--strategy", "retry"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("recovery execute failed: %v", err)
	}
}

func TestRecoveryHistoryRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"recovery", "history"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("recovery history failed: %v", err)
	}
}
