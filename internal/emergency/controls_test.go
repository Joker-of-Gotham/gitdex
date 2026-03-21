package emergency

import (
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/orchestrator"
)

func TestDefaultControlEngine_PauseTask(t *testing.T) {
	eng := NewDefaultControlEngine()
	if err := eng.taskStore.SaveTask(&orchestrator.Task{
		TaskID:      "task_123",
		Status:      orchestrator.TaskExecuting,
		CurrentStep: 1,
	}); err != nil {
		t.Fatalf("SaveTask error: %v", err)
	}

	req := ControlRequest{
		Action:    ControlPauseTask,
		Scope:     "task_123",
		Reason:    "test",
		Actor:     "admin",
		Timestamp: time.Now().UTC(),
	}

	result, err := eng.Execute(req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got message %q", result.Message)
	}
	if len(result.AffectedTasks) != 1 || result.AffectedTasks[0] != "task_123" {
		t.Errorf("AffectedTasks: got %v", result.AffectedTasks)
	}
}

func TestDefaultControlEngine_KillSwitch(t *testing.T) {
	eng := NewDefaultControlEngine()

	req := ControlRequest{
		Action:    ControlKillSwitch,
		Scope:     "*",
		Reason:    "emergency",
		Actor:     "ops",
		Timestamp: time.Now().UTC(),
	}

	result, err := eng.Execute(req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success")
	}
	// With real TaskStore, AffectedTasks lists cancelled task IDs (empty when no active tasks)
	if result.AffectedTasks == nil {
		t.Errorf("AffectedTasks should not be nil")
	}
}

func TestDefaultControlEngine_Status(t *testing.T) {
	eng := NewDefaultControlEngine()

	_, _ = eng.Execute(ControlRequest{
		Action: ControlPauseTask,
		Scope:  "task_a",
		Actor:  "cli",
	})

	controls := eng.Status()
	if len(controls) != 1 {
		t.Errorf("expected 1 control, got %d", len(controls))
	}
	if controls[0].Action != ControlPauseTask {
		t.Errorf("Action: got %s", controls[0].Action)
	}
}

func TestDefaultControlEngine_PauseScope(t *testing.T) {
	eng := NewDefaultControlEngine()

	req := ControlRequest{
		Action:    ControlPauseScope,
		Scope:     "scope_repo_main",
		Reason:    "test",
		Actor:     "admin",
		Timestamp: time.Now().UTC(),
	}

	result, err := eng.Execute(req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Success {
		t.Errorf("expected failure, got message %q", result.Message)
	}
	if len(result.AffectedScopes) != 1 || result.AffectedScopes[0] != "scope_repo_main" {
		t.Errorf("AffectedScopes: got %v", result.AffectedScopes)
	}
	if !contains(result.Message, "not implemented") {
		t.Errorf("Message should mention not implemented: %q", result.Message)
	}
}

func TestDefaultControlEngine_SuspendCapability(t *testing.T) {
	eng := NewDefaultControlEngine()

	req := ControlRequest{
		Action:    ControlSuspendCapability,
		Scope:     "write_repo",
		Reason:    "test",
		Actor:     "ops",
		Timestamp: time.Now().UTC(),
	}

	result, err := eng.Execute(req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Success {
		t.Errorf("expected failure, got message %q", result.Message)
	}
	if len(result.AffectedScopes) != 1 || result.AffectedScopes[0] != "write_repo" {
		t.Errorf("AffectedScopes: got %v", result.AffectedScopes)
	}
	if !contains(result.Message, "requires a real capability registry") {
		t.Errorf("Message should mention capability registry: %q", result.Message)
	}
}

func TestDefaultControlEngine_UnknownAction(t *testing.T) {
	eng := NewDefaultControlEngine()

	req := ControlRequest{
		Action:    ControlAction("unknown_action"),
		Scope:     "x",
		Reason:    "test",
		Actor:     "cli",
		Timestamp: time.Now().UTC(),
	}

	result, err := eng.Execute(req)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Success {
		t.Error("expected success=false for unknown action")
	}
	if !contains(result.Message, "unknown") {
		t.Errorf("Message should mention unknown: %q", result.Message)
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
