package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func TestBuildFlowReportIncludesAutomationRecoveryState(t *testing.T) {
	m := NewModel()
	m.session.ActiveGoal = "Restore unattended execution"
	m.automationObserveOnly = true
	m.automationFailures = map[string]int{"pages": 2}
	m.lastEscalation = time.Unix(1700000100, 0).UTC()
	m.lastRecovery = time.Unix(1700000200, 0).UTC()
	m.workflowFlow = &workflowFlowState{
		WorkflowID:        "pages_setup",
		WorkflowLabel:     "Pages Setup",
		SelectedStepIndex: 0,
		Steps: []workflowFlowStep{{
			Index:  0,
			Status: workflowFlowDeadLetter,
			Step: prompt.WorkflowOrchestrationStep{
				Title:      "Validate domain",
				Capability: "pages",
				Flow:       "validate",
			},
		}},
	}

	report := m.buildFlowReport()
	if !report.ObserveOnly {
		t.Fatal("expected observe-only state in flow report")
	}
	if report.EscalatedAt.IsZero() || report.RecoveredAt.IsZero() {
		t.Fatal("expected escalation and recovery timestamps in flow report")
	}
	if report.AutomationFailures["pages"] != 2 {
		t.Fatalf("expected failure counters in report, got %+v", report.AutomationFailures)
	}
	if !strings.Contains(report.RecoveryPath, "recover-auto") {
		t.Fatalf("expected recovery path hint, got %q", report.RecoveryPath)
	}
	markdown := renderFlowReportMarkdown(report)
	if !strings.Contains(markdown, "Recovery path:") {
		t.Fatalf("expected recovery path in markdown, got %s", markdown)
	}
}

func TestRenderLedgerMarkdownIncludesRollbackAndBoundary(t *testing.T) {
	markdown := renderLedgerMarkdown([]platform.MutationLedgerEntry{{
		At:           time.Unix(1700000000, 0).UTC(),
		CapabilityID: "pages",
		Flow:         "mutate",
		Operation:    "update",
		ExecMeta: platform.ExecutionMeta{
			Adapter:        platform.AdapterBrowser,
			Coverage:       platform.CoveragePartial,
			Rollback:       platform.RollbackCompensating,
			BoundaryReason: "external DNS and certificate state remain outside repository control",
		},
		Metadata: map[string]string{
			"browser_driver":             "playwright",
			"manual_completion_required": "true",
		},
		DiagnosticDecision: platform.DiagnosticAllow,
		Summary:            "browser-backed mutation queued for operator follow-up",
	}})
	if !strings.Contains(markdown, "Rollback") || !strings.Contains(markdown, "Boundary") {
		t.Fatalf("expected ledger markdown headers to include rollback and boundary columns, got %s", markdown)
	}
	if !strings.Contains(markdown, "external DNS") {
		t.Fatalf("expected boundary reason in ledger markdown, got %s", markdown)
	}
	if !strings.Contains(markdown, "driver=playwright") {
		t.Fatalf("expected adapter detail in ledger markdown, got %s", markdown)
	}
}

func TestRenderOperatorReportMarkdownIncludesRecoveryContext(t *testing.T) {
	report := OperatorReport{
		GeneratedAt:  time.Unix(1700000300, 0).UTC(),
		ObserveOnly:  true,
		EscalatedAt:  time.Unix(1700000100, 0).UTC(),
		RecoveredAt:  time.Unix(1700000200, 0).UTC(),
		RecoveryPath: "H recover-auto -> R resume-step -> X retry-step -> C compensate-step",
		Flow: FlowReport{
			Label:              "Pages Setup",
			Health:             "attention_required",
			Approval:           "approval_required",
			AutomationFailures: map[string]int{"pages": 3},
		},
	}

	markdown := renderOperatorReportMarkdown(report)
	if !strings.Contains(markdown, "Escalated:") || !strings.Contains(markdown, "Recovered:") {
		t.Fatalf("expected escalation and recovery timestamps in operator report markdown, got %s", markdown)
	}
	if !strings.Contains(markdown, "Automation failure counter pages: 3") {
		t.Fatalf("expected failure counter in operator report markdown, got %s", markdown)
	}
	if !strings.Contains(markdown, "Recovery path:") {
		t.Fatalf("expected recovery path in operator report markdown, got %s", markdown)
	}
}
