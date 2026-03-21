package autonomy

import (
	"testing"
)

func TestMemoryHandoffStore_SaveAndGet(t *testing.T) {
	store := NewMemoryHandoffStore()

	pkg := &HandoffPackage{
		TaskID:         "task_1",
		TaskSummary:    "test summary",
		CurrentState:   "running",
		CompletedSteps: []string{"step1"},
		PendingSteps:   []string{"step2"},
	}

	err := store.SavePackage(pkg)
	if err != nil {
		t.Fatalf("SavePackage error: %v", err)
	}
	if pkg.PackageID == "" {
		t.Error("PackageID should be assigned after save")
	}

	got, err := store.GetPackage(pkg.PackageID)
	if err != nil {
		t.Fatalf("GetPackage error: %v", err)
	}
	if got.TaskID != pkg.TaskID {
		t.Errorf("TaskID: got %q", got.TaskID)
	}
	if got.TaskSummary != pkg.TaskSummary {
		t.Errorf("TaskSummary: got %q", got.TaskSummary)
	}
}

func TestMemoryHandoffStore_GetByTaskID(t *testing.T) {
	store := NewMemoryHandoffStore()
	pkg := &HandoffPackage{
		TaskID:       "task_by_id",
		TaskSummary:  "test",
		CurrentState: "running",
	}
	if err := store.SavePackage(pkg); err != nil {
		t.Fatalf("SavePackage: %v", err)
	}

	got, err := store.GetByTaskID("task_by_id")
	if err != nil {
		t.Fatalf("GetByTaskID error: %v", err)
	}
	if got.PackageID != pkg.PackageID {
		t.Errorf("PackageID: got %q", got.PackageID)
	}
}

func TestMemoryHandoffStore_ListPackages(t *testing.T) {
	store := NewMemoryHandoffStore()
	for _, id := range []string{"t1", "t2"} {
		pkg := &HandoffPackage{TaskID: id, TaskSummary: id, CurrentState: "running"}
		if err := store.SavePackage(pkg); err != nil {
			t.Fatalf("SavePackage(%s): %v", id, err)
		}
	}
	list, err := store.ListPackages()
	if err != nil {
		t.Fatalf("ListPackages error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListPackages: got %d, want 2", len(list))
	}
}
