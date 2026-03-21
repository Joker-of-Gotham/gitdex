package gitops

import (
	"testing"
	"time"
)

func TestEvidenceCollector_Collect_Get(t *testing.T) {
	dir := t.TempDir()
	c := NewEvidenceCollector(dir)

	ev := &ExecutionEvidence{
		TaskID:    "task-123",
		Action:    "sync",
		RepoPath:  "/tmp/repo",
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
		Result:    "success",
	}
	if err := c.Collect(ev); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	got, err := c.Get("task-123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.TaskID != ev.TaskID {
		t.Errorf("Get: TaskID=%q, want %q", got.TaskID, ev.TaskID)
	}
	if got.Action != ev.Action {
		t.Errorf("Get: Action=%q, want %q", got.Action, ev.Action)
	}
	if got.Result != ev.Result {
		t.Errorf("Get: Result=%q, want %q", got.Result, ev.Result)
	}
}

func TestEvidenceCollector_List(t *testing.T) {
	dir := t.TempDir()
	c := NewEvidenceCollector(dir)

	for _, taskID := range []string{"task-a", "task-b", "task-c"} {
		ev := &ExecutionEvidence{
			TaskID:    taskID,
			Action:    "sync",
			Timestamp: time.Now(),
			Result:    "ok",
		}
		if err := c.Collect(ev); err != nil {
			t.Fatalf("Collect %s: %v", taskID, err)
		}
	}

	list, err := c.List(EvidenceFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("List: expected 3 entries, got %d", len(list))
	}
	ids := make(map[string]bool)
	for _, e := range list {
		ids[e.TaskID] = true
	}
	for _, id := range []string{"task-a", "task-b", "task-c"} {
		if !ids[id] {
			t.Errorf("List: missing %s", id)
		}
	}
}
