package policy

import (
	"context"
	"testing"

	"github.com/your-org/gitdex/internal/planning"
)

func TestEvaluate_LowRisk_Allowed(t *testing.T) {
	eng := NewDefaultEngine()
	plan := &planning.Plan{
		RiskLevel: planning.RiskLow,
		Scope:     planning.PlanScope{Owner: "o", Repo: "r"},
	}

	result, err := eng.Evaluate(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verdict != planning.VerdictAllowed {
		t.Errorf("expected allowed, got %s", result.Verdict)
	}
	if result.Explanation == "" {
		t.Error("expected non-empty explanation")
	}
}

func TestEvaluate_HighRisk_Escalated(t *testing.T) {
	eng := NewDefaultEngine()
	plan := &planning.Plan{
		RiskLevel: planning.RiskHigh,
		Scope:     planning.PlanScope{Owner: "o", Repo: "r", Branch: "main"},
	}

	result, err := eng.Evaluate(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verdict != planning.VerdictEscalated {
		t.Errorf("expected escalated, got %s", result.Verdict)
	}
	if len(result.RequiredApprovals) == 0 {
		t.Error("expected required approvals for escalated verdict")
	}
}

func TestEvaluate_CriticalRisk_Blocked(t *testing.T) {
	eng := NewDefaultEngine()
	plan := &planning.Plan{
		RiskLevel: planning.RiskCritical,
		Scope:     planning.PlanScope{Owner: "o", Repo: "r", Environment: "production"},
	}

	result, err := eng.Evaluate(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verdict != planning.VerdictBlocked {
		t.Errorf("expected blocked, got %s", result.Verdict)
	}
	if result.Explanation == "" {
		t.Error("expected operator-readable explanation for blocked verdict")
	}
}

func TestEvaluate_NilPlan(t *testing.T) {
	eng := NewDefaultEngine()
	_, err := eng.Evaluate(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil plan")
	}
}

func TestEvaluate_RiskFactors_ProtectedBranch(t *testing.T) {
	eng := NewDefaultEngine()
	plan := &planning.Plan{
		RiskLevel: planning.RiskHigh,
		Scope:     planning.PlanScope{Owner: "o", Repo: "r", Branch: "main"},
	}

	result, _ := eng.Evaluate(context.Background(), plan)
	found := false
	for _, f := range result.RiskFactors {
		if f == "targets protected branch" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'targets protected branch' risk factor")
	}
}

func TestEvaluate_RiskFactors_Production(t *testing.T) {
	eng := NewDefaultEngine()
	plan := &planning.Plan{
		RiskLevel: planning.RiskCritical,
		Scope:     planning.PlanScope{Owner: "o", Repo: "r", Environment: "production"},
	}

	result, _ := eng.Evaluate(context.Background(), plan)
	found := false
	for _, f := range result.RiskFactors {
		if f == "targets production environment" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'targets production environment' risk factor")
	}
}

func TestEvaluate_Explanation_HumanReadable(t *testing.T) {
	eng := NewDefaultEngine()
	verdicts := []struct {
		risk planning.RiskLevel
		want planning.PolicyVerdict
	}{
		{planning.RiskLow, planning.VerdictAllowed},
		{planning.RiskHigh, planning.VerdictEscalated},
		{planning.RiskCritical, planning.VerdictBlocked},
	}

	for _, v := range verdicts {
		plan := &planning.Plan{RiskLevel: v.risk, Scope: planning.PlanScope{Owner: "o", Repo: "r"}}
		result, err := eng.Evaluate(context.Background(), plan)
		if err != nil {
			t.Fatalf("risk %s: unexpected error: %v", v.risk, err)
		}
		if result.Explanation == "" {
			t.Errorf("risk %s: expected non-empty explanation", v.risk)
		}
		if result.Reason == "" {
			t.Errorf("risk %s: expected non-empty reason", v.risk)
		}
	}
}
