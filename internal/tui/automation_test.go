package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func TestAutoExecuteNextSafeSuggestionSkipsNeedsInputInAutoMode(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.automation = config.AutomationConfig{
		Mode:           config.AutomationModeAuto,
		Enabled:        true,
		Unattended:     true,
		AutoAcceptSafe: true,
		TrustedMode:    false,
		MaxAutoSteps:   4,
	}
	m.session.ActiveGoal = "Inspect repository readiness"
	m.suggestions = []git.Suggestion{
		{Action: "Need branch name", RiskLevel: git.RiskSafe, Interaction: git.NeedsInput, Inputs: []git.InputField{{Key: "branch", Label: "Branch"}}},
		{Action: "Inspect status", Command: []string{"git", "status"}, RiskLevel: git.RiskSafe, Interaction: git.AutoExec},
	}
	m.suggExecState = make([]git.ExecState, len(m.suggestions))
	m.suggExecMsg = make([]string, len(m.suggestions))

	next, _, ok := m.autoExecuteNextSafeSuggestion(false)
	if !ok {
		t.Fatal("expected auto mode to skip NeedsInput and proceed to next suggestion")
	}
	if next.suggExecState[0] != git.ExecSkipped {
		t.Fatalf("expected NeedsInput suggestion to be marked as skipped, got %d", next.suggExecState[0])
	}
}

func TestAutoExecuteStopsAtBlockedInManualMode(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.automation = config.AutomationConfig{
		Mode:           config.AutomationModeManual,
		Enabled:        true,
		Unattended:     true,
		AutoAcceptSafe: true,
		TrustedMode:    false,
		MaxAutoSteps:   4,
	}
	m.suggestions = []git.Suggestion{
		{Action: "Need branch name", RiskLevel: git.RiskSafe, Interaction: git.NeedsInput, Inputs: []git.InputField{{Key: "branch", Label: "Branch"}}},
		{Action: "Inspect status", Command: []string{"git", "status"}, RiskLevel: git.RiskSafe, Interaction: git.AutoExec},
	}
	m.suggExecState = make([]git.ExecState, len(m.suggestions))
	m.suggExecMsg = make([]string, len(m.suggestions))

	_, _, ok := m.autoExecuteNextSafeSuggestion(false)
	if ok {
		t.Fatal("expected manual mode to stop at blocked NeedsInput suggestion")
	}
}

func TestAutomationSafeCommandAllowsGitOperations(t *testing.T) {
	for _, sub := range []string{"push", "pull", "add", "commit", "merge", "rebase", "switch", "checkout", "reset", "fetch"} {
		if !isAutomationSafeCommand([]string{"git", sub}) {
			t.Fatalf("git %s should be allowed", sub)
		}
	}
	if isAutomationSafeCommand([]string{"rm", "-rf", "/"}) {
		t.Fatal("non-git commands should not be allowed")
	}
	if isAutomationSafeCommand([]string{"git"}) {
		t.Fatal("bare git without subcommand should not be allowed")
	}
}

func TestRecordAutomationTransitionsCapturesLocalAndSyncChanges(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.automation = config.AutomationConfig{Mode: config.AutomationModeAssist, Enabled: true}

	prev := &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main", Ahead: 0, Behind: 0},
	}
	next := &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main", Ahead: 1, Behind: 2},
		WorkingTree: []git.FileStatus{{Path: "README.md", WorktreeCode: git.StatusModified}},
	}

	updated := m.recordAutomationTransitions(prev, next)
	entries := updated.opLog.Entries()
	if len(entries) < 2 {
		t.Fatalf("expected transition logs, got %d", len(entries))
	}
}

func TestAutomationWithinMaintenanceWindow(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.automation = config.AutomationConfig{
		MaintenanceWindows: []config.AutomationMaintenanceWindow{{
			Days:  []string{"thu"},
			Start: "09:00",
			End:   "18:00",
		}},
	}
	now := time.Date(2026, 3, 12, 10, 0, 0, 0, time.Local)
	if !m.automationWithinMaintenanceWindow(now) {
		t.Fatal("expected time to fall within maintenance window")
	}
	late := time.Date(2026, 3, 12, 22, 0, 0, 0, time.Local)
	if m.automationWithinMaintenanceWindow(late) {
		t.Fatal("did not expect time outside maintenance window")
	}
}

func TestRecordAutomationOutcomeEscalatesToObserveOnly(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.automation = config.AutomationConfig{
		Escalation: config.AutomationEscalationPolicy{FailureThreshold: 2},
	}

	m.recordAutomationOutcome("pages", false, platform.FailureExecutor)
	if m.automationObserveOnly {
		t.Fatal("should not escalate after first failure")
	}
	m.recordAutomationOutcome("pages", false, platform.FailureExecutor)
	if !m.automationObserveOnly {
		t.Fatal("expected observe-only escalation after repeated failures")
	}
}

func TestRecoverAutomationEscalationClearsObserveOnlyAndFailureCounters(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.automationObserveOnly = true
	m.automationFailures = map[string]int{"pages": 2, "release": 1}
	m.lastEscalation = time.Unix(1700000200, 0).UTC()

	if !m.recoverAutomationEscalation("operator override") {
		t.Fatal("expected recovery path to clear observe-only state")
	}
	if m.automationObserveOnly {
		t.Fatal("expected observe-only mode to be cleared")
	}
	if len(m.automationFailures) != 0 {
		t.Fatalf("expected failure counters to be cleared, got %+v", m.automationFailures)
	}
	if m.lastRecovery.IsZero() {
		t.Fatal("expected recovery timestamp to be recorded")
	}
	if m.lastEscalation.IsZero() {
		t.Fatal("expected escalation timestamp to remain for audit history")
	}
}

func TestShouldAllowAutomationSuggestionBlocksApprovalRequiredPlatformMutation(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.automation = config.AutomationConfig{
		Enabled:        true,
		Unattended:     true,
		AutoAcceptSafe: true,
		ApprovalPolicy: config.AutomationApprovalPolicy{
			RequireForPartial:       true,
			RequireForComposed:      true,
			RequireForAdapterBacked: true,
			RequireForIrreversible:  true,
		},
	}
	allowed, reason := m.shouldAllowAutomationSuggestion(git.Suggestion{
		Action:      "Publish release",
		RiskLevel:   git.RiskSafe,
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "release",
			Flow:         "mutate",
			Operation:    "publish_draft",
		},
	})
	if allowed {
		t.Fatal("expected approval-required platform mutation to be blocked")
	}
	if reason == "" {
		t.Fatal("expected policy reason")
	}
}

func TestShouldAllowAutomationSuggestionBlocksDiagnosticFailures(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.gitState = &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:owner/repo.git",
		}},
	}
	m.automation = config.AutomationConfig{
		Enabled:        true,
		Unattended:     true,
		AutoAcceptSafe: true,
	}
	allowed, reason := m.shouldAllowAutomationSuggestion(git.Suggestion{
		Action:      "Inspect platform state with unresolved placeholders",
		RiskLevel:   git.RiskSafe,
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "pages",
			Flow:         "inspect",
			ResourceID:   "<site_id>",
		},
	})
	if allowed {
		t.Fatal("expected diagnostic-blocked platform mutation to be rejected")
	}
	if reason == "" || !strings.Contains(reason, "diagnostic blocked") {
		t.Fatalf("expected diagnostic reason, got %q", reason)
	}
}

func TestShouldAllowAutomationSuggestionAllowsTrustedPlatformMutation(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.gitState = &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:owner/repo.git",
		}},
	}
	m.automation = config.AutomationConfig{
		Enabled:        true,
		Unattended:     true,
		AutoAcceptSafe: true,
		TrustedMode:    true,
	}
	allowed, reason := m.shouldAllowAutomationSuggestion(git.Suggestion{
		Action:      "Publish release",
		RiskLevel:   git.RiskSafe,
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "release",
			Flow:         "mutate",
			Operation:    "publish_draft",
		},
	})
	if !allowed {
		t.Fatalf("expected trusted platform mutation to be allowed, got reason %q", reason)
	}
}

func TestShouldAllowAutomationSuggestionAllowsOperatorApprovedStep(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.gitState = &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:owner/repo.git",
		}},
	}
	m.automation = config.AutomationConfig{
		Enabled:        true,
		Unattended:     true,
		AutoAcceptSafe: true,
	}
	m.workflowFlow = &workflowFlowState{
		WorkflowID: "release_flow",
		Steps: []workflowFlowStep{{
			Index:  0,
			Status: workflowFlowReady,
			Policy: WorkflowStepPolicy{
				ApprovalRequired: true,
			},
			ApprovalState: "approved",
			Step: prompt.WorkflowOrchestrationStep{
				Title:      "Publish release",
				Capability: "release",
				Flow:       "mutate",
				Operation:  "publish_draft",
			},
		}},
	}

	allowed, reason := m.shouldAllowAutomationSuggestion(git.Suggestion{
		Action:      "Publish release",
		RiskLevel:   git.RiskSafe,
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "release",
			Flow:         "mutate",
			Operation:    "publish_draft",
		},
	})
	if !allowed {
		t.Fatalf("expected operator-approved step to be allowed, got reason %q", reason)
	}
}

func TestBatchRunContinuesAfterCommandFailure(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.batchRunRequested = true
	m.autoSteps = 1
	m.suggestions = []git.Suggestion{
		{Action: "Broken command", Command: []string{"git", "status"}, RiskLevel: git.RiskSafe, Interaction: git.AutoExec},
		{Action: "Inspect status", Command: []string{"git", "status"}, RiskLevel: git.RiskSafe, Interaction: git.AutoExec},
	}
	m.suggExecState = []git.ExecState{git.ExecRunning, git.ExecPending}
	m.suggExecMsg = []string{"running...", ""}
	m.execSuggIdx = 0
	m.automation = config.AutomationConfig{MaxAutoSteps: 8}

	next, cmd := m.Update(commandResultMsg{err: nil, result: &git.ExecutionResult{
		Command: []string{"git", "status"},
		Success: false,
		Stderr:  "simulated failure",
	}})
	updated := next.(Model)

	if cmd == nil {
		t.Fatal("expected batch run to continue with next suggestion")
	}
	if updated.execSuggIdx != 1 {
		t.Fatalf("expected next suggestion to start, got exec index %d", updated.execSuggIdx)
	}
	if updated.suggExecState[0] != git.ExecFailed {
		t.Fatalf("expected first suggestion to be failed, got %v", updated.suggExecState[0])
	}
	if updated.suggExecState[1] != git.ExecRunning {
		t.Fatalf("expected second suggestion to be running, got %v", updated.suggExecState[1])
	}
}

func TestBatchRunStopsAtBlockedSuggestion(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.batchRunRequested = true
	m.suggestions = []git.Suggestion{
		{
			Action:      "Inspect Pages with unresolved placeholder",
			RiskLevel:   git.RiskSafe,
			Interaction: git.PlatformExec,
			PlatformOp: &git.PlatformExecInfo{
				CapabilityID: "pages",
				Flow:         "inspect",
				ResourceID:   "<site_id>",
			},
		},
		{
			Action:      "Inspect status",
			Command:     []string{"git", "status"},
			RiskLevel:   git.RiskSafe,
			Interaction: git.AutoExec,
		},
	}
	m.suggExecState = []git.ExecState{git.ExecPending, git.ExecPending}
	m.suggExecMsg = []string{"", ""}
	m.automation = config.AutomationConfig{MaxAutoSteps: 8}
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:owner/repo.git",
		}},
	}

	_, _, ok := m.autoExecuteNextSafeSuggestion(true)
	if ok {
		t.Fatal("expected batch run to stop at blocked suggestion, not skip to next")
	}
	if m.suggExecState[0] != git.ExecPending {
		t.Fatalf("expected blocked suggestion to remain pending, got %v", m.suggExecState[0])
	}
	if m.suggExecState[1] != git.ExecPending {
		t.Fatalf("expected second suggestion to remain pending due to sequential execution, got %v", m.suggExecState[1])
	}
}

func TestBatchRunForceExecutesExplicitGitCommands(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.batchRunRequested = true
	m.suggestions = []git.Suggestion{
		{
			Action:      "Commit staged changes",
			Command:     []string{"git", "commit", "-m", "checkpoint"},
			RiskLevel:   git.RiskSafe,
			Interaction: git.AutoExec,
		},
		{
			Action:      "Create and switch branch",
			Command:     []string{"git", "switch", "-c", "new-branch"},
			RiskLevel:   git.RiskSafe,
			Interaction: git.AutoExec,
		},
	}
	m.suggExecState = []git.ExecState{git.ExecPending, git.ExecPending}
	m.suggExecMsg = []string{"", ""}
	m.automation = config.AutomationConfig{MaxAutoSteps: 8}

	updated, cmd, ok := m.autoExecuteNextSafeSuggestion(true)
	if !ok || cmd == nil {
		t.Fatal("expected explicit batch run to execute the first command")
	}
	if updated.execSuggIdx != 0 {
		t.Fatalf("expected first suggestion to start, got exec index %d", updated.execSuggIdx)
	}
	if updated.suggExecState[0] != git.ExecRunning {
		t.Fatalf("expected first suggestion to be running, got %v", updated.suggExecState[0])
	}
}

func TestBatchRunContinuesIntoFileWriteSuggestion(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.batchRunRequested = true
	m.suggestions = []git.Suggestion{
		{
			Action:      "Create branch",
			Command:     []string{"git", "switch", "-c", "new-branch"},
			RiskLevel:   git.RiskSafe,
			Interaction: git.AutoExec,
		},
		{
			Action:      "Create file",
			RiskLevel:   git.RiskSafe,
			Interaction: git.FileWrite,
			FileOp: &git.FileWriteInfo{
				Path:      "newfile.txt",
				Operation: "create",
				Content:   "hello",
			},
		},
	}
	m.suggExecState = []git.ExecState{git.ExecRunning, git.ExecPending}
	m.suggExecMsg = []string{"running...", ""}
	m.execSuggIdx = 0
	m.automation = config.AutomationConfig{MaxAutoSteps: 8}

	next, cmd := m.Update(commandResultMsg{result: &git.ExecutionResult{
		Command: []string{"git", "switch", "-c", "new-branch"},
		Success: true,
	}})
	updated := next.(Model)

	if cmd == nil {
		t.Fatal("expected batch run to continue into file-write suggestion")
	}
	if updated.execSuggIdx != 1 {
		t.Fatalf("expected file-write suggestion to start next, got exec index %d", updated.execSuggIdx)
	}
	if updated.suggExecState[1] != git.ExecRunning {
		t.Fatalf("expected file-write suggestion to be running, got %v", updated.suggExecState[1])
	}
}

func TestRunAllReportsWhyNothingCanRun(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.suggestions = []git.Suggestion{
		{
			Action:      "Need manual input",
			Interaction: git.NeedsInput,
			Inputs: []git.InputField{{
				Key:   "branch",
				Label: "Branch",
			}},
		},
	}
	m.suggExecState = []git.ExecState{git.ExecPending}
	m.suggExecMsg = []string{""}

	model, cmd := m.runExecutionSlashCommand([]string{"all"})
	updated := model.(Model)

	if cmd != nil {
		t.Fatal("expected no command when nothing is runnable")
	}
	if !strings.Contains(updated.commandResponseBody, "manual input required") && !strings.Contains(updated.commandResponseBody, "需要手动输入") {
		t.Fatalf("expected command response to explain why batch run cannot start, got %q", updated.commandResponseBody)
	}
}

func TestSubmitInlineRunAllStartsBatchImmediately(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.screen = screenMain
	m.composerInput = "/run all"
	m.composerCursor = len(m.composerInput)
	m.suggestions = []git.Suggestion{
		{
			Action:      "Create branch",
			Command:     []string{"git", "switch", "-c", "new-branch"},
			RiskLevel:   git.RiskSafe,
			Interaction: git.AutoExec,
		},
	}
	m.suggExecState = []git.ExecState{git.ExecPending}
	m.suggExecMsg = []string{""}
	m.automation = config.AutomationConfig{MaxAutoSteps: 8}

	model, cmd := m.submitInlineGoal()
	updated := model.(Model)

	if cmd == nil {
		t.Fatal("expected /run all to start batch execution immediately")
	}
	if !updated.batchRunRequested {
		t.Fatal("expected batchRunRequested to stay enabled after /run all")
	}
	if updated.execSuggIdx != 0 {
		t.Fatalf("expected first suggestion to start, got exec index %d", updated.execSuggIdx)
	}
}

func TestSyncTaskMemoryMarksGoalInProgress(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.session.ActiveGoal = "Ship the current branch"
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:owner/repo.git",
		}},
	}

	m.syncTaskMemory()

	snapshot := m.memoryStore.Snapshot()
	repo := snapshot.Repos[m.repoFingerprint()]
	if repo == nil || repo.Task == nil {
		t.Fatal("expected task memory to be persisted")
	}
	if repo.Task.Status != "in_progress" {
		t.Fatalf("expected goal status to default to in_progress, got %q", repo.Task.Status)
	}
}
