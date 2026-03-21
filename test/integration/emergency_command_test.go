package integration

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/cli/command"
	"github.com/your-org/gitdex/internal/emergency"
	"github.com/your-org/gitdex/internal/orchestrator"
)

func TestEmergencyCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	emergCmd, _, err := root.Find([]string{"emergency"})
	if err != nil {
		t.Fatalf("emergency command not found: %v", err)
	}

	subs := []string{"pause", "suspend", "kill", "status"}
	for _, name := range subs {
		found := false
		for _, c := range emergCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'emergency'", name)
		}
	}
}

func TestEmergencyPauseRequiresTaskID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"emergency", "pause"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when task_id is missing")
	}
}

func TestEmergencySuspendRequiresScope(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"emergency", "suspend"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when scope is missing")
	}
}

func TestEmergencyKillRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"emergency", "kill"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("emergency kill failed: %v", err)
	}
}

func TestEmergencyStatusRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"emergency", "status"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("emergency status failed: %v", err)
	}
}

func TestEmergencyPauseHappyPath(t *testing.T) {
	taskStore := orchestrator.NewMemoryTaskStore()
	_ = taskStore.SaveTask(&orchestrator.Task{
		TaskID:      "task_123",
		Status:      orchestrator.TaskExecuting,
		CurrentStep: 1,
		UpdatedAt:   time.Now().UTC(),
	})
	ledger := audit.NewMemoryAuditLedger()
	restore := command.SetControlEngineForTest(emergency.NewControlEngine(
		taskStore,
		ledger,
		autonomy.NewTaskController(taskStore, ledger),
	))
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"emergency", "pause", "task_123"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("emergency pause task_123 failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "OK") && !strings.Contains(output, "paused") {
		t.Errorf("output should indicate success or paused: %q", output)
	}
	if !strings.Contains(output, "task_123") {
		t.Error("output should mention task_123")
	}
}

func TestEmergencySuspendHappyPath(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"emergency", "suspend", "write_repo"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("emergency suspend write_repo failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "FAILED") || !strings.Contains(output, "requires a real capability registry") {
		t.Errorf("output should indicate unsupported real capability suspension: %q", output)
	}
	if !strings.Contains(output, "write_repo") {
		t.Error("output should mention scope write_repo")
	}
}
