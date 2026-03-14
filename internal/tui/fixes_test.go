package tui

import (
	"path/filepath"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	gitplatform "github.com/Joker-of-Gotham/gitdex/internal/platform"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFixesTest(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	require.NoError(t, i18n.Init("en"))
	t.Cleanup(func() { _ = i18n.Init("en") })
	config.Set(config.DefaultConfig())
}

func TestAutomationModeThreeModesOnly(t *testing.T) {
	assert.Equal(t, "manual", config.NormalizeAutomationMode("manual"))
	assert.Equal(t, "manual", config.NormalizeAutomationMode("assist"))
	assert.Equal(t, "auto", config.NormalizeAutomationMode("auto"))
	assert.Equal(t, "cruise", config.NormalizeAutomationMode("cruise"))
	assert.Equal(t, "manual", config.NormalizeAutomationMode("unknown"))
	assert.Equal(t, "manual", config.NormalizeAutomationMode(""))
}

func TestAutomationModeIsAutoLoop(t *testing.T) {
	assert.False(t, config.AutomationModeIsAutoLoop("manual"))
	assert.False(t, config.AutomationModeIsAutoLoop("assist"))
	assert.True(t, config.AutomationModeIsAutoLoop("auto"))
	assert.True(t, config.AutomationModeIsAutoLoop("cruise"))
}

func TestManualModeFlags(t *testing.T) {
	cfg := config.AutomationConfig{Mode: "manual"}
	config.ApplyAutomationMode(&cfg)
	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.AutoAnalyze)
	assert.False(t, cfg.Unattended)
	assert.False(t, cfg.AutoAcceptSafe)
}

func TestAutoModeFlags(t *testing.T) {
	cfg := config.AutomationConfig{Mode: "auto"}
	config.ApplyAutomationMode(&cfg)
	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.AutoAnalyze)
	assert.True(t, cfg.Unattended)
	assert.True(t, cfg.AutoAcceptSafe)
}

func TestCruiseModeFlags(t *testing.T) {
	cfg := config.AutomationConfig{Mode: "cruise"}
	config.ApplyAutomationMode(&cfg)
	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.AutoAnalyze)
	assert.True(t, cfg.Unattended)
	assert.True(t, cfg.AutoAcceptSafe)
	assert.True(t, config.AutomationModeAllowsSelfDirectedGoals("cruise"))
}

func TestCommandResultDoesNotSwitchTabDuringBatchRun(t *testing.T) {
	setupFixesTest(t)

	m := NewModel()
	m.screen = screenMain
	m.width = 120
	m.height = 40
	m.ready = true
	m.opLog = oplog.New(oplog.DefaultMaxEntries)
	m.batchRunRequested = true
	m.workspaceTab = workspaceTabSuggestions
	m.suggestions = []git.Suggestion{
		{Action: "git status", Command: []string{"git", "status"}, Interaction: git.AutoExec, RiskLevel: git.RiskSafe},
		{Action: "git diff", Command: []string{"git", "diff"}, Interaction: git.AutoExec, RiskLevel: git.RiskSafe},
	}
	m.suggExecState = []git.ExecState{git.ExecRunning, git.ExecPending}
	m.suggExecMsg = []string{"running...", ""}
	m.execSuggIdx = 0

	model, _ := m.Update(commandResultMsg{
		result: &git.ExecutionResult{
			Command: []string{"git", "status"},
			Success: true,
			Stdout:  "On branch main",
		},
	})
	updated := model.(Model)

	assert.Equal(t, workspaceTabSuggestions, updated.workspaceTab,
		"workspace tab should stay on suggestions during batch run, not switch to result")
}

func TestCommandResultSwitchesTabOnManualExec(t *testing.T) {
	setupFixesTest(t)

	m := NewModel()
	m.screen = screenMain
	m.width = 120
	m.height = 40
	m.ready = true
	m.opLog = oplog.New(oplog.DefaultMaxEntries)
	m.batchRunRequested = false
	m.automation = config.AutomationConfig{Mode: "manual"}
	config.ApplyAutomationMode(&m.automation)
	m.workspaceTab = workspaceTabSuggestions
	m.suggestions = []git.Suggestion{
		{Action: "git status", Command: []string{"git", "status"}, Interaction: git.AutoExec},
	}
	m.suggExecState = []git.ExecState{git.ExecRunning}
	m.suggExecMsg = []string{"running..."}
	m.execSuggIdx = 0

	model, _ := m.Update(commandResultMsg{
		result: &git.ExecutionResult{
			Command: []string{"git", "status"},
			Success: true,
			Stdout:  "On branch main",
		},
	})
	updated := model.(Model)

	assert.Equal(t, workspaceTabResult, updated.workspaceTab,
		"workspace tab should switch to result on manual single execution")
}

func TestFailedCommandTriggersReAnalysisInAutoMode(t *testing.T) {
	setupFixesTest(t)

	m := NewModel()
	m.screen = screenMain
	m.width = 120
	m.height = 40
	m.ready = true
	m.opLog = oplog.New(oplog.DefaultMaxEntries)
	m.automation = config.AutomationConfig{
		Mode:    "auto",
		Enabled: true,
	}
	config.ApplyAutomationMode(&m.automation)
	m.session.ActiveGoal = "fix tests"
	m.batchRunRequested = false
	m.suggestions = []git.Suggestion{
		{Action: "git push", Command: []string{"git", "push"}, Interaction: git.AutoExec},
		{Action: "git status", Command: []string{"git", "status"}, Interaction: git.AutoExec},
	}
	m.suggExecState = []git.ExecState{git.ExecRunning, git.ExecPending}
	m.suggExecMsg = []string{"running...", ""}
	m.execSuggIdx = 0

	model, cmd := m.Update(commandResultMsg{
		result: &git.ExecutionResult{
			Command:  []string{"git", "push"},
			Success:  false,
			ExitCode: 1,
			Stderr:   "fatal: remote rejected",
		},
	})
	updated := model.(Model)

	assert.NotNil(t, cmd, "should trigger re-analysis after failure in auto mode")
	assert.Empty(t, updated.lastAnalysisFingerprint,
		"analysis fingerprint should be cleared to force re-analysis")
	assert.False(t, updated.batchRunRequested,
		"batch run should be cancelled on failure in auto mode")
}

func TestBatchRunCanBeReTriggeredAfterNewSuggestions(t *testing.T) {
	setupFixesTest(t)

	m := NewModel()
	m.screen = screenMain
	m.width = 120
	m.height = 40
	m.ready = true
	m.opLog = oplog.New(oplog.DefaultMaxEntries)
	m.automation = config.AutomationConfig{Mode: "manual"}
	config.ApplyAutomationMode(&m.automation)
	m.suggestions = []git.Suggestion{
		{Action: "git status", Command: []string{"git", "status"}, Interaction: git.AutoExec, RiskLevel: git.RiskSafe},
	}
	m.suggExecState = []git.ExecState{git.ExecPending}
	m.suggExecMsg = []string{""}

	assert.True(t, m.shouldAllowBatchRun(), "batch run should be allowed with pending suggestions")

	m.suggExecState[0] = git.ExecDone
	assert.False(t, m.shouldAllowBatchRun(), "batch run should not be allowed when all done")

	m.suggestions = append(m.suggestions, git.Suggestion{
		Action: "git diff", Command: []string{"git", "diff"}, Interaction: git.AutoExec, RiskLevel: git.RiskSafe,
	})
	m.suggExecState = append(m.suggExecState, git.ExecPending)
	m.suggExecMsg = append(m.suggExecMsg, "")
	assert.True(t, m.shouldAllowBatchRun(), "batch run should be allowed again after new suggestions")
}

func TestScrollClampOffsetAllowsBottomReach(t *testing.T) {
	assert.Equal(t, 0, clampOffset(10, 20, 0), "fewer lines than height: stay at 0")
	assert.Equal(t, 0, clampOffset(10, 10, 0), "exact fit: stay at 0")
	assert.Equal(t, 10, clampOffset(20, 10, 15), "should clamp to max")
	assert.Equal(t, 10, clampOffset(20, 10, 10), "should allow scroll to totalLines-height")
	assert.Equal(t, 5, clampOffset(20, 10, 5), "should allow intermediate scroll")
	assert.Equal(t, 0, clampOffset(20, 10, -1), "should not go below 0")
}

func TestLayoutMetricsConsistency(t *testing.T) {
	m := Model{width: 120, height: 40, ready: true}
	metrics := m.computeLayoutMetrics()

	assert.Greater(t, metrics.workspaceHeight, 0, "workspace height should be positive")
	assert.Greater(t, metrics.logHeight, 0, "log height should be positive")
	assert.Equal(t, metrics.workspaceHeight+metrics.logGap+metrics.logHeight, m.contentHeight(),
		"workspace + gap + log should equal content height (wide layout)")

	m.logExpanded = true
	metricsExpanded := m.computeLayoutMetrics()
	assert.Greater(t, metricsExpanded.logHeight, metrics.logHeight,
		"expanded log should be taller")
	assert.Equal(t, metricsExpanded.workspaceHeight+metricsExpanded.logGap+metricsExpanded.logHeight, m.contentHeight(),
		"expanded layout should still sum to content height")
}

func TestLayoutMetricsNarrow(t *testing.T) {
	m := Model{width: 80, height: 30, ready: true}
	metrics := m.computeLayoutMetrics()
	assert.Greater(t, metrics.workspaceHeight, 0)
	assert.LessOrEqual(t, metrics.workspaceHeight+metrics.logGap+metrics.logHeight, m.contentHeight())
}

func TestGoalPersistedAcrossRestarts(t *testing.T) {
	setupFixesTest(t)

	m := NewModel()
	m.session.ActiveGoal = "deploy feature"
	m.persistAutomationCheckpoint()

	state := loadAutomationCheckpoint()
	assert.Equal(t, "deploy feature", state.ActiveGoal, "goal should be persisted")
}

func TestGoalPreservedOnSameRepo(t *testing.T) {
	setupFixesTest(t)

	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:     "origin",
			FetchURL: "https://github.com/example/repo.git",
			PushURL:  "https://github.com/example/repo.git",
		}},
	}
	m.session.ActiveGoal = "deploy feature"
	m.loadedCheckpointRepo = ""

	m.reconcileRepoScopedState()
	assert.Equal(t, "deploy feature", m.session.ActiveGoal,
		"goal should be preserved when loading for the first time")
}

func TestGoalClearedOnDifferentRepo(t *testing.T) {
	setupFixesTest(t)

	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:     "origin",
			FetchURL: "https://github.com/example/new-repo.git",
			PushURL:  "https://github.com/example/new-repo.git",
		}},
	}
	m.session.ActiveGoal = "deploy feature"
	m.loadedCheckpointRepo = "old-repo-fingerprint"

	m.reconcileRepoScopedState()
	assert.Empty(t, m.session.ActiveGoal,
		"goal from a different repo should be cleared")
	assert.Nil(t, m.workflowFlow, "workflow should be cleared on repo change")
}

func TestCheckpointTTLIs72Hours(t *testing.T) {
	assert.Equal(t, 72*60*60, int(checkpointResumeTTL.Seconds()),
		"checkpoint TTL should be 72 hours")
}

func TestHandlePostExecution_AllDoneTriggersReAnalysis(t *testing.T) {
	setupFixesTest(t)

	m := NewModel()
	m.screen = screenMain
	m.width = 120
	m.height = 40
	m.ready = true
	m.opLog = oplog.New(oplog.DefaultMaxEntries)
	m.suggestions = []git.Suggestion{
		{Action: "git status", Command: []string{"git", "status"}, Interaction: git.AutoExec},
	}
	m.suggExecState = []git.ExecState{git.ExecDone}
	m.suggExecMsg = []string{"success"}
	m.lastAnalysisFingerprint = "old"

	result, cmd := m.handlePostExecution(false)
	updated := result.(Model)

	assert.NotNil(t, cmd, "should trigger re-analysis when all suggestions are done")
	assert.Empty(t, updated.lastAnalysisFingerprint,
		"fingerprint should be cleared for re-analysis")
}

func TestAutoModeLabelsCorrect(t *testing.T) {
	setupFixesTest(t)

	assert.Equal(t, "manual", localizedAutomationModeLabel("manual"))
	assert.Equal(t, "auto", localizedAutomationModeLabel("auto"))
	assert.Equal(t, "cruise", localizedAutomationModeLabel("cruise"))
}

func TestAutoModeDescriptionsNoAssist(t *testing.T) {
	setupFixesTest(t)

	desc := localizedAutomationModeDescription("manual")
	assert.Contains(t, desc, "/run accept")
	assert.Contains(t, desc, "/run all")

	desc = localizedAutomationModeDescription("auto")
	assert.Contains(t, desc, "fully automatic")

	desc = localizedAutomationModeDescription("cruise")
	assert.Contains(t, desc, "self-checks")
}

func TestSlashModeRejectsAssist(t *testing.T) {
	setupFixesTest(t)

	m := NewModel()
	model, _ := m.runModeSlashCommand([]string{"assist"})
	updated := model.(Model)
	assert.Contains(t, updated.statusMsg, "Unknown mode")
}

func TestNewModelRestoresGoalStatusAfterReconcile(t *testing.T) {
	setupFixesTest(t)

	m := NewModel()
	fp := "https://github.com/example/repo.git"
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{Name: "origin", FetchURL: fp, PushURL: fp}},
	}
	m.loadedCheckpointRepo = m.repoFingerprint()
	m.session.ActiveGoal = "fix CI"
	m.persistAutomationCheckpoint()

	m2 := NewModel()
	assert.Empty(t, m2.session.ActiveGoal, "goal should be deferred until reconcile")
	m2.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{Name: "origin", FetchURL: fp, PushURL: fp}},
	}
	m2.reconcileRepoScopedState()
	assert.Equal(t, "fix CI", m2.session.ActiveGoal, "goal should be restored after reconcile")
	assert.Equal(t, "in_progress", m2.llmGoalStatus, "goal status should be restored as in_progress")
}

func TestSuggestionFilterTrustsLLMWithGoal(t *testing.T) {
	setupFixesTest(t)
	m := NewModel()
	m.session.ActiveGoal = "Create PR"
	m.gitState = &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}}

	s := git.Suggestion{
		Action:      "Enable Pages",
		Interaction: git.PlatformExec,
		PlatformOp:  &git.PlatformExecInfo{CapabilityID: "pages", Flow: "inspect"},
	}
	assert.True(t, m.suggestionRelevantToRepo(s, m.gitState, m.session.ActiveGoal))
}

func TestSuggestionFilterTrustsLLMWithoutGoal(t *testing.T) {
	setupFixesTest(t)
	m := NewModel()
	m.session.ActiveGoal = ""
	m.gitState = &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}}

	s := git.Suggestion{
		Action:      "Enable Pages",
		Interaction: git.PlatformExec,
		PlatformOp:  &git.PlatformExecInfo{CapabilityID: "pages", Flow: "inspect"},
	}
	assert.True(t, m.suggestionRelevantToRepo(s, m.gitState, ""))
}

func TestDetectedPlatformCaching(t *testing.T) {
	setupFixesTest(t)
	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "https://github.com/user/repo.git",
		}},
	}
	m.refreshCachedPlatform()
	assert.NotEqual(t, gitplatform.PlatformUnknown, m.cachedPlatformID, "cache should be populated after refresh")
	assert.Equal(t, m.cachedPlatformID, m.detectedPlatform(), "detectedPlatform should use cache")
}

func TestTaskMemorySyncOnWorkflowSelect(t *testing.T) {
	setupFixesTest(t)
	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{Name: "origin", PushURL: "https://github.com/user/repo.git"}},
	}
	m.session.ActiveGoal = "Create new branch"
	m.syncTaskMemory()
	mem := m.currentPromptMemory()
	if mem != nil && mem.TaskState != nil {
		assert.Equal(t, "Create new branch", mem.TaskState.Goal, "store should have user goal")
	}

	m.session.ActiveGoal = "Deploy to pages"
	m.syncTaskMemory()
	mem = m.currentPromptMemory()
	if mem != nil && mem.TaskState != nil {
		assert.Equal(t, "Deploy to pages", mem.TaskState.Goal, "store should sync after goal change")
	}
}

func TestRepoScopedStateClearsGoalOnRepoSwitch(t *testing.T) {
	setupFixesTest(t)
	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{Name: "origin", PushURL: "https://github.com/user/repo1.git"}},
	}
	m.session.ActiveGoal = "Fix bug"
	m.loadedCheckpointRepo = "old-repo-fingerprint"
	m.reconcileRepoScopedState()
	assert.Empty(t, m.session.ActiveGoal, "goal from different repo should be cleared")
}

func TestAutoRetryOnLLMError(t *testing.T) {
	setupFixesTest(t)
	m := NewModel()
	m.automation = config.AutomationConfig{
		Enabled:       true,
		AutoAnalyze:   true,
		Unattended:    true,
		AutoAcceptSafe: true,
		Mode:          "auto",
	}
	config.ApplyAutomationMode(&m.automation)
	m.session.ActiveGoal = "test goal"
	m.gitState = &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}}
	m.pendingAnalysisID = 1

	result, cmd := m.Update(llmResultMsg{
		requestID: 1,
		err:       assert.AnError,
	})
	updated := result.(Model)
	assert.Equal(t, 1, updated.consecutiveAnalysisFailures, "failure count should increment")
	assert.NotNil(t, cmd, "should schedule a retry command")
}

func TestAutoRetryResetsOnSuccess(t *testing.T) {
	setupFixesTest(t)
	m := NewModel()
	m.consecutiveAnalysisFailures = 2
	m.automation = config.AutomationConfig{
		Enabled:       true,
		AutoAnalyze:   true,
		Unattended:    true,
		AutoAcceptSafe: true,
		Mode:          "auto",
	}
	config.ApplyAutomationMode(&m.automation)
	m.session.ActiveGoal = "test goal"
	m.gitState = &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}}
	m.pendingAnalysisID = 1

	result, _ := m.Update(llmResultMsg{
		requestID: 1,
		analysis:  "All looks good.",
		suggestions: []git.Suggestion{{
			ID:          "test-1",
			Action:      "git status",
			Command:     []string{"git", "status"},
			RiskLevel:   git.RiskSafe,
			Interaction: git.AutoExec,
		}},
	})
	updated := result.(Model)
	assert.Equal(t, 0, updated.consecutiveAnalysisFailures, "failures should reset on successful analysis with suggestions")
}

func TestHandlePostExecutionReAnalyzesInAutoMode(t *testing.T) {
	setupFixesTest(t)
	m := NewModel()
	m.automation = config.AutomationConfig{
		Enabled:       true,
		AutoAnalyze:   true,
		Unattended:    true,
		AutoAcceptSafe: true,
		Mode:          "auto",
	}
	config.ApplyAutomationMode(&m.automation)
	m.session.ActiveGoal = "test goal"
	m.gitState = &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}}
	m.suggestions = []git.Suggestion{{
		ID:          "s1",
		Action:      "git status",
		Command:     []string{"git", "status"},
		RiskLevel:   git.RiskSafe,
		Interaction: git.AutoExec,
	}}
	m.suggExecState = []git.ExecState{git.ExecDone}
	m.suggExecMsg = []string{"success"}

	_, cmd := m.handlePostExecution(false)
	assert.NotNil(t, cmd, "should trigger refresh for re-analysis when all done in auto mode")
	assert.Equal(t, "", m.lastAnalysisFingerprint, "fingerprint should be cleared to force re-analysis")
}
