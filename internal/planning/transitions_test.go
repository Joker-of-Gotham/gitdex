package planning

import (
	"testing"
)

func TestValidateTransition_ReviewRequiredActions(t *testing.T) {
	tests := []struct {
		name       string
		action     ApprovalAction
		wantStatus PlanStatus
	}{
		{"approve", ActionApprove, PlanApproved},
		{"reject", ActionReject, PlanBlocked},
		{"defer", ActionDefer, PlanDraft},
		{"edit", ActionEdit, PlanReviewRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateTransition(PlanReviewRequired, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantStatus {
				t.Errorf("got status %q, want %q", got, tt.wantStatus)
			}
		})
	}
}

func TestValidateTransition_BlockedCanOnlyEdit(t *testing.T) {
	got, err := ValidateTransition(PlanBlocked, ActionEdit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != PlanReviewRequired {
		t.Errorf("got status %q, want %q", got, PlanReviewRequired)
	}
}

func TestValidateTransition_BlockedCannotApprove(t *testing.T) {
	_, err := ValidateTransition(PlanBlocked, ActionApprove)
	if err == nil {
		t.Fatal("expected error for approving blocked plan")
	}
	if want := "blocked plans cannot be directly approved"; !contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}

func TestValidateTransition_ApprovedCannotBeModified(t *testing.T) {
	for _, action := range []ApprovalAction{ActionApprove, ActionReject, ActionDefer, ActionEdit} {
		_, err := ValidateTransition(PlanApproved, action)
		if err == nil {
			t.Errorf("expected error for action %q on approved plan", action)
		}
	}
}

func TestValidateTransition_ExecutingCannotBeModified(t *testing.T) {
	_, err := ValidateTransition(PlanExecuting, ActionApprove)
	if err == nil {
		t.Fatal("expected error for action on executing plan")
	}
}

func TestValidateTransition_CompletedCannotBeModified(t *testing.T) {
	_, err := ValidateTransition(PlanCompleted, ActionApprove)
	if err == nil {
		t.Fatal("expected error for action on completed plan")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
