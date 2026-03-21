package conformance

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/planning"
)

func TestApprovalRecord_JSONContract(t *testing.T) {
	rec := &planning.ApprovalRecord{
		RecordID:       "approval_abc123",
		PlanID:         "plan_xyz789",
		Action:         planning.ActionApprove,
		Actor:          "admin",
		Reason:         "looks good",
		PreviousStatus: planning.PlanReviewRequired,
		NewStatus:      planning.PlanApproved,
		CreatedAt:      time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"record_id"`,
		`"plan_id"`,
		`"action"`,
		`"actor"`,
		`"reason"`,
		`"previous_status"`,
		`"new_status"`,
		`"created_at"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !containsStr(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestApprovalRecord_RoundTrip(t *testing.T) {
	original := &planning.ApprovalRecord{
		RecordID:       "approval_roundtrip",
		PlanID:         "plan_roundtrip",
		Action:         planning.ActionReject,
		Actor:          "security",
		Reason:         "policy violation",
		PreviousStatus: planning.PlanReviewRequired,
		NewStatus:      planning.PlanBlocked,
		CreatedAt:      time.Date(2026, 3, 18, 14, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded planning.ApprovalRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.RecordID != original.RecordID {
		t.Errorf("RecordID: got %q, want %q", decoded.RecordID, original.RecordID)
	}
	if decoded.Action != original.Action {
		t.Errorf("Action: got %q, want %q", decoded.Action, original.Action)
	}
	if decoded.PreviousStatus != original.PreviousStatus {
		t.Errorf("PreviousStatus: got %q, want %q", decoded.PreviousStatus, original.PreviousStatus)
	}
	if decoded.NewStatus != original.NewStatus {
		t.Errorf("NewStatus: got %q, want %q", decoded.NewStatus, original.NewStatus)
	}
}

func TestExecutionMode_Values(t *testing.T) {
	modes := []planning.ExecutionMode{
		planning.ModeObserve,
		planning.ModeRecommend,
		planning.ModeDryRun,
		planning.ModeExecute,
	}

	for _, m := range modes {
		if m == "" {
			t.Error("execution mode should not be empty")
		}
	}
}

func TestPlanWithExecutionMode_JSONContract(t *testing.T) {
	p := &planning.Plan{
		SchemaVersion: "v1",
		PlanID:        "plan_mode_test",
		Status:        planning.PlanApproved,
		ExecutionMode: planning.ModeDryRun,
		Intent: planning.PlanIntent{
			Source:     "command",
			RawInput:   "test",
			ActionType: "test",
		},
		Scope: planning.PlanScope{
			Owner: "org",
			Repo:  "repo",
		},
		RiskLevel: planning.RiskLow,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !containsStr(raw, `"execution_mode"`) {
		t.Error("JSON missing execution_mode field")
	}
	if !containsStr(raw, `"dry_run"`) {
		t.Error("JSON missing dry_run value for execution_mode")
	}
}

func TestApprovalAction_AllValues(t *testing.T) {
	actions := []planning.ApprovalAction{
		planning.ActionApprove,
		planning.ActionReject,
		planning.ActionEdit,
		planning.ActionDefer,
	}

	seen := make(map[planning.ApprovalAction]bool)
	for _, a := range actions {
		if a == "" {
			t.Error("approval action should not be empty")
		}
		if seen[a] {
			t.Errorf("duplicate action: %s", a)
		}
		seen[a] = true
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
