package autonomy

import (
	"context"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/orchestrator"
)

func seedTask(t *testing.T, store orchestrator.TaskStore, taskID string, status orchestrator.TaskStatus) {
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

func TestDefaultTaskController_Pause(t *testing.T) {
	taskStore := orchestrator.NewMemoryTaskStore()
	auditLedger := audit.NewMemoryAuditLedger()
	seedTask(t, taskStore, "task_123", orchestrator.TaskExecuting)
	ctrl := NewTaskController(taskStore, auditLedger)
	ctx := context.Background()

	req := TaskControlRequest{
		Action:    TaskControlPause,
		TaskID:    "task_123",
		Reason:    "test",
		Actor:     "admin",
		Timestamp: time.Now().UTC(),
	}

	result, err := ctrl.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got message %q", result.Message)
	}
	if result.NewStatus != "paused" {
		t.Errorf("NewStatus: got %q, want paused", result.NewStatus)
	}
}

func TestDefaultTaskController_Resume(t *testing.T) {
	taskStore := orchestrator.NewMemoryTaskStore()
	auditLedger := audit.NewMemoryAuditLedger()
	seedTask(t, taskStore, "task_resume", orchestrator.TaskPaused)
	ctrl := NewTaskController(taskStore, auditLedger)
	ctx := context.Background()

	req := TaskControlRequest{
		Action:    TaskControlResume,
		TaskID:    "task_resume",
		Actor:     "admin",
		Timestamp: time.Now().UTC(),
	}

	result, err := ctrl.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got message %q", result.Message)
	}
	if result.NewStatus != "queued" {
		t.Errorf("NewStatus: got %q, want queued", result.NewStatus)
	}
}

func TestDefaultTaskController_Cancel(t *testing.T) {
	taskStore := orchestrator.NewMemoryTaskStore()
	auditLedger := audit.NewMemoryAuditLedger()
	seedTask(t, taskStore, "task_cancel", orchestrator.TaskExecuting)
	ctrl := NewTaskController(taskStore, auditLedger)
	ctx := context.Background()

	req := TaskControlRequest{
		Action:    TaskControlCancel,
		TaskID:    "task_cancel",
		Reason:    "user requested",
		Actor:     "cli",
		Timestamp: time.Now().UTC(),
	}

	result, err := ctrl.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success")
	}
	if result.NewStatus != "cancelled" {
		t.Errorf("NewStatus: got %q, want cancelled", result.NewStatus)
	}
}

func TestDefaultTaskController_Takeover(t *testing.T) {
	taskStore := orchestrator.NewMemoryTaskStore()
	auditLedger := audit.NewMemoryAuditLedger()
	seedTask(t, taskStore, "task_takeover", orchestrator.TaskExecuting)
	ctrl := NewTaskController(taskStore, auditLedger)
	ctx := context.Background()

	req := TaskControlRequest{
		Action:    TaskControlTakeover,
		TaskID:    "task_takeover",
		Actor:     "cli",
		Timestamp: time.Now().UTC(),
	}

	result, err := ctrl.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success")
	}
	if result.NewStatus != "manual" {
		t.Errorf("NewStatus: got %q, want manual", result.NewStatus)
	}
}

func TestDefaultTaskController_ResumeWithoutPause(t *testing.T) {
	taskStore := orchestrator.NewMemoryTaskStore()
	auditLedger := audit.NewMemoryAuditLedger()
	seedTask(t, taskStore, "task_not_paused", orchestrator.TaskQueued)
	ctrl := NewTaskController(taskStore, auditLedger)
	ctx := context.Background()

	req := TaskControlRequest{
		Action:    TaskControlResume,
		TaskID:    "task_not_paused",
		Actor:     "cli",
		Timestamp: time.Now().UTC(),
	}

	result, err := ctrl.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Success {
		t.Error("expected failure when resuming non-paused task")
	}
}
