package conformance

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/orchestrator"
)

func TestTask_JSONContract(t *testing.T) {
	now := time.Now().UTC()
	task := &orchestrator.Task{
		TaskID:        "task_contract",
		CorrelationID: "corr_contract",
		PlanID:        "plan_contract",
		Status:        orchestrator.TaskQueued,
		CurrentStep:   1,
		Steps: []orchestrator.StepResult{
			{Sequence: 1, Action: "test", Target: "repo", Status: orchestrator.StepPending},
		},
		UpdatedAt: now,
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"task_id"`,
		`"correlation_id"`,
		`"plan_id"`,
		`"status"`,
		`"current_step"`,
		`"steps"`,
		`"updated_at"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !containsStr(raw, f) {
			t.Errorf("JSON missing field %s", f)
		}
	}
}

func TestTaskEvent_JSONContract(t *testing.T) {
	evt := &orchestrator.TaskEvent{
		EventID:      "evt_contract",
		TaskID:       "task_contract",
		FromStatus:   orchestrator.TaskQueued,
		ToStatus:     orchestrator.TaskExecuting,
		StepSequence: 1,
		Message:      "test event",
		Timestamp:    time.Now().UTC(),
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"event_id"`,
		`"task_id"`,
		`"from_status"`,
		`"to_status"`,
		`"step_sequence"`,
		`"message"`,
		`"timestamp"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !containsStr(raw, f) {
			t.Errorf("JSON missing field %s", f)
		}
	}
}

func TestTask_RoundTrip(t *testing.T) {
	now := time.Now().UTC()
	original := &orchestrator.Task{
		TaskID:        "task_rt",
		CorrelationID: "corr_rt",
		PlanID:        "plan_rt",
		Status:        orchestrator.TaskSucceeded,
		CurrentStep:   2,
		Steps: []orchestrator.StepResult{
			{Sequence: 1, Action: "fetch", Target: "origin", Status: orchestrator.StepSucceeded},
			{Sequence: 2, Action: "merge", Target: "main", Status: orchestrator.StepSucceeded},
		},
		StartedAt:  &now,
		UpdatedAt:  now,
		FinishedAt: &now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded orchestrator.Task
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.TaskID != original.TaskID {
		t.Errorf("TaskID: got %q, want %q", decoded.TaskID, original.TaskID)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status: got %q, want %q", decoded.Status, original.Status)
	}
	if len(decoded.Steps) != len(original.Steps) {
		t.Errorf("Steps: got %d, want %d", len(decoded.Steps), len(original.Steps))
	}
}

func TestTaskStatus_AllValues(t *testing.T) {
	statuses := []orchestrator.TaskStatus{
		orchestrator.TaskQueued,
		orchestrator.TaskExecuting,
		orchestrator.TaskReconciling,
		orchestrator.TaskSucceeded,
		orchestrator.TaskFailedHandoffPending,
		orchestrator.TaskQuarantined,
		orchestrator.TaskCancelled,
		orchestrator.TaskFailedWithHandoff,
	}

	seen := make(map[orchestrator.TaskStatus]bool)
	for _, s := range statuses {
		if s == "" {
			t.Error("task status should not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate status: %s", s)
		}
		seen[s] = true
	}
}

func TestStepStatus_AllValues(t *testing.T) {
	statuses := []orchestrator.StepStatus{
		orchestrator.StepPending,
		orchestrator.StepRunning,
		orchestrator.StepSucceeded,
		orchestrator.StepFailed,
		orchestrator.StepSkipped,
	}

	seen := make(map[orchestrator.StepStatus]bool)
	for _, s := range statuses {
		if s == "" {
			t.Error("step status should not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate step status: %s", s)
		}
		seen[s] = true
	}
}
