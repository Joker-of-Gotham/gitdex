package planning

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

func TestGeneratePlanID(t *testing.T) {
	id := GeneratePlanID()
	if !strings.HasPrefix(id, "plan_") {
		t.Errorf("plan ID should start with plan_, got %s", id)
	}
	id2 := GeneratePlanID()
	if id == id2 {
		t.Error("two generated plan IDs should be unique")
	}
}

func TestGenerateTaskID(t *testing.T) {
	id := GenerateTaskID()
	if !strings.HasPrefix(id, "task_") {
		t.Errorf("task ID should start with task_, got %s", id)
	}
}

func TestPlan_JSONRoundTrip(t *testing.T) {
	plan := &Plan{
		SchemaVersion: "v1",
		PlanID:        "plan_test123",
		TaskID:        "task_test456",
		Status:        PlanDraft,
		Intent:        PlanIntent{Source: "command", RawInput: "test goal", ActionType: "plan"},
		Scope:         PlanScope{Owner: "org", Repo: "repo", Branch: "main"},
		Steps: []PlanStep{
			{Sequence: 1, Action: "review", Target: "org/repo", Description: "test step", RiskLevel: RiskLow, Reversible: true},
		},
		RiskLevel: RiskLow,
		PolicyResult: &PolicyResult{
			Verdict:     VerdictAllowed,
			Reason:      "low risk",
			Explanation: "safe to proceed",
		},
		CreatedAt: time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded Plan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded.PlanID != plan.PlanID {
		t.Errorf("PlanID mismatch: %s vs %s", decoded.PlanID, plan.PlanID)
	}
	if decoded.Status != plan.Status {
		t.Errorf("Status mismatch: %s vs %s", decoded.Status, plan.Status)
	}
	if decoded.RiskLevel != plan.RiskLevel {
		t.Errorf("RiskLevel mismatch: %s vs %s", decoded.RiskLevel, plan.RiskLevel)
	}
}

func TestPlan_YAMLRoundTrip(t *testing.T) {
	plan := &Plan{
		SchemaVersion: "v1",
		PlanID:        "plan_yaml_test",
		Status:        PlanReviewRequired,
		Intent:        PlanIntent{Source: "chat", RawInput: "fix bugs"},
		Scope:         PlanScope{Owner: "o", Repo: "r"},
		RiskLevel:     RiskMedium,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	data, err := yaml.Marshal(plan)
	if err != nil {
		t.Fatalf("YAML marshal failed: %v", err)
	}

	var decoded Plan
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("YAML unmarshal failed: %v", err)
	}

	if decoded.PlanID != plan.PlanID {
		t.Errorf("PlanID mismatch: %s vs %s", decoded.PlanID, plan.PlanID)
	}
}

func TestPlanStatus_Values(t *testing.T) {
	statuses := []PlanStatus{PlanDraft, PlanReviewRequired, PlanApproved, PlanBlocked, PlanExecuting, PlanCompleted}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("status should not be empty")
		}
	}
}

func TestPolicyVerdict_Values(t *testing.T) {
	verdicts := []PolicyVerdict{VerdictAllowed, VerdictEscalated, VerdictBlocked, VerdictDegraded}
	for _, v := range verdicts {
		if string(v) == "" {
			t.Errorf("verdict should not be empty")
		}
	}
}
