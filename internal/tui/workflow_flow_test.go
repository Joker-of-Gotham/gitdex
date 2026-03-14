package tui

import (
	"testing"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	gitplatform "github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func TestMaterializeWorkflowFlowBuildsPendingSteps(t *testing.T) {
	plan := &prompt.WorkflowOrchestration{
		WorkflowID:    "pages_setup",
		WorkflowLabel: "Pages setup",
		Goal:          "Prepare Pages rollout",
		Steps: []prompt.WorkflowOrchestrationStep{
			{Title: "Inspect latest build", Capability: "pages", Flow: "inspect", Query: map[string]string{"view": "latest_build"}},
			{Title: "Inspect branch rules", Capability: "branch_rulesets", Flow: "inspect", Query: map[string]string{"view": "branch_rules", "branch": "main"}},
		},
	}

	flow := materializeWorkflowFlow(gitplatform.PlatformGitHub, plan)
	if flow == nil || len(flow.Steps) != 2 {
		t.Fatalf("expected workflow flow with 2 steps, got %+v", flow)
	}
	if flow.RunID == "" {
		t.Fatalf("expected flow run id to be set")
	}
	if flow.CheckpointVersion == 0 {
		t.Fatalf("expected checkpoint version to be initialized")
	}
	if flow.Steps[0].Status != workflowFlowPending || flow.Steps[1].Status != workflowFlowPending {
		t.Fatalf("expected pending steps, got %+v", flow.Steps)
	}
	if !flow.Steps[0].Policy.SchedulerSafe {
		t.Fatalf("expected inspect step to be scheduler-safe")
	}
}

func TestWorkflowFlowReconcilesPlatformSuggestions(t *testing.T) {
	m := NewModel()
	m.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "pages_setup",
		WorkflowLabel: "Pages setup",
		Goal:          "Prepare Pages rollout",
		Steps: []prompt.WorkflowOrchestrationStep{
			{Title: "Inspect latest build", Capability: "pages", Flow: "inspect", Query: map[string]string{"view": "latest_build"}},
		},
	}
	m.syncWorkflowFlowFromPlan()
	m.suggestions = []git.Suggestion{{
		Action:      "Inspect latest Pages build",
		Reason:      "Need current publish state",
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "pages",
			Flow:         "inspect",
			Query:        map[string]string{"view": "latest_build"},
		},
	}}

	m.reconcileWorkflowFlowSuggestions()
	if m.workflowFlow == nil || len(m.workflowFlow.Steps) != 1 {
		t.Fatalf("unexpected workflow flow %+v", m.workflowFlow)
	}
	if m.workflowFlow.Steps[0].Status != workflowFlowSuggested {
		t.Fatalf("expected suggested status, got %s", m.workflowFlow.Steps[0].Status)
	}
}

func TestWorkflowFlowTracksPlatformExecutionResult(t *testing.T) {
	m := NewModel()
	m.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "pages_setup",
		WorkflowLabel: "Pages setup",
		Goal:          "Prepare Pages rollout",
		Steps: []prompt.WorkflowOrchestrationStep{
			{Title: "Inspect latest build", Capability: "pages", Flow: "inspect", Query: map[string]string{"view": "latest_build"}},
		},
	}
	m.syncWorkflowFlowFromPlan()
	op := &git.PlatformExecInfo{
		CapabilityID: "pages",
		Flow:         "inspect",
		Query:        map[string]string{"view": "latest_build"},
	}

	m.markWorkflowFlowRunning(op)
	if m.workflowFlow.Steps[0].Status != workflowFlowRunning {
		t.Fatalf("expected running status, got %s", m.workflowFlow.Steps[0].Status)
	}
	m.markWorkflowFlowResult(op, false, "inspect completed")
	if m.workflowFlow.Steps[0].Status != workflowFlowDone {
		t.Fatalf("expected done status, got %s", m.workflowFlow.Steps[0].Status)
	}
}

func TestWorkflowFlowSchedulesRetryBeforeDeadLetter(t *testing.T) {
	m := NewModel()
	m.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "pages_setup",
		WorkflowLabel: "Pages setup",
		Goal:          "Prepare Pages rollout",
		Steps: []prompt.WorkflowOrchestrationStep{
			{Title: "Inspect latest build", Capability: "pages", Flow: "inspect", Query: map[string]string{"view": "latest_build"}},
		},
	}
	m.syncWorkflowFlowFromPlan()
	op := &git.PlatformExecInfo{
		CapabilityID: "pages",
		Flow:         "inspect",
		Query:        map[string]string{"view": "latest_build"},
	}

	m.markWorkflowFlowRunning(op)
	m.markWorkflowFlowResult(op, true, "temporary failure")
	if m.workflowFlow.Steps[0].Status != workflowFlowRetrying {
		t.Fatalf("expected retrying status, got %s", m.workflowFlow.Steps[0].Status)
	}
	if m.workflowFlow.Steps[0].NextRetryAt.IsZero() {
		t.Fatalf("expected next retry time to be set")
	}
	m.workflowFlow.Steps[0].NextRetryAt = time.Now().Add(-time.Second)
	if !m.promoteDueWorkflowRetries(time.Now()) {
		t.Fatalf("expected due retry promotion")
	}
	if m.workflowFlow.Steps[0].Status != workflowFlowReady {
		t.Fatalf("expected ready status after due retry, got %s", m.workflowFlow.Steps[0].Status)
	}
}

func TestWorkflowFlowDeadLettersAfterRetryBudget(t *testing.T) {
	m := NewModel()
	m.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "pages_setup",
		WorkflowLabel: "Pages setup",
		Goal:          "Prepare Pages rollout",
		Steps: []prompt.WorkflowOrchestrationStep{
			{Title: "Mutate pages", Capability: "pages", Flow: "mutate", Operation: "update"},
		},
	}
	m.syncWorkflowFlowFromPlan()
	op := &git.PlatformExecInfo{
		CapabilityID: "pages",
		Flow:         "mutate",
		Operation:    "update",
	}

	for i := 0; i < 2; i++ {
		m.markWorkflowFlowRunning(op)
		m.markWorkflowFlowResult(op, true, "hard failure")
	}
	if m.workflowFlow.Steps[0].Status != workflowFlowDeadLetter {
		t.Fatalf("expected deadletter status, got %s", m.workflowFlow.Steps[0].Status)
	}
	if m.workflowFlow.Steps[0].DeadLetter == "" {
		t.Fatalf("expected dead letter reason to be recorded")
	}
	if len(m.workflowFlow.DeadLetterEntries) == 0 {
		t.Fatalf("expected dead-letter queue entry to be recorded")
	}
}

func TestWorkflowFlowTracksSelectionHealthAndApproval(t *testing.T) {
	m := NewModel()
	m.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "release_flow",
		WorkflowLabel: "Release flow",
		Goal:          "Publish release",
		Steps: []prompt.WorkflowOrchestrationStep{
			{Title: "Publish release", Capability: "release", Flow: "mutate", Operation: "publish_draft"},
			{Title: "Validate release", Capability: "release", Flow: "validate", Operation: "publish_draft"},
		},
	}
	m.syncWorkflowFlowFromPlan()
	if m.workflowFlow.Health != "approval_pending" {
		t.Fatalf("expected approval_pending health, got %s", m.workflowFlow.Health)
	}
	if m.workflowFlow.SelectedStepIndex != 0 {
		t.Fatalf("expected first step to be selected, got %d", m.workflowFlow.SelectedStepIndex)
	}
	if _, ok := m.moveWorkflowStepSelection(1); !ok {
		t.Fatal("expected step selection to move")
	}
	if m.workflowFlow.SelectedStepIndex != 1 {
		t.Fatalf("expected second step to be selected, got %d", m.workflowFlow.SelectedStepIndex)
	}

	op := &git.PlatformExecInfo{
		CapabilityID: "release",
		Flow:         "mutate",
		Operation:    "publish_draft",
	}
	m.markWorkflowFlowRunning(op)
	m.markWorkflowFlowResult(op, true, "publish failed")
	if m.workflowFlow.Health != "approval_pending" {
		t.Fatalf("expected approval_pending after retry scheduling, got %s", m.workflowFlow.Health)
	}
	if m.workflowFlow.NextRetryAt.IsZero() {
		t.Fatal("expected next retry to be populated")
	}
	if m.workflowFlow.NextRetryStep != "Publish release" {
		t.Fatalf("expected next retry step label, got %q", m.workflowFlow.NextRetryStep)
	}
	if m.workflowFlow.ApprovalDetail == "" {
		t.Fatal("expected approval detail to be populated")
	}
}

func TestWorkflowFlowRecordsLedgerRefsAndCompensation(t *testing.T) {
	m := NewModel()
	m.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "pull_request_flow",
		WorkflowLabel: "PR flow",
		Goal:          "Ship PR",
		Steps: []prompt.WorkflowOrchestrationStep{
			{Title: "Merge PR", Capability: "pull_request", Flow: "rollback", Operation: "disable_auto_merge"},
		},
	}
	m.syncWorkflowFlowFromPlan()
	op := &git.PlatformExecInfo{
		CapabilityID: "pull_request",
		Flow:         "rollback",
		Operation:    "disable_auto_merge",
	}

	m.recordWorkflowFlowLedger(op, "ledger-1")
	m.markWorkflowFlowRunning(op)
	m.markWorkflowFlowResult(op, false, "rollback applied")
	if m.workflowFlow.Steps[0].Status != workflowFlowCompensated {
		t.Fatalf("expected compensated status, got %s", m.workflowFlow.Steps[0].Status)
	}
	if len(m.workflowFlow.Steps[0].LedgerRefs) != 1 || m.workflowFlow.Steps[0].LedgerRefs[0] != "ledger-1" {
		t.Fatalf("expected ledger ref to be recorded, got %+v", m.workflowFlow.Steps[0].LedgerRefs)
	}
	if len(m.workflowFlow.CompensationRefs) != 1 {
		t.Fatalf("expected compensation ref to be recorded")
	}
}

func TestWorkflowFlowAckAndSkipDeadLetter(t *testing.T) {
	m := NewModel()
	m.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "pages_setup",
		WorkflowLabel: "Pages setup",
		Goal:          "Prepare Pages rollout",
		Steps: []prompt.WorkflowOrchestrationStep{
			{Title: "Mutate pages", Capability: "pages", Flow: "mutate", Operation: "update"},
		},
	}
	m.syncWorkflowFlowFromPlan()
	op := &git.PlatformExecInfo{
		CapabilityID: "pages",
		Flow:         "mutate",
		Operation:    "update",
	}
	for i := 0; i < 2; i++ {
		m.markWorkflowFlowRunning(op)
		m.markWorkflowFlowResult(op, true, "hard failure")
	}
	if _, ok := m.ackDeadLetterWorkflowStep(); !ok {
		t.Fatalf("expected dead-letter ack to succeed")
	}
	if !m.workflowFlow.DeadLetterEntries[0].Acked {
		t.Fatalf("expected dead-letter entry to be acknowledged")
	}
	if _, ok := m.skipDeadLetterWorkflowStep("operator skip"); !ok {
		t.Fatalf("expected dead-letter skip to succeed")
	}
	if m.workflowFlow.Steps[0].Status != workflowFlowSkipped {
		t.Fatalf("expected skipped status, got %s", m.workflowFlow.Steps[0].Status)
	}
}
