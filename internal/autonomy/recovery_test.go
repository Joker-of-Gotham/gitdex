package autonomy

import (
	"context"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
)

func seedRecoveryTask(t *testing.T, store orchestrator.TaskStore, taskID string, status orchestrator.TaskStatus) {
	t.Helper()
	task := &orchestrator.Task{
		TaskID:        taskID,
		CorrelationID: "corr_" + taskID,
		PlanID:        "plan_1",
		Status:        status,
		CurrentStep:   0,
		Steps:         []orchestrator.StepResult{},
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.SaveTask(task); err != nil {
		t.Fatalf("seed task: %v", err)
	}
}

func TestDefaultRecoveryEngine_Assess(t *testing.T) {
	taskStore := orchestrator.NewMemoryTaskStore()
	seedRecoveryTask(t, taskStore, "task_123", orchestrator.TaskFailedHandoffPending)
	eng := NewRecoveryEngine(taskStore, planning.NewMemoryPlanStore(), audit.NewMemoryAuditLedger(), NewMemoryHandoffStore())
	ctx := context.Background()

	assessment, err := eng.Assess(ctx, "task_123")
	if err != nil {
		t.Fatalf("Assess error: %v", err)
	}
	if assessment.TaskID != "task_123" {
		t.Errorf("TaskID: got %q, want task_123", assessment.TaskID)
	}
	if assessment.FailureType == "" {
		t.Error("FailureType should not be empty")
	}
	if assessment.RecommendedStrategy == "" {
		t.Error("RecommendedStrategy should not be empty")
	}
}

func TestDefaultRecoveryEngine_Execute(t *testing.T) {
	taskStore := orchestrator.NewMemoryTaskStore()
	seedRecoveryTask(t, taskStore, "task_456", orchestrator.TaskFailedHandoffPending)
	eng := NewRecoveryEngine(taskStore, planning.NewMemoryPlanStore(), audit.NewMemoryAuditLedger(), NewMemoryHandoffStore())
	ctx := context.Background()

	req := RecoveryRequest{
		TaskID:   "task_456",
		Strategy: RecoveryRetry,
		Actor:    "cli",
	}

	result, err := eng.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got message %q", result.Message)
	}
	if result.FinalStatus == "" {
		t.Error("FinalStatus should not be empty")
	}
	if result.RecoveredAt.IsZero() {
		t.Error("RecoveredAt should not be zero")
	}
}

func TestDefaultRecoveryEngine_History(t *testing.T) {
	taskStore := orchestrator.NewMemoryTaskStore()
	seedRecoveryTask(t, taskStore, "t1", orchestrator.TaskFailedHandoffPending)
	seedRecoveryTask(t, taskStore, "t2", orchestrator.TaskFailedHandoffPending)
	eng := NewRecoveryEngine(taskStore, planning.NewMemoryPlanStore(), audit.NewMemoryAuditLedger(), NewMemoryHandoffStore())
	ctx := context.Background()

	_, _ = eng.Execute(ctx, RecoveryRequest{TaskID: "t1", Strategy: RecoveryRetry, Actor: "cli"})
	_, _ = eng.Execute(ctx, RecoveryRequest{TaskID: "t2", Strategy: RecoveryRollback, Actor: "cli"})

	all := eng.History("")
	if len(all) != 2 {
		t.Errorf("History(): got %d entries, want 2", len(all))
	}

	filtered := eng.History("t1")
	if len(filtered) != 1 {
		t.Errorf("History(t1): got %d entries, want 1", len(filtered))
	}
	if filtered[0].Request.TaskID != "t1" {
		t.Errorf("filtered task: got %q", filtered[0].Request.TaskID)
	}
}
