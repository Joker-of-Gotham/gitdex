package reviewer

import (
	"context"
	"testing"

	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/policy"
)

func seedPlan(store planning.PlanStore, status planning.PlanStatus, risk planning.RiskLevel) *planning.Plan {
	p := &planning.Plan{
		PlanID:    "plan_test123",
		Status:    status,
		RiskLevel: risk,
		Intent: planning.PlanIntent{
			Source:     "command",
			RawInput:   "test goal",
			ActionType: "plan",
		},
		Scope: planning.PlanScope{
			Owner: "org",
			Repo:  "repo",
		},
		Steps: []planning.PlanStep{
			{Sequence: 1, Action: "test", Target: "repo", Description: "test step", RiskLevel: risk, Reversible: true},
		},
		PolicyResult: &planning.PolicyResult{
			Verdict:     planning.VerdictAllowed,
			Reason:      "low risk",
			Explanation: "safe",
		},
	}
	_ = store.Save(p)
	return p
}

func TestApprove_Success(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanReviewRequired, planning.RiskLow)

	rev := New(store, policy.NewDefaultEngine())
	err := rev.Approve(context.Background(), "plan_test123", "admin", "looks good", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plan, _ := store.Get("plan_test123")
	if plan.Status != planning.PlanApproved {
		t.Errorf("got status %q, want %q", plan.Status, planning.PlanApproved)
	}
	if plan.ExecutionMode != planning.ModeExecute {
		t.Errorf("got mode %q, want %q", plan.ExecutionMode, planning.ModeExecute)
	}

	records, _ := store.GetApprovals("plan_test123")
	if len(records) != 1 {
		t.Fatalf("expected 1 approval record, got %d", len(records))
	}
	if records[0].Action != planning.ActionApprove {
		t.Errorf("got action %q, want %q", records[0].Action, planning.ActionApprove)
	}
}

func TestApprove_WithMode(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanReviewRequired, planning.RiskLow)

	mode := planning.ModeDryRun
	rev := New(store, policy.NewDefaultEngine())
	err := rev.Approve(context.Background(), "plan_test123", "admin", "", &mode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plan, _ := store.Get("plan_test123")
	if plan.ExecutionMode != planning.ModeDryRun {
		t.Errorf("got mode %q, want %q", plan.ExecutionMode, planning.ModeDryRun)
	}
}

func TestApprove_BlockedPlanFails(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanBlocked, planning.RiskCritical)

	rev := New(store, policy.NewDefaultEngine())
	err := rev.Approve(context.Background(), "plan_test123", "admin", "", nil)
	if err == nil {
		t.Fatal("expected error when approving blocked plan")
	}
}

func TestReject_Success(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanReviewRequired, planning.RiskLow)

	rev := New(store, policy.NewDefaultEngine())
	err := rev.Reject(context.Background(), "plan_test123", "admin", "too risky")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plan, _ := store.Get("plan_test123")
	if plan.Status != planning.PlanBlocked {
		t.Errorf("got status %q, want %q", plan.Status, planning.PlanBlocked)
	}

	records, _ := store.GetApprovals("plan_test123")
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Reason != "too risky" {
		t.Errorf("got reason %q, want %q", records[0].Reason, "too risky")
	}
}

func TestReject_EmptyReasonFails(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanReviewRequired, planning.RiskLow)

	rev := New(store, policy.NewDefaultEngine())
	err := rev.Reject(context.Background(), "plan_test123", "admin", "")
	if err == nil {
		t.Fatal("expected error for empty rejection reason")
	}
}

func TestDefer_Success(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanReviewRequired, planning.RiskLow)

	rev := New(store, policy.NewDefaultEngine())
	err := rev.Defer(context.Background(), "plan_test123", "admin", "not ready")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plan, _ := store.Get("plan_test123")
	if plan.Status != planning.PlanDraft {
		t.Errorf("got status %q, want %q", plan.Status, planning.PlanDraft)
	}
}

func TestEdit_ChangeBranch(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanReviewRequired, planning.RiskLow)

	rev := New(store, policy.NewDefaultEngine())
	branch := "feature/safe"
	err := rev.Edit(context.Background(), "plan_test123", "admin", PlanEdits{Branch: &branch})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plan, _ := store.Get("plan_test123")
	if plan.Scope.Branch != "feature/safe" {
		t.Errorf("got branch %q, want %q", plan.Scope.Branch, "feature/safe")
	}
	if plan.Status != planning.PlanReviewRequired {
		t.Errorf("got status %q, want %q", plan.Status, planning.PlanReviewRequired)
	}
}

func TestEdit_BlockedPlanCanBeEdited(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanBlocked, planning.RiskCritical)

	rev := New(store, policy.NewDefaultEngine())
	branch := "feature/safe"
	err := rev.Edit(context.Background(), "plan_test123", "admin", PlanEdits{Branch: &branch})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plan, _ := store.Get("plan_test123")
	if plan.Scope.Branch != "feature/safe" {
		t.Errorf("got branch %q, want %q", plan.Scope.Branch, "feature/safe")
	}
}

func TestEdit_NoChangesFails(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanReviewRequired, planning.RiskLow)

	rev := New(store, policy.NewDefaultEngine())
	err := rev.Edit(context.Background(), "plan_test123", "admin", PlanEdits{})
	if err == nil {
		t.Fatal("expected error for empty edits")
	}
}

func TestEdit_ApprovedPlanCannotBeEdited(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanApproved, planning.RiskLow)

	rev := New(store, policy.NewDefaultEngine())
	branch := "new-branch"
	err := rev.Edit(context.Background(), "plan_test123", "admin", PlanEdits{Branch: &branch})
	if err == nil {
		t.Fatal("expected error editing approved plan")
	}
}

func TestApprovalRecord_Persists(t *testing.T) {
	store := planning.NewMemoryPlanStore()
	seedPlan(store, planning.PlanReviewRequired, planning.RiskLow)

	rev := New(store, policy.NewDefaultEngine())
	_ = rev.Approve(context.Background(), "plan_test123", "admin1", "ok", nil)

	store2 := planning.NewMemoryPlanStore()
	seedPlan(store2, planning.PlanReviewRequired, planning.RiskLow)
	rev2 := New(store2, policy.NewDefaultEngine())
	_ = rev2.Reject(context.Background(), "plan_test123", "admin2", "no")

	records, _ := store2.GetApprovals("plan_test123")
	if len(records) != 1 {
		t.Fatalf("expected 1 record in second store, got %d", len(records))
	}
}
