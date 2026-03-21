package compiler

import (
	"context"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/planning/intent"
)

func TestCompile_BasicGoal(t *testing.T) {
	c := New("org", "repo")
	i := intent.NewCommandIntent("update dependencies", "plan", nil)

	plan, err := c.Compile(context.Background(), i)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(plan.PlanID, "plan_") {
		t.Errorf("plan ID missing prefix: %s", plan.PlanID)
	}
	if !strings.HasPrefix(plan.TaskID, "task_") {
		t.Errorf("task ID missing prefix: %s", plan.TaskID)
	}
	if plan.Status != planning.PlanDraft {
		t.Errorf("expected draft status, got %s", plan.Status)
	}
	if plan.Scope.Owner != "org" {
		t.Errorf("expected owner org, got %s", plan.Scope.Owner)
	}
	if len(plan.Steps) == 0 {
		t.Error("expected at least one step")
	}
}

func TestCompile_EmptyInput(t *testing.T) {
	c := New("org", "repo")
	i := intent.NewCommandIntent("", "plan", nil)

	_, err := c.Compile(context.Background(), i)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestCompile_HighRisk_MainBranch(t *testing.T) {
	c := New("org", "repo")
	params := map[string]string{"branch": "main"}
	i := intent.NewCommandIntent("deploy to main", "deploy", params)

	plan, err := c.Compile(context.Background(), i)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.RiskLevel != planning.RiskHigh {
		t.Errorf("expected high risk for main branch, got %s", plan.RiskLevel)
	}
}

func TestCompile_CriticalRisk_Production(t *testing.T) {
	c := New("org", "repo")
	params := map[string]string{"environment": "production"}
	i := intent.NewCommandIntent("deploy to prod", "deploy", params)

	plan, err := c.Compile(context.Background(), i)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.RiskLevel != planning.RiskCritical {
		t.Errorf("expected critical risk for production, got %s", plan.RiskLevel)
	}
}

func TestCompile_LowRisk_Default(t *testing.T) {
	c := New("org", "repo")
	i := intent.NewCommandIntent("list branches", "list", nil)

	plan, err := c.Compile(context.Background(), i)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.RiskLevel != planning.RiskLow {
		t.Errorf("expected low risk, got %s", plan.RiskLevel)
	}
}

func TestCompile_ChatIntent(t *testing.T) {
	c := New("org", "repo")
	i := intent.NewChatIntent("help me clean up stale branches")

	plan, err := c.Compile(context.Background(), i)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Intent.Source != "chat" {
		t.Errorf("expected chat source, got %s", plan.Intent.Source)
	}
}
