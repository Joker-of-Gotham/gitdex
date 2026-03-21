package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/planning"
)

func TestPlanContract_JSONFieldNames_SnakeCase(t *testing.T) {
	plan := &planning.Plan{
		SchemaVersion: "v1",
		PlanID:        "plan_contract_test",
		TaskID:        "task_contract_test",
		Status:        planning.PlanDraft,
		Intent:        planning.PlanIntent{Source: "command", RawInput: "test", ActionType: "test"},
		Scope:         planning.PlanScope{Owner: "o", Repo: "r"},
		RiskLevel:     planning.RiskLow,
		PolicyResult: &planning.PolicyResult{
			Verdict:     planning.VerdictAllowed,
			Reason:      "test",
			Explanation: "test",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(data)
	requiredFields := []string{
		"schema_version", "plan_id", "task_id", "status",
		"risk_level", "created_at", "updated_at",
		"raw_input", "action_type",
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, "\""+field+"\"") {
			t.Errorf("JSON missing snake_case field %q", field)
		}
	}
}

func TestPlanContract_TimestampRFC3339(t *testing.T) {
	plan := &planning.Plan{
		PlanID:    "plan_ts",
		CreatedAt: time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if !strings.Contains(string(data), "2026-03-18T12:00:00Z") {
		t.Errorf("timestamp not in RFC3339 format: %s", string(data))
	}
}

func TestPlanContract_PolicyVerdictValues(t *testing.T) {
	verdicts := []planning.PolicyVerdict{
		planning.VerdictAllowed,
		planning.VerdictEscalated,
		planning.VerdictBlocked,
		planning.VerdictDegraded,
	}

	for _, v := range verdicts {
		if strings.ToLower(string(v)) != string(v) {
			t.Errorf("verdict %q should be lower_snake_case", v)
		}
	}
}

func TestPlanContract_StatusValues(t *testing.T) {
	statuses := []planning.PlanStatus{
		planning.PlanDraft,
		planning.PlanReviewRequired,
		planning.PlanApproved,
		planning.PlanBlocked,
		planning.PlanExecuting,
		planning.PlanCompleted,
	}

	for _, s := range statuses {
		if strings.ToLower(string(s)) != string(s) {
			t.Errorf("status %q should be lower_snake_case", s)
		}
	}
}
