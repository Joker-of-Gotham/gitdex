package tui

import (
	"testing"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

func TestApplyDueScheduledAutomationSetsGoalAndWorkflowPlan(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.automation = config.AutomationConfig{
		Enabled: true,
		Schedules: []config.AutomationSchedule{
			{
				ID:         "advanced-security-hourly",
				Enabled:    true,
				WorkflowID: "advanced_security",
				Interval:   300,
			},
		},
	}
	m.scheduleLastRun = map[string]time.Time{}
	m.workflows = []workflowDefinition{{
		ID:           "advanced_security",
		Label:        "Advanced Security",
		Goal:         "Audit GitHub Advanced Security posture",
		Capabilities: []string{"advanced_security"},
		Prefill: []workflowPrefillDefinition{{
			CapabilityID: "advanced_security",
			Flow:         "inspect",
			Query:        map[string]string{"view": "summary"},
		}},
	}}
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       "git@github.com:Joker-of-Gotham/gitdex.git",
			FetchURL:      "git@github.com:Joker-of-Gotham/gitdex.git",
			PushURLValid:  true,
			FetchURLValid: true,
		}},
	}

	updated, ok := m.applyDueScheduledAutomation(time.Now())
	if !ok {
		t.Fatal("expected scheduled automation to trigger")
	}
	if updated.session.ActiveGoal != "Audit GitHub Advanced Security posture" {
		t.Fatalf("unexpected active goal %q", updated.session.ActiveGoal)
	}
	if updated.workflowPlan == nil || len(updated.workflowPlan.Steps) != 1 {
		t.Fatalf("expected workflow plan, got %+v", updated.workflowPlan)
	}
	if updated.workflowFlow == nil || len(updated.workflowFlow.Steps) != 1 {
		t.Fatalf("expected workflow flow, got %+v", updated.workflowFlow)
	}
	if updated.workflowPlan.Steps[0].Capability != "advanced_security" {
		t.Fatalf("unexpected workflow capability %q", updated.workflowPlan.Steps[0].Capability)
	}
}

func TestApplyDueScheduledAutomationHonorsInterval(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.automation = config.AutomationConfig{
		Enabled: true,
		Schedules: []config.AutomationSchedule{{
			ID:       "health",
			Enabled:  true,
			Goal:     "Run a repository health check",
			Interval: 600,
		}},
	}
	m.scheduleLastRun = map[string]time.Time{
		"health": time.Now(),
	}

	if _, ok := m.applyDueScheduledAutomation(time.Now().Add(10 * time.Second)); ok {
		t.Fatal("did not expect scheduled automation before interval elapsed")
	}
}
