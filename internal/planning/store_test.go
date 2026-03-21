package planning

import (
	"testing"
	"time"
)

func TestMemoryPlanStore_SaveAndGet(t *testing.T) {
	store := NewMemoryPlanStore()
	plan := &Plan{
		PlanID:    "plan_test1",
		TaskID:    "task_test1",
		Status:    PlanDraft,
		CreatedAt: time.Now().UTC(),
	}

	if err := store.Save(plan); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, err := store.Get("plan_test1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.PlanID != "plan_test1" {
		t.Errorf("expected plan_test1, got %s", got.PlanID)
	}
}

func TestMemoryPlanStore_GetNotFound(t *testing.T) {
	store := NewMemoryPlanStore()
	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

func TestMemoryPlanStore_GetByTaskID(t *testing.T) {
	store := NewMemoryPlanStore()
	plan := &Plan{PlanID: "plan_t2", TaskID: "task_t2", Status: PlanDraft}
	_ = store.Save(plan)

	got, err := store.GetByTaskID("task_t2")
	if err != nil {
		t.Fatalf("get by task ID failed: %v", err)
	}
	if got.PlanID != "plan_t2" {
		t.Errorf("expected plan_t2, got %s", got.PlanID)
	}
}

func TestMemoryPlanStore_GetByTaskID_NotFound(t *testing.T) {
	store := NewMemoryPlanStore()
	_, err := store.GetByTaskID("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task ID")
	}
}

func TestMemoryPlanStore_List(t *testing.T) {
	store := NewMemoryPlanStore()
	_ = store.Save(&Plan{PlanID: "plan_a", Status: PlanDraft})
	_ = store.Save(&Plan{PlanID: "plan_b", Status: PlanDraft})

	plans, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}
}

func TestMemoryPlanStore_UpdateStatus(t *testing.T) {
	store := NewMemoryPlanStore()
	_ = store.Save(&Plan{PlanID: "plan_u", Status: PlanDraft})

	if err := store.UpdateStatus("plan_u", PlanApproved); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	got, _ := store.Get("plan_u")
	if got.Status != PlanApproved {
		t.Errorf("expected approved, got %s", got.Status)
	}
}

func TestMemoryPlanStore_UpdateStatus_NotFound(t *testing.T) {
	store := NewMemoryPlanStore()
	err := store.UpdateStatus("nonexistent", PlanApproved)
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

func TestMemoryPlanStore_SaveNil(t *testing.T) {
	store := NewMemoryPlanStore()
	err := store.Save(nil)
	if err == nil {
		t.Error("expected error for nil plan")
	}
}

func TestMemoryPlanStore_IsolatesCopies(t *testing.T) {
	store := NewMemoryPlanStore()
	plan := &Plan{PlanID: "plan_iso", Status: PlanDraft}
	_ = store.Save(plan)

	plan.Status = PlanApproved

	got, _ := store.Get("plan_iso")
	if got.Status != PlanDraft {
		t.Error("store should isolate copies from mutations")
	}
}
