package autonomy

import (
	"context"
	"testing"
	"time"
)

func TestDefaultCruiseConfig(t *testing.T) {
	cfg := DefaultCruiseConfig()
	if cfg.Enabled {
		t.Error("default should be disabled")
	}
	if cfg.Interval != 30*time.Minute {
		t.Errorf("expected 30m interval, got %v", cfg.Interval)
	}
	if cfg.AutoExecuteThreshold != RiskLow {
		t.Errorf("expected low auto threshold, got %v", cfg.AutoExecuteThreshold)
	}
}

func TestCruiseEngine_Lifecycle(t *testing.T) {
	guard := NewGuardrails()
	executor := NewPlanExecutor(guard)
	reporter := NewReporter(10)
	planner := NewPlanner(nil, nil)

	engine := NewCruiseEngine(DefaultCruiseConfig(), planner, guard, executor, reporter)

	if engine.State() != CruiseIdle {
		t.Errorf("expected idle, got %s", engine.State())
	}

	engine.Pause()
	if engine.State() != CruiseIdle {
		t.Error("pause from idle should stay idle")
	}

	engine.Resume()
	if engine.State() != CruiseIdle {
		t.Error("resume from idle should stay idle")
	}

	if engine.CycleCount() != 0 {
		t.Errorf("expected 0 cycles, got %d", engine.CycleCount())
	}
}

func TestGuardrails_EvaluateRisk(t *testing.T) {
	g := NewGuardrails()

	plan := ActionPlan{
		Steps: []PlanStep{
			{Action: "git.branch.delete", Args: map[string]string{"name": "old"}},
		},
	}
	if risk := g.EvaluateRisk(plan); risk != RiskLow {
		t.Errorf("branch delete should be low risk, got %v", risk)
	}

	plan.Steps = []PlanStep{
		{Action: "git.push"},
	}
	if risk := g.EvaluateRisk(plan); risk != RiskHigh {
		t.Errorf("push should be high risk, got %v", risk)
	}

	plan.Steps = []PlanStep{
		{Action: "git.push.force"},
	}
	if risk := g.EvaluateRisk(plan); risk != RiskCritical {
		t.Errorf("force push should be critical, got %v", risk)
	}
}

func TestGuardrails_CheckPolicy(t *testing.T) {
	g := NewGuardrails()

	plan := ActionPlan{
		Steps: []PlanStep{{Action: "git.push.force"}},
	}
	allowed, reason := g.CheckPolicy(plan)
	if allowed {
		t.Error("force push should be blocked")
	}
	if reason == "" {
		t.Error("expected block reason")
	}

	plan.Steps = []PlanStep{{Action: "git.add"}}
	allowed, _ = g.CheckPolicy(plan)
	if !allowed {
		t.Error("git add should be allowed")
	}
}

func TestGuardrails_CustomRisk(t *testing.T) {
	g := NewGuardrails()
	g.SetActionRisk("custom.action", RiskCritical)

	plan := ActionPlan{Steps: []PlanStep{{Action: "custom.action"}}}
	if risk := g.EvaluateRisk(plan); risk != RiskCritical {
		t.Errorf("custom action should be critical, got %v", risk)
	}
}

func TestGuardrails_CustomBlock(t *testing.T) {
	g := NewGuardrails()
	g.BlockAction("my.dangerous.action", "too dangerous")

	plan := ActionPlan{Steps: []PlanStep{{Action: "my.dangerous.action"}}}
	allowed, reason := g.CheckPolicy(plan)
	if allowed {
		t.Error("should be blocked")
	}
	if reason != "too dangerous" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestRiskLevel_String(t *testing.T) {
	tests := []struct {
		level RiskLevel
		want  string
	}{
		{RiskLow, "low"},
		{RiskMedium, "medium"},
		{RiskHigh, "high"},
		{RiskCritical, "critical"},
		{RiskLevel(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestParseRiskLevel(t *testing.T) {
	tests := []struct {
		input string
		want  RiskLevel
	}{
		{"low", RiskLow},
		{"medium", RiskMedium},
		{"HIGH", RiskHigh},
		{"Critical", RiskCritical},
		{"invalid", RiskHigh},
	}
	for _, tt := range tests {
		if got := ParseRiskLevel(tt.input); got != tt.want {
			t.Errorf("ParseRiskLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParsePlans(t *testing.T) {
	raw := `[{"description":"clean merged branches","steps":[{"order":1,"action":"git.branch.delete","args":{"name":"feature/old"},"reversible":true,"description":"delete old branch"}],"risk_level":"low","rationale":"branch already merged"}]`

	plans, err := ParsePlans(raw)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Description != "clean merged branches" {
		t.Errorf("unexpected description: %s", plans[0].Description)
	}
	if len(plans[0].Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plans[0].Steps))
	}
}

func TestParsePlans_Empty(t *testing.T) {
	plans, err := ParsePlans("[]")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}
}

func TestReporter(t *testing.T) {
	r := NewReporter(3)
	if r.Count() != 0 {
		t.Errorf("expected 0 reports, got %d", r.Count())
	}

	r.Add(CruiseReport{CycleID: "c1"})
	r.Add(CruiseReport{CycleID: "c2"})
	if r.Count() != 2 {
		t.Errorf("expected 2 reports, got %d", r.Count())
	}

	latest := r.Latest()
	if latest == nil || latest.CycleID != "c2" {
		t.Error("latest should be c2")
	}

	r.Add(CruiseReport{CycleID: "c3"})
	r.Add(CruiseReport{CycleID: "c4"})
	if r.Count() != 3 {
		t.Errorf("expected 3 reports (max), got %d", r.Count())
	}
}

func TestFormatReport(t *testing.T) {
	report := CruiseReport{
		CycleID:   "test-1",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Second),
		Executed: []ExecutedAction{
			{
				Plan:   ActionPlan{Description: "cleaned branches"},
				Result: ExecutionResult{Success: true},
			},
		},
	}

	text := FormatReport(report)
	if text == "" {
		t.Error("expected non-empty report")
	}
	if !containsStr(text, "test-1") {
		t.Error("should contain cycle ID")
	}
	if !containsStr(text, "cleaned branches") {
		t.Error("should contain executed action")
	}
}

func TestToolRegistry(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(Tool{
		Name:        "test.action",
		Description: "test action",
		Handler: func(ctx context.Context, args map[string]string) (string, error) {
			return "ok", nil
		},
	})

	if _, ok := reg.Get("test.action"); !ok {
		t.Error("should find registered tool")
	}
	if _, ok := reg.Get("nonexistent"); ok {
		t.Error("should not find unregistered tool")
	}

	tools := reg.List()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	prompt := reg.GenerateToolPrompt()
	if !containsStr(prompt, "test.action") {
		t.Error("prompt should contain tool name")
	}
}

func TestPlanExecutor_Execute(t *testing.T) {
	guard := NewGuardrails()
	exec := NewPlanExecutor(guard)

	exec.RegisterHandler("test.ok", func(ctx context.Context, args map[string]string) (string, error) {
		return "done", nil
	})

	plan := ActionPlan{
		ID: "plan-1",
		Steps: []PlanStep{
			{Order: 1, Action: "test.ok", Args: map[string]string{}},
		},
	}

	result := exec.Execute(context.Background(), plan)
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.StepsRun != 1 {
		t.Errorf("expected 1 step run, got %d", result.StepsRun)
	}
}

func TestPlanExecutor_BlockedByGuardrail(t *testing.T) {
	guard := NewGuardrails()
	exec := NewPlanExecutor(guard)

	plan := ActionPlan{
		ID: "plan-blocked",
		Steps: []PlanStep{
			{Order: 1, Action: "git.push.force", Args: map[string]string{}},
		},
	}

	result := exec.Execute(context.Background(), plan)
	if result.Success {
		t.Error("should fail when blocked")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
