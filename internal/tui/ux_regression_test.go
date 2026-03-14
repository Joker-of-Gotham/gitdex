package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	platformruntime "github.com/Joker-of-Gotham/gitdex/internal/platform/runtime"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func prepareTUIEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	require.NoError(t, i18n.Init("en"))
	t.Cleanup(func() {
		_ = i18n.Init("en")
	})
	config.Set(config.DefaultConfig())
}

func TestUpdateMainPromptFocusAllowsShortcutCharacters(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.screen = screenMain

	model, _ := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "/"}))
	updated := model.(Model)
	require.True(t, updated.composerFocused)

	model, cmd := updated.updateMain(tea.KeyPressMsg(tea.Key{Text: "q"}))
	updated = model.(Model)
	assert.Nil(t, cmd)
	assert.True(t, updated.composerFocused)
	assert.Equal(t, "q", updated.composerInput)
}

func TestSlashSettingsIntervalUpdatesConfig(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.screen = screenMain
	m.composerInput = "/settings interval 900"
	m.composerCursor = len(m.composerInput)

	model, _ := m.submitInlineGoal()
	updated := model.(Model)

	assert.Equal(t, 900, updated.automation.MonitorInterval)
	assert.Contains(t, updated.statusMsg, "interval=900s")
}

func TestSlashModeCruiseUpdatesAutomationFlags(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	model, cmd := m.runSlashCommand("/mode cruise")
	updated := model.(Model)

	assert.NotNil(t, model)
	_ = cmd
	assert.Equal(t, config.AutomationModeCruise, updated.automation.Mode)
	assert.True(t, updated.automation.Enabled)
	assert.True(t, updated.automation.AutoAnalyze)
	assert.True(t, updated.automation.Unattended)
	assert.True(t, updated.automation.AutoAcceptSafe)
}

func TestSlashEnterAutocompletesPartialCommand(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.screen = screenMain
	m.composerFocused = true
	m.composerInput = "/he"
	m.composerCursor = len(m.composerInput)

	model, cmd := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	updated := model.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, "/help", updated.composerInput)
	assert.True(t, updated.composerFocused)
}

func TestSlashHelpWritesCommandResponsePanel(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	model, cmd := m.runSlashCommand("/help")
	updated := model.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, "Slash commands", updated.commandResponseTitle)
	assert.Contains(t, updated.commandResponseBody, "/help")
	assert.Contains(t, updated.commandResponseBody, "/settings")
}

func TestCruiseModeSynthesizesGoalWithoutHumanInput(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.gitState = &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}}
	m.automation = config.AutomationConfig{
		Mode:           config.AutomationModeCruise,
		Enabled:        true,
		AutoAnalyze:    true,
		Unattended:     true,
		AutoAcceptSafe: true,
	}

	updated, cmd, ok := m.applyCruiseGoalIfNeeded()

	assert.True(t, ok)
	assert.Nil(t, cmd)
	assert.NotEmpty(t, updated.session.ActiveGoal)
	assert.Contains(t, strings.ToLower(updated.session.ActiveGoal), "audit")
}

func TestSlashLLMCommandOpensModelSetup(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.screen = screenMain
	m.composerInput = "/llm"
	m.composerCursor = len(m.composerInput)

	model, cmd := m.submitInlineGoal()
	updated := model.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, screenModelSelect, updated.screen)
}

func TestRunConfigSlashCommandShowsStatus(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.screen = screenMain
	m.composerInput = "/config status"
	m.composerCursor = len(m.composerInput)

	model, cmd := m.submitInlineGoal()
	updated := model.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, "Configuration", updated.commandResponseTitle)
	assert.Contains(t, updated.commandResponseBody, "Platform access")
	assert.Contains(t, updated.commandResponseBody, "/settings")
}

func TestReconcileRepoScopedStateClearsForeignGoalAndWorkflow(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURL:      "https://github.com/example/new-repo.git",
			PushURL:       "https://github.com/example/new-repo.git",
			FetchURLValid: true,
			PushURLValid:  true,
		}},
	}
	m.loadedCheckpointRepo = "foreign-repo"
	m.session.ActiveGoal = "Plan Pages deployment"
	m.workflowFlow = &workflowFlowState{WorkflowID: "pages_setup"}
	m.mutationLedger = []platform.MutationLedgerEntry{{ID: "ledger-1"}}
	m.suggestions = []git.Suggestion{{Action: "Deploy Pages"}}
	m.lastAnalysisFingerprint = "old"

	m.reconcileRepoScopedState()

	assert.Empty(t, m.session.ActiveGoal, "goal from foreign repo should be cleared")
	assert.Nil(t, m.workflowFlow)
	assert.Empty(t, m.suggestions)
	assert.Empty(t, m.lastAnalysisFingerprint)
	assert.Contains(t, m.commandResponseTitle, "Repository context reset")
}

func TestRenderInlineComposerShowsSlashSuggestions(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.screen = screenMain
	m.composerFocused = true
	m.composerInput = "/co"
	m.composerCursor = len(m.composerInput)

	out := m.renderInlineComposer(100)

	assert.Contains(t, out, "Commands")
	assert.Contains(t, out, "/config status")
	assert.Contains(t, out, "/config platform")
}

func TestUIClickObservabilityTabSelectsTab(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()

	model, cmd := m.handleUIClick(uiClickMsg{action: "observability_tab", index: 2})
	updated := model.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, observabilityContext, updated.obsTab)
}

func TestUIClickSelectSuggestionShowsGuidance(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.suggIdx = -1
	m.suggestions = []git.Suggestion{{
		Action:      "Inspect branch rulesets",
		Reason:      "Inspect current repository rules before proposing mutations.",
		Interaction: git.PlatformExec,
		PlatformOp:  &git.PlatformExecInfo{CapabilityID: "branch_rulesets", Flow: "inspect"},
	}}
	m.suggExecState = []git.ExecState{git.ExecPending}

	model, cmd := m.handleUIClick(uiClickMsg{action: "select_suggestion", index: 0})
	updated := model.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, 0, updated.suggIdx)
	assert.Contains(t, updated.commandResponseBody, "/accept")
	assert.Contains(t, updated.commandResponseBody, "/refresh")
	assert.Contains(t, updated.commandResponseBody, "/quit")
}

func TestViewSuggestionsKeepsPrimaryWorkspaceFocused(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.screen = screenMain
	m.width = 120
	m.height = 40
	m.suggestions = []git.Suggestion{{
		Action:      "Inspect branch rulesets",
		Reason:      "Review repository policy posture.",
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "branch_rulesets",
			Flow:         "inspect",
		},
	}}
	m.suggExecState = make([]git.ExecState, len(m.suggestions))
	m.suggExecMsg = make([]string, len(m.suggestions))
	m.llmAnalysis = "Analysis should live behind the analysis tab."
	m.lastCommand = commandTrace{
		Title:  "pages / inspect",
		Status: "platform unavailable",
		Output: "Platform action unavailable",
	}

	model, cmd := m.runSlashCommand("/view suggestions")
	updated := model.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, workspaceTabSuggestions, updated.workspaceTab)
	out, _ := updated.renderLeftWorkspaceWithRegions(80)
	assert.Contains(t, out, "Inspect branch rulesets")
	assert.NotContains(t, out, localizedLatestResultTitle())
}

func TestProviderConfigHidesStoredAPIKeyAndPreservesUntouchedValue(t *testing.T) {
	prepareTUIEnv(t)

	cfg := config.DefaultConfig()
	cfg.LLM.Provider = "deepseek"
	cfg.LLM.Model = "deepseek-chat"
	cfg.LLM.Endpoint = "https://api.deepseek.com"
	cfg.LLM.Primary.Provider = "deepseek"
	cfg.LLM.Primary.Model = "deepseek-chat"
	cfg.LLM.Primary.Endpoint = "https://api.deepseek.com"
	cfg.LLM.Primary.APIKey = "sk-stored-key"
	cfg.LLM.Primary.APIKeyEnv = ""
	config.Set(cfg)

	m := NewModel()
	m = m.SetLLMConfig(cfg.LLM)
	m = m.openProviderConfig(selectPrimary)

	assert.Empty(t, m.providerDraft.APIKey)
	assert.Equal(t, "sk-stored-key", m.providerStoredKey)

	model, cmd := m.persistProviderConfig()
	updated := model.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, "sk-stored-key", config.Get().LLM.Primary.APIKey)
	assert.Equal(t, "sk-stored-key", updated.llmConfig.Primary.APIKey)
}

func TestRenderOperationLogPanelCollapsedOnlyShowsHeader(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.logExpanded = false
	m.opLog.Add(oplogEntry("entry hidden while collapsed"))

	out := m.renderOperationLogPanelCached(60, 1)

	assert.Equal(t, "> Operation Log (collapsed)", strings.TrimSpace(stripANSI(out)))
	assert.NotContains(t, out, "entry hidden while collapsed")
	assert.NotContains(t, out, "pgup/pgdn")
	assert.Equal(t, 1, lipgloss.Height(out))
}

func TestRenderMainLayoutKeepsCollapsedLogAtBottom(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.width = 120
	m.height = 40
	m.gitState = &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}}
	m.logExpanded = false
	m.opLog.Add(oplogEntry("entry hidden while collapsed"))

	out, _, _ := m.renderMainLayoutWithRegions(20)
	lines := strings.Split(out, "\n")
	lastNonEmpty := ""
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			lastNonEmpty = line
		}
	}

	assert.Contains(t, lastNonEmpty, "Operation Log")
}

func TestRenderAnalysisPanelOmitsContextStats(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.width = 100
	m.gitState = &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}}
	m.session.ActiveGoal = "Inspect pages"
	m.llmAnalysis = "Pages configuration needs validation."
	m.llmDebugInfo = "sys:128 usr:256 budget:4096"

	out := m.renderAnalysisPanelCached(90)

	assert.Contains(t, out, "Goal: Inspect pages")
	assert.NotContains(t, out, "[ctx]")
	assert.NotContains(t, out, "branch:main")
}

func TestRenderSuggestionCardsPlatformShowsNotesWithoutFakeShellPrompt(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.suggestions = []git.Suggestion{{
		Action:      "Check latest Pages build",
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "pages",
			Flow:         "inspect",
			Query:        map[string]string{"view": "latest_build"},
		},
	}}
	m.suggExecState = []git.ExecState{git.ExecPending}
	m.suggExecMsg = []string{"Platform access needs configuration. Run /config status."}

	out, _ := m.renderSuggestionCardsCompactWithRegions(120)

	assert.Contains(t, out, "Inspect GitHub Pages latest build")
	assert.Contains(t, out, "/config status")
	assert.NotContains(t, out, "$ Platform access needs configuration")
	assert.NotContains(t, out, "coverage:")
	assert.NotContains(t, out, "request:")
}

func TestRenderStatusBarFallsBackToLatestResult(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.width = 120
	m.lastCommand = commandTrace{
		Title:              "pages / inspect",
		Status:             "platform inspect success",
		Output:             "latest build healthy",
		ResultKind:         resultKindPlatformAdmin,
		PlatformCapability: "pages",
		PlatformFlow:       "inspect",
	}

	out := m.renderStatusBar()

	assert.Contains(t, out, "Platform inspect succeeded")
	assert.Contains(t, out, "GitHub Pages")
	assert.Contains(t, out, "latest build healthy")
}

func TestRenderLatestResultPanelLocalizesLabelsInChinese(t *testing.T) {
	prepareTUIEnv(t)
	require.NoError(t, i18n.Init("zh"))
	defer func() {
		_ = i18n.Init("en")
	}()

	m := NewModel()
	m.lastCommand = commandTrace{
		Title:              "pages / inspect",
		Status:             "platform unavailable",
		Output:             "adapter unavailable",
		ResultKind:         resultKindPlatformAdmin,
		PlatformCapability: "pages",
		PlatformFlow:       "inspect",
	}

	out := m.renderLatestResultPanel(100)

	assert.Contains(t, out, "最近结果")
	assert.Contains(t, out, "状态：")
	assert.Contains(t, out, "目标：")
}

func TestPrepareSuggestionsForDisplayTrustsLLMJudgment(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.session.ActiveGoal = ""
	m.gitState = &status.GitState{
		LocalBranch:   git.BranchInfo{Name: "main"},
		LocalBranches: []string{"main"},
	}
	suggestions := []git.Suggestion{{
		Action:      "Inspect GitHub Pages latest build",
		Interaction: git.PlatformExec,
		PlatformOp:  &git.PlatformExecInfo{CapabilityID: "pages", Flow: "inspect"},
	}}

	prepared, _, dropped := m.prepareSuggestionsForDisplay(suggestions)

	assert.Len(t, prepared, 1)
	assert.Equal(t, 0, dropped)
}

func TestPrepareSuggestionsForDisplayAddsPlatformInputsForPlaceholders(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURL:      "https://github.com/example/repo.git",
			PushURL:       "https://github.com/example/repo.git",
			FetchURLValid: true,
			PushURLValid:  true,
		}},
	}

	suggestions := []git.Suggestion{{
		Action:      "Inspect branch rulesets",
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "branch_rulesets",
			Flow:         "inspect",
			Query:        map[string]string{"branch": "<branch>"},
		},
	}}

	prepared, notes, dropped := m.prepareSuggestionsForDisplay(suggestions)

	require.Len(t, prepared, 1)
	assert.Equal(t, 0, dropped)
	assert.Len(t, prepared[0].Inputs, 1)
	assert.Equal(t, "<branch>", prepared[0].Inputs[0].Key)
	assert.Contains(t, notes[0], "/accept")
}

func TestRenderMainLayoutNarrowDoesNotOverflowWidth(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.width = 72
	m.height = 28
	m.gitState = &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}}
	m.commandResponseTitle = "Assistant"
	m.commandResponseBody = strings.Repeat("Long wrapped response ", 12)
	m.suggestions = []git.Suggestion{{
		Action:      "Inspect branch rulesets",
		Interaction: git.PlatformExec,
		PlatformOp:  &git.PlatformExecInfo{CapabilityID: "branch_rulesets", Flow: "inspect"},
	}}
	m.suggExecState = []git.ExecState{git.ExecPending}

	out, _, _ := m.renderMainLayoutWithRegions(24)
	for _, line := range strings.Split(out, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), 72)
	}
}

func TestUpdateMainPlatformSuggestionPreflightDoesNotHardFail(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.screen = screenMain
	m.platformCfg = config.PlatformConfig{}
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return nil, fmt.Errorf("browser adapter disabled")
	}
	m.suggestions = []git.Suggestion{{
		Action:      "Check latest Pages build",
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "pages",
			Flow:         "inspect",
			Query:        map[string]string{"view": "latest_build"},
		},
	}}
	m.suggExecState = []git.ExecState{git.ExecPending}
	m.suggExecMsg = []string{""}

	model, cmd := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "y"}))
	updated := model.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, git.ExecPending, updated.suggExecState[0])
	assert.Contains(t, updated.suggExecMsg[0], "GitHub Pages unavailable")
	assert.Contains(t, updated.statusMsg, "GitHub Pages unavailable")
	assert.Contains(t, updated.commandResponseBody, "/config status")
}

func oplogEntry(summary string) oplog.Entry {
	return oplog.Entry{Summary: summary}
}

func stripANSI(text string) string {
	return ansiRE.ReplaceAllString(text, "")
}

func TestConfigStatusBodyUsesAutomationModeSummary(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.automation = config.AutomationConfig{
		Mode:            config.AutomationModeManual,
		Enabled:         true,
		MonitorInterval: 900,
		MaxAutoSteps:    6,
	}

	body := m.configStatusBody()

	assert.Contains(t, body, "Automation: Mode: manual")
	assert.Contains(t, body, "Manual:")
	assert.Contains(t, body, "/mode show")
	assert.NotContains(t, body, "analyze=")
	assert.NotContains(t, body, "unattended=")
}

func TestOpenAutomationConfigCommandResponseUsesModeDescriptions(t *testing.T) {
	prepareTUIEnv(t)

	m := NewModel()
	m.automation = config.AutomationConfig{
		Mode:            config.AutomationModeCruise,
		Enabled:         true,
		MonitorInterval: 1200,
		TrustedMode:     true,
		MaxAutoSteps:    8,
	}

	updated := m.openAutomationConfig()

	assert.Equal(t, screenAutomationConfig, updated.screen)
	assert.Contains(t, updated.commandResponseBody, "Mode: cruise")
	assert.Contains(t, updated.commandResponseBody, "Fully autonomous:")
	assert.Contains(t, updated.commandResponseBody, "self-checks")
}
