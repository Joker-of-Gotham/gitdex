package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func TestAutomationCheckpointPersistsAndLoads(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.session.ActiveGoal = "Audit security posture"
	m.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "advanced_security",
		WorkflowLabel: "Advanced Security",
		Goal:          "Audit security posture",
	}
	m.workflowFlow = &workflowFlowState{
		WorkflowID:    "advanced_security",
		WorkflowLabel: "Advanced Security",
		Goal:          "Audit security posture",
		Steps: []workflowFlowStep{{
			Index:  0,
			Status: workflowFlowSuggested,
			Step: prompt.WorkflowOrchestrationStep{
				Title:      "Inspect security summary",
				Capability: "advanced_security",
				Flow:       "inspect",
			},
		}},
	}
	m.scheduleLastRun["advanced-security-hourly"] = time.Unix(1700000000, 0).UTC()
	m.automationLocks = map[string]string{"pages:main": "pages / mutate"}
	m.automationFailures = map[string]int{"pages": 2}
	m.automationObserveOnly = true
	m.lastEscalation = time.Unix(1700000100, 0).UTC()
	m.lastRecovery = time.Unix(1700000200, 0).UTC()
	m.mutationLedger = []platform.MutationLedgerEntry{{
		ID:           "ledger-1",
		CapabilityID: "advanced_security",
		Flow:         "inspect",
		Summary:      "security summary inspected",
	}}
	m.persistAutomationCheckpoint()

	state := loadAutomationCheckpoint()
	if state.ActiveGoal != "Audit security posture" {
		t.Fatalf("unexpected active goal %q", state.ActiveGoal)
	}
	if state.Workflow == nil || state.Workflow.WorkflowID != "advanced_security" {
		t.Fatalf("unexpected workflow state %+v", state.Workflow)
	}
	if state.Flow == nil || len(state.Flow.Steps) != 1 || state.Flow.Steps[0].Step.Capability != "advanced_security" {
		t.Fatalf("unexpected flow state %+v", state.Flow)
	}
	if state.ScheduleLastRun["advanced-security-hourly"].IsZero() {
		t.Fatal("expected persisted schedule timestamp")
	}
	if state.AutomationLocks["pages:main"] != "pages / mutate" {
		t.Fatalf("unexpected automation locks %+v", state.AutomationLocks)
	}
	if state.AutomationFailures["pages"] != 2 {
		t.Fatalf("unexpected automation failures %+v", state.AutomationFailures)
	}
	if !state.ObserveOnly {
		t.Fatal("expected observe-only flag to persist")
	}
	if state.EscalatedAt.IsZero() {
		t.Fatal("expected escalation timestamp")
	}
	if state.RecoveredAt.IsZero() {
		t.Fatal("expected recovery timestamp")
	}
	if len(state.Ledger) != 1 || state.Ledger[0].ID != "ledger-1" {
		t.Fatalf("unexpected ledger state %+v", state.Ledger)
	}
}

func configureAutomationStateEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	t.Setenv("APPDATA", filepath.Join(root, "appdata"))
	dir, err := config.GlobalConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}
