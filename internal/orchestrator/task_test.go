package orchestrator

import (
	"testing"
)

func TestTaskStatus_IsTerminal(t *testing.T) {
	terminal := []TaskStatus{TaskSucceeded, TaskCancelled, TaskFailedWithHandoff}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("expected %q to be terminal", s)
		}
	}

	nonTerminal := []TaskStatus{TaskQueued, TaskExecuting, TaskReconciling, TaskFailedHandoffPending, TaskQuarantined}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("expected %q to be non-terminal", s)
		}
	}
}

func TestTask_CompletedSteps(t *testing.T) {
	task := &Task{
		Steps: []StepResult{
			{Sequence: 1, Status: StepSucceeded},
			{Sequence: 2, Status: StepRunning},
			{Sequence: 3, Status: StepPending},
		},
	}

	completed := task.CompletedSteps()
	if len(completed) != 1 {
		t.Errorf("expected 1 completed step, got %d", len(completed))
	}
}

func TestTask_RunningStep(t *testing.T) {
	task := &Task{
		Steps: []StepResult{
			{Sequence: 1, Status: StepSucceeded},
			{Sequence: 2, Status: StepRunning},
			{Sequence: 3, Status: StepPending},
		},
	}

	running := task.RunningStep()
	if running == nil {
		t.Fatal("expected running step")
	}
	if running.Sequence != 2 {
		t.Errorf("expected sequence 2, got %d", running.Sequence)
	}
}

func TestTask_NoRunningStep(t *testing.T) {
	task := &Task{
		Steps: []StepResult{
			{Sequence: 1, Status: StepSucceeded},
		},
	}

	if task.RunningStep() != nil {
		t.Error("expected no running step")
	}
}

func TestGenerateIDs(t *testing.T) {
	taskID := GenerateTaskID()
	if taskID[:5] != "task_" {
		t.Errorf("task ID should start with 'task_', got %q", taskID)
	}

	corrID := GenerateCorrelationID()
	if corrID[:5] != "corr_" {
		t.Errorf("correlation ID should start with 'corr_', got %q", corrID)
	}

	evtID := GenerateEventID()
	if evtID[:4] != "evt_" {
		t.Errorf("event ID should start with 'evt_', got %q", evtID)
	}
}
