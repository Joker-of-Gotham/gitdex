package orchestrator

import (
	"testing"
	"time"
)

func TestMemoryTaskStore_SaveAndGet(t *testing.T) {
	store := NewMemoryTaskStore()
	task := &Task{
		TaskID:        "task_abc",
		CorrelationID: "corr_abc",
		PlanID:        "plan_abc",
		Status:        TaskQueued,
		Steps: []StepResult{
			{Sequence: 1, Action: "test", Target: "repo", Status: StepPending},
		},
	}

	if err := store.SaveTask(task); err != nil {
		t.Fatalf("save error: %v", err)
	}

	got, err := store.GetTask("task_abc")
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if got.TaskID != "task_abc" {
		t.Errorf("got TaskID %q, want %q", got.TaskID, "task_abc")
	}
	if len(got.Steps) != 1 {
		t.Errorf("got %d steps, want 1", len(got.Steps))
	}
}

func TestMemoryTaskStore_GetNotFound(t *testing.T) {
	store := NewMemoryTaskStore()
	_, err := store.GetTask("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}

func TestMemoryTaskStore_GetByCorrelationID(t *testing.T) {
	store := NewMemoryTaskStore()
	task := &Task{
		TaskID:        "task_corr",
		CorrelationID: "corr_xyz",
		PlanID:        "plan_xyz",
		Status:        TaskQueued,
	}
	_ = store.SaveTask(task)

	got, err := store.GetByCorrelationID("corr_xyz")
	if err != nil {
		t.Fatalf("get by correlation error: %v", err)
	}
	if got.TaskID != "task_corr" {
		t.Errorf("got TaskID %q, want %q", got.TaskID, "task_corr")
	}
}

func TestMemoryTaskStore_ListTasks(t *testing.T) {
	store := NewMemoryTaskStore()
	_ = store.SaveTask(&Task{TaskID: "t1", Status: TaskQueued})
	_ = store.SaveTask(&Task{TaskID: "t2", Status: TaskExecuting})

	tasks, err := store.ListTasks()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2", len(tasks))
	}
}

func TestMemoryTaskStore_UpdateTask(t *testing.T) {
	store := NewMemoryTaskStore()
	task := &Task{TaskID: "t_update", Status: TaskQueued}
	_ = store.SaveTask(task)

	task.Status = TaskExecuting
	if err := store.UpdateTask(task); err != nil {
		t.Fatalf("update error: %v", err)
	}

	got, _ := store.GetTask("t_update")
	if got.Status != TaskExecuting {
		t.Errorf("got status %q, want %q", got.Status, TaskExecuting)
	}
}

func TestMemoryTaskStore_UpdateNotFound(t *testing.T) {
	store := NewMemoryTaskStore()
	err := store.UpdateTask(&Task{TaskID: "nonexistent", Status: TaskQueued})
	if err == nil {
		t.Fatal("expected error updating nonexistent task")
	}
}

func TestMemoryTaskStore_AppendAndGetEvents(t *testing.T) {
	store := NewMemoryTaskStore()

	evt := &TaskEvent{
		TaskID:     "task_evt",
		FromStatus: TaskQueued,
		ToStatus:   TaskExecuting,
		Message:    "started",
		Timestamp:  time.Now().UTC(),
	}
	if err := store.AppendEvent(evt); err != nil {
		t.Fatalf("append error: %v", err)
	}

	events, err := store.GetEvents("task_evt")
	if err != nil {
		t.Fatalf("get events error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].EventID == "" {
		t.Error("expected auto-generated event ID")
	}
	if events[0].Message != "started" {
		t.Errorf("got message %q, want %q", events[0].Message, "started")
	}
}

func TestMemoryTaskStore_CopyIsolation(t *testing.T) {
	store := NewMemoryTaskStore()
	task := &Task{
		TaskID: "t_iso",
		Status: TaskQueued,
		Steps: []StepResult{
			{Sequence: 1, Status: StepPending},
		},
	}
	_ = store.SaveTask(task)

	task.Steps[0].Status = StepRunning

	got, _ := store.GetTask("t_iso")
	if got.Steps[0].Status != StepPending {
		t.Error("store should be isolated from caller mutations")
	}
}
