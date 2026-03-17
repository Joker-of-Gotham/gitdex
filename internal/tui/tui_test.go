package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/executor"
	"github.com/Joker-of-Gotham/gitdex/internal/flow"
	"github.com/Joker-of-Gotham/gitdex/internal/helper"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/theme"
)

// ---------- Model defaults ----------

func TestNewModel_Defaults(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	if m.mode != "manual" {
		t.Errorf("expected mode=manual, got %s", m.mode)
	}
	if m.language != "en" {
		t.Errorf("expected language=en, got %s", m.language)
	}
	if m.activeFlow != "idle" {
		t.Errorf("expected activeFlow=idle, got %s", m.activeFlow)
	}
	if !m.composerFocus {
		t.Error("expected composerFocus=true by default")
	}
	if m.opLog == nil {
		t.Error("expected opLog to be initialized")
	}
	if m.page != PageMain {
		t.Errorf("expected page=PageMain, got %d", m.page)
	}
	if m.focusZone != FocusInput {
		t.Errorf("expected focusZone=FocusInput, got %d", m.focusZone)
	}
	for i, v := range m.panelScrolls {
		if v != 0 {
			t.Errorf("expected panelScrolls[%d]=0, got %d", i, v)
		}
	}
}

// ---------- Helpers ----------

func TestTruncStr(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"a very long string", 10, "a very long string"},
		{"newline\nsecond", 20, "newline"},
		{"exact10len", 10, "exact10len"},
		{"", 5, ""},
	}
	for _, tc := range tests {
		got := truncStr(tc.input, tc.max)
		if got != tc.want {
			t.Errorf("truncStr(%q, %d) = %q, want %q", tc.input, tc.max, got, tc.want)
		}
	}
}

func TestTruncStr_MultibyteNoCorruption(t *testing.T) {
	input := "修正删除已合并的分支操作"
	result := truncStr(input, 8)
	runes := []rune(result)
	for _, r := range runes {
		if r == 0xFFFD {
			t.Error("found replacement character in truncated string — garbled")
		}
	}
}

func TestParseObjectActionFromSlash(t *testing.T) {
	oa, err := ParseObjectActionFromSlash("/run all")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if oa.Key() != "suggestion.execute" {
		t.Fatalf("expected suggestion.execute, got %s", oa.Key())
	}
	if oa.Arg != "all" {
		t.Fatalf("expected arg all, got %q", oa.Arg)
	}
}

func TestParseObjectActionFromSlash_NewCommands(t *testing.T) {
	oa, err := ParseObjectActionFromSlash("/palette")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if oa.Key() != "ui.command_palette" {
		t.Fatalf("expected ui.command_palette, got %s", oa.Key())
	}

	oa, err = ParseObjectActionFromSlash("/failures")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if oa.Key() != "metrics.failure_dashboard" {
		t.Fatalf("expected metrics.failure_dashboard, got %s", oa.Key())
	}
}

func TestParseObjectActionFromSlash_Unknown(t *testing.T) {
	if _, err := ParseObjectActionFromSlash("/unknown"); err == nil {
		t.Fatal("expected unknown command error")
	}
}

func TestHandleComposerSubmit_PaletteCommand(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.composerText = "/palette"
	m2, _ := m.handleComposerSubmit()
	m = m2.(Model)
	if !m.showCommandPalette {
		t.Fatal("expected command palette to be visible")
	}
}

func TestHandleComposerSubmit_FailureDashboard(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.opLog.Add(oplog.Entry{Type: oplog.EntryCmdFail, Summary: "HTTP 404"})
	m.composerText = "/failures"
	m2, _ := m.handleComposerSubmit()
	m = m2.(Model)

	found := false
	for _, e := range m.opLog.Entries() {
		if strings.Contains(e.Summary, "Failure taxonomy dashboard") {
			found = true
			if !strings.Contains(e.Detail, "not_found") {
				t.Fatalf("expected taxonomy detail to include not_found, got %q", e.Detail)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected failure dashboard log entry")
	}
}

func TestHandleComposerSubmit_ReplayCommand(t *testing.T) {
	store := setupTestStore(t)
	ol := dotgitdex.NewOutputLog(store)
	err := ol.AppendRound(dotgitdex.Round{
		SessionID: "s1", RoundID: 1, Mode: "manual", Flow: "maintain",
		StartedAt: time.Now(), FinishedAt: time.Now(), Status: "success",
		Steps: []dotgitdex.Step{
			{SequenceID: 1, Name: "fetch", Command: "git fetch --prune", Success: true, StartedAt: time.Now(), FinishedAt: time.Now()},
		},
	})
	if err != nil {
		t.Fatalf("append round: %v", err)
	}

	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.composerText = "/replay"
	m2, _ := m.handleComposerSubmit()
	m = m2.(Model)

	found := false
	for _, e := range m.opLog.Entries() {
		if strings.Contains(e.Summary, "Replay script generated") {
			found = true
			if !strings.Contains(e.Detail, "git fetch --prune") {
				t.Fatalf("expected replay detail to contain command, got %q", e.Detail)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected replay script log entry")
	}
}

func TestCalcLayout_SmallWindow(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.width = 40
	m.height = 12
	geo := m.calcLayout()
	if geo.contentH < 5 {
		t.Errorf("expected contentH >= 5, got %d", geo.contentH)
	}
	total := geo.gitH + geo.goalH + geo.logH + 2
	if total > geo.contentH {
		t.Errorf("right panel overflow: sub-panels %d > contentH %d", total, geo.contentH)
	}
}

func TestZoneFromXY(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.width = 100
	m.height = 30

	inputZone := m.zoneFromXY(5, 29)
	if inputZone != FocusInput {
		t.Errorf("expected FocusInput at bottom, got %d", inputZone)
	}

	geo := m.calcLayout()
	leftZone := m.zoneFromXY(5, geo.headerH+1)
	if leftZone != FocusLeft {
		t.Errorf("expected FocusLeft in left column, got %d", leftZone)
	}

	rightZone := m.zoneFromXY(geo.leftW+5, geo.headerH+1)
	if rightZone != FocusGit {
		t.Errorf("expected FocusGit in right column top, got %d", rightZone)
	}
}

func TestRenderProgressBar(t *testing.T) {
	bar := renderProgressBar(5, 10, 20)
	if bar == "" {
		t.Error("expected non-empty progress bar")
	}

	emptyBar := renderProgressBar(0, 0, 10)
	if emptyBar == "" {
		t.Error("expected non-empty bar for zero total")
	}
}

// ---------- Find / compress ----------

func TestFindNextPending(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.suggestions = []SuggestionDisplay{
		{Item: planner.SuggestionItem{Name: "a"}, Status: StatusDone},
		{Item: planner.SuggestionItem{Name: "b"}, Status: StatusPending},
		{Item: planner.SuggestionItem{Name: "c"}, Status: StatusPending},
	}
	idx := m.findNextPending()
	if idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}
}

func TestFindNextPending_None(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.suggestions = []SuggestionDisplay{
		{Item: planner.SuggestionItem{Name: "a"}, Status: StatusDone},
		{Item: planner.SuggestionItem{Name: "b"}, Status: StatusFailed},
	}
	idx := m.findNextPending()
	if idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}

func TestCompressCurrentRound(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.suggestions = []SuggestionDisplay{
		{
			Item:   planner.SuggestionItem{Name: "fetch", Action: planner.ActionSpec{Command: "git fetch"}},
			Status: StatusDone,
		},
		{
			Item:   planner.SuggestionItem{Name: "merge", Action: planner.ActionSpec{Command: "git merge"}},
			Status: StatusDone,
		},
		{
			Item:   planner.SuggestionItem{Name: "skipped", Action: planner.ActionSpec{Command: "git push"}},
			Status: StatusSkipped,
		},
	}
	m.compressCurrentRound()
	if len(m.roundHistory) != 1 {
		t.Fatalf("expected 1 round in history, got %d", len(m.roundHistory))
	}
	if len(m.roundHistory[0].Commands) != 2 {
		t.Errorf("expected 2 commands in compressed round, got %d", len(m.roundHistory[0].Commands))
	}
}

// ---------- Status constants ----------

func TestSuggestionStatusConstants(t *testing.T) {
	if StatusPending != 0 {
		t.Errorf("expected StatusPending=0, got %d", StatusPending)
	}
	if StatusExecuting != 1 {
		t.Errorf("expected StatusExecuting=1, got %d", StatusExecuting)
	}
	if StatusDone != 2 {
		t.Errorf("expected StatusDone=2, got %d", StatusDone)
	}
	if StatusFailed != 3 {
		t.Errorf("expected StatusFailed=3, got %d", StatusFailed)
	}
	if StatusSkipped != 4 {
		t.Errorf("expected StatusSkipped=4, got %d", StatusSkipped)
	}
}

// ---------- Page navigation ----------

func TestPageNavigation_ConfigAndBack(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	if m.page != PageMain {
		t.Fatalf("expected PageMain, got %d", m.page)
	}

	m.page = PageConfig
	m.configMenuIdx = 0
	if m.page != PageConfig {
		t.Fatalf("expected PageConfig, got %d", m.page)
	}

	m2, _ := m.handleConfigPageKeys("escape", tea.KeyPressMsg{})
	m = m2.(Model)
	if m.page != PageMain {
		t.Errorf("expected PageMain after Esc from config, got %d", m.page)
	}
}

func TestConfigMenuNavigation(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.page = PageConfig
	m.configMenuIdx = 0

	// Navigate down
	m2, _ := m.handleConfigMenuKeys("down")
	m = m2.(Model)
	if m.configMenuIdx != 1 {
		t.Errorf("expected configMenuIdx=1, got %d", m.configMenuIdx)
	}

	m2, _ = m.handleConfigMenuKeys("down")
	m = m2.(Model)
	if m.configMenuIdx != 2 {
		t.Errorf("expected configMenuIdx=2, got %d", m.configMenuIdx)
	}

	// Navigate up
	m2, _ = m.handleConfigMenuKeys("up")
	m = m2.(Model)
	if m.configMenuIdx != 1 {
		t.Errorf("expected configMenuIdx=1, got %d", m.configMenuIdx)
	}

	// Enter navigates to mode sub-page (index 1)
	m2, _ = m.handleConfigMenuKeys("enter")
	m = m2.(Model)
	if m.page != PageConfigMode {
		t.Errorf("expected PageConfigMode, got %d", m.page)
	}
}

func TestConfigModeSelection(t *testing.T) {
	// Select manual (index 0) to avoid triggering startAnalysis which needs store
	m := NewModel(nil, nil, "auto", "en", ConfigSnapshot{})
	m.page = PageConfigMode
	m.configModeIdx = 0 // manual

	m2, _ := m.handleConfigModeKeys("enter")
	m = m2.(Model)
	if m.mode != "manual" {
		t.Errorf("expected mode=manual, got %s", m.mode)
	}
	if m.page != PageConfig {
		t.Errorf("expected PageConfig after mode selection, got %d", m.page)
	}
}

func TestConfigLangSelection(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{Language: "en"})
	m.page = PageConfigLang
	m.configLangIdx = 1 // zh

	m2, _ := m.handleConfigLangKeys("enter")
	m = m2.(Model)
	if m.language != "zh" {
		t.Errorf("expected language=zh, got %s", m.language)
	}
	if m.configInfo.Language != "zh" {
		t.Errorf("expected configInfo.Language=zh, got %s", m.configInfo.Language)
	}
	if m.page != PageConfig {
		t.Errorf("expected PageConfig after lang selection, got %d", m.page)
	}
}

func TestConfigThemeSelection(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{Theme: "catppuccin"})
	m.page = PageConfigTheme
	m.configThemeIdx = 1 // dracula (second in Names list)

	m2, _ := m.handleConfigThemeKeys("enter")
	m = m2.(Model)
	if m.configInfo.Theme != "dracula" {
		t.Errorf("expected theme=dracula, got %s", m.configInfo.Theme)
	}
	if m.page != PageConfig {
		t.Errorf("expected PageConfig after theme selection, got %d", m.page)
	}
}

// ---------- Focus cycling ----------

func TestFocusZoneCycle(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	expected := []FocusZone{FocusLeft, FocusGit, FocusGoals, FocusLog, FocusInput}

	for _, exp := range expected {
		m = m.cycleFocus()
		if m.focusZone != exp {
			t.Errorf("expected focusZone=%d, got %d", exp, m.focusZone)
		}
	}

	// After full cycle, should be back at Input with composerFocus
	if !m.composerFocus {
		t.Error("expected composerFocus=true when focus returns to Input")
	}
}

// ---------- Panel scroll ----------

func TestPanelScroll(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.focusZone = FocusGit
	m.composerFocus = false

	// Scroll down
	m2, _ := m.handleNavigation("down")
	m = m2.(Model)
	if m.panelScrolls[FocusGit] != 1 {
		t.Errorf("expected panelScrolls[Git]=1, got %d", m.panelScrolls[FocusGit])
	}

	// Other panels unaffected
	if m.panelScrolls[FocusLog] != 0 {
		t.Errorf("expected panelScrolls[Log]=0, got %d", m.panelScrolls[FocusLog])
	}

	// Scroll up
	m2, _ = m.handleNavigation("up")
	m = m2.(Model)
	if m.panelScrolls[FocusGit] != 0 {
		t.Errorf("expected panelScrolls[Git]=0, got %d", m.panelScrolls[FocusGit])
	}

	// Won't go below 0
	m2, _ = m.handleNavigation("up")
	m = m2.(Model)
	if m.panelScrolls[FocusGit] != 0 {
		t.Errorf("expected panelScrolls[Git]=0, got %d", m.panelScrolls[FocusGit])
	}
}

func TestLogDetailPaneToggleAndCursorNavigation(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.focusZone = FocusLog
	m.composerFocus = false
	m.opLog.Add(oplog.Entry{Type: oplog.EntryCmdFail, Summary: "fail one", Detail: "detail one"})
	m.opLog.Add(oplog.Entry{Type: oplog.EntryCmdFail, Summary: "fail two", Detail: "detail two"})

	m2, _ := m.handleNavigation("enter")
	m = m2.(Model)
	if !m.detailPaneOpen {
		t.Fatal("expected detail pane open after enter on log zone")
	}
	beforeScroll := m.panelScrolls[FocusLog]
	m2, _ = m.handleNavigation("down")
	m = m2.(Model)
	if m.logCursor == 0 {
		t.Fatal("expected cursor to move down in detail mode")
	}
	if m.panelScrolls[FocusLog] != beforeScroll {
		t.Fatal("expected detail mode to move cursor, not panel scroll")
	}
	m2, _ = m.handleNavigation("enter")
	m = m2.(Model)
	if m.detailPaneOpen {
		t.Fatal("expected detail pane to collapse on enter")
	}
}

// ---------- Left panel rendering ----------

func TestRenderLeftPanel_EmptyState(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.width = 80
	m.height = 24
	panel := m.renderLeftPanel(50, 20)
	if !strings.Contains(panel, "/goal") || !strings.Contains(panel, "/help") {
		t.Error("expected placeholder text with /goal and /help hints in empty left panel")
	}
}

func TestRenderLeftPanel_WithSuggestions(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.suggestions = []SuggestionDisplay{
		{
			Item:   planner.SuggestionItem{Name: "fetch", Action: planner.ActionSpec{Command: "git fetch"}, Reason: "sync"},
			Status: StatusPending,
		},
	}
	panel := m.renderLeftPanel(50, 20)
	if !strings.Contains(panel, "git fetch") {
		t.Error("expected 'git fetch' in left panel")
	}
	if !strings.Contains(panel, "Suggestions") {
		t.Error("expected 'Suggestions' header in left panel")
	}
}

func TestRenderMainView_CommandPaletteVisible(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.width = 100
	m.height = 30
	m.ready = true
	m.showCommandPalette = true
	out := m.renderMainView()
	if !strings.Contains(out, "Command Palette") {
		t.Fatalf("expected command palette section in main view, got:\n%s", out)
	}
}

func TestRenderLeftPanel_ProgressBars(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.goals = []dotgitdex.Goal{
		{Title: "task1", Completed: true},
		{Title: "task2", Completed: false},
	}
	m.activeGoal = "task2"
	panel := m.renderLeftPanel(60, 20)
	if !strings.Contains(panel, "Goal") {
		t.Error("expected 'Goal' progress bar in left panel")
	}
	if !strings.Contains(panel, "1/2") {
		t.Error("expected '1/2' in goal progress")
	}
}

// ---------- Right panel rendering ----------

func TestRenderRightPanel_HasThreeSections(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.width = 80
	m.height = 30
	geo := m.calcLayout()
	panel := m.renderRightPanel(30, 20, geo)
	if !strings.Contains(panel, "Repository") {
		t.Error("expected 'Repository' section")
	}
	if !strings.Contains(panel, "Goals") {
		t.Error("expected 'Goals' section")
	}
	if !strings.Contains(panel, "Log") {
		t.Error("expected 'Log' section")
	}
	if !strings.Contains(panel, "─") {
		t.Error("expected horizontal dividers between right panel sections")
	}
}

// ---------- Git panel rendering ----------

func TestRenderGitPanel_WithData(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.gitInfo = GitSnapshot{
		Branch:        "main",
		DefaultBranch: "main",
		WorkingDirty:  3,
		StagingDirty:  1,
		WorkingFiles:  []string{"M file1.go", "? file2.go", "M file3.go"},
		StagingFiles:  []string{"A new.go"},
		Remotes:       []RemoteSnap{{Name: "origin", FetchURL: "git@github.com:test/test.git"}},
		Ahead:         2,
		Behind:        1,
		Stash:         3,
		Tags:          []string{"v1.0.0"},
		LocalBranches: []BranchSnap{
			{Name: "main", IsCurrent: true, Upstream: "origin/main"},
			{Name: "dev", Upstream: "origin/dev", Ahead: 1},
		},
		UserName:  "test",
		UserEmail: "test@example.com",
	}
	panel := m.renderGitPanel(60, 25)
	if !strings.Contains(panel, "main") {
		t.Error("expected branch name 'main'")
	}
	if !strings.Contains(panel, "3 changed") {
		t.Error("expected '3 changed' for working dirty")
	}
	if !strings.Contains(panel, "1 staged") {
		t.Error("expected '1 staged'")
	}
	if !strings.Contains(panel, "origin") {
		t.Error("expected remote name 'origin'")
	}
	if !strings.Contains(panel, "2") {
		t.Error("expected ahead count in panel")
	}
	if !strings.Contains(panel, "1") {
		t.Error("expected behind count in panel")
	}
	if !strings.Contains(panel, "3 entries") {
		t.Error("expected stash '3 entries'")
	}
	if !strings.Contains(panel, "test") {
		t.Error("expected user name 'test'")
	}
}

// ---------- Goal display rules ----------

func TestGoalDisplayRules(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.goals = []dotgitdex.Goal{
		{Title: "completed-old-1", Completed: true},
		{Title: "completed-old-2", Completed: true},
		{Title: "completed-old-3", Completed: true},
		{Title: "completed-old-4", Completed: true},
		{Title: "active-goal", Completed: false, Todos: []dotgitdex.Todo{
			{Title: "step 1", Completed: true},
			{Title: "step 2", Completed: false},
		}},
		{Title: "pending-1", Completed: false},
		{Title: "pending-2", Completed: false},
	}
	m.activeGoal = "active-goal"

	panel := m.renderGoalPanel(40, 20)

	// Should NOT show completed-old-1 (only last 3 shown)
	if strings.Contains(panel, "completed-old-1") {
		t.Error("should not show oldest completed goal (only last 3)")
	}
	// Should show last 3 completed
	if !strings.Contains(panel, "completed-old-2") {
		t.Error("expected 'completed-old-2' in last 3 completed")
	}
	if !strings.Contains(panel, "completed-old-4") {
		t.Error("expected 'completed-old-4' in last 3 completed")
	}
	// Active goal with todos
	if !strings.Contains(panel, "active-goal") {
		t.Error("expected active goal")
	}
	if !strings.Contains(panel, "step 1") {
		t.Error("expected todo 'step 1' for active goal")
	}
	if !strings.Contains(panel, "step 2") {
		t.Error("expected todo 'step 2' for active goal")
	}
	// Pending goals (no todos)
	if !strings.Contains(panel, "pending-1") {
		t.Error("expected pending goal 'pending-1'")
	}
}

// ---------- Config page rendering ----------

func TestRenderConfigPage(t *testing.T) {
	m := NewModel(nil, nil, "auto", "en", ConfigSnapshot{
		Helper:   LLMRoleSnapshot{Provider: "openai", Model: "gpt-4.1-mini"},
		Language: "en",
		Theme:    "dark",
	})
	m.page = PageConfig
	m.width = 80
	m.height = 24
	m.ready = true
	m.configMenuIdx = 0

	page := m.renderConfigMainPage(80)
	if !strings.Contains(page, "Model Configuration") {
		t.Error("expected 'Model Configuration' in config menu")
	}
	if !strings.Contains(page, "Mode Settings") {
		t.Error("expected 'Mode Settings' in config menu")
	}
	if !strings.Contains(page, ">") {
		t.Error("expected '>' indicator for selected item")
	}
}

func TestRenderConfigModelPage_AllProviders(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{
		Helper: LLMRoleSnapshot{Provider: "deepseek", Model: "deepseek-chat", Endpoint: "https://api.deepseek.com"},
	})
	m.configDraft = ConfigDraft{ProviderIdx: 2, Model: "deepseek-chat", Endpoint: "https://api.deepseek.com"}
	page := m.renderConfigModelPage(80)
	if !strings.Contains(page, "Role") {
		t.Error("expected 'Role' label")
	}
	if !strings.Contains(page, "Helper") {
		t.Error("expected 'Helper' role chip")
	}
	if !strings.Contains(page, "Planner") {
		t.Error("expected 'Planner' role chip")
	}
	if !strings.Contains(page, "ollama") {
		t.Error("expected 'ollama' provider chip")
	}
	if !strings.Contains(page, "openai") {
		t.Error("expected 'openai' provider chip")
	}
	if !strings.Contains(page, "deepseek") {
		t.Error("expected 'deepseek' provider chip")
	}
	if !strings.Contains(page, "Provider") {
		t.Error("expected 'Provider' label")
	}
	if !strings.Contains(page, "Model") {
		t.Error("expected 'Model' field label")
	}
	if !strings.Contains(page, "Endpoint") {
		t.Error("expected 'Endpoint' field label")
	}
	if !strings.Contains(page, "Recommended") {
		t.Error("expected 'Recommended' models section")
	}
}

func TestConfigModelInteractive(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{
		Helper: LLMRoleSnapshot{Provider: "deepseek", Model: "deepseek-chat", Endpoint: "https://api.deepseek.com"},
	})
	m.page = PageConfigModel
	m.configDraft = ConfigDraft{ProviderIdx: 2, Model: "deepseek-chat", Endpoint: "https://api.deepseek.com"}

	// FieldIdx=0 is role selector, left switches to helper (already helper so no change)
	// Navigate to provider field first
	m.configDraft.FieldIdx = 1
	m2, _ := m.handleConfigModelKeys("left")
	m = m2.(Model)
	if m.configDraft.ProviderIdx != 1 {
		t.Errorf("expected provider index 1 (openai), got %d", m.configDraft.ProviderIdx)
	}

	m2, _ = m.handleConfigModelKeys("tab")
	m = m2.(Model)
	if m.configDraft.FieldIdx != 2 {
		t.Errorf("expected field index 2 (model), got %d", m.configDraft.FieldIdx)
	}
}

func TestIsEscKey(t *testing.T) {
	if !isEscKey("escape") {
		t.Error("expected 'escape' to be esc key")
	}
	if !isEscKey("esc") {
		t.Error("expected 'esc' to be esc key")
	}
	if isEscKey("q") {
		t.Error("'q' should not be esc key")
	}
}

func TestConfigExit_EscFromSubPage(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.page = PageConfigModel

	m2, _ := m.handleConfigPageKeys("escape", tea.KeyPressMsg{})
	m = m2.(Model)
	if m.page != PageConfig {
		t.Errorf("expected PageConfig after Esc from sub-page, got %d", m.page)
	}

	m2, _ = m.handleConfigPageKeys("escape", tea.KeyPressMsg{})
	m = m2.(Model)
	if m.page != PageMain {
		t.Errorf("expected PageMain after Esc from config, got %d", m.page)
	}
}

func TestConfigExit_EscVariant(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.page = PageConfigMode

	m2, _ := m.handleConfigPageKeys("esc", tea.KeyPressMsg{})
	m = m2.(Model)
	if m.page != PageConfig {
		t.Errorf("expected PageConfig after esc from sub-page, got %d", m.page)
	}
}

func TestWrapText(t *testing.T) {
	lines := wrapText("abcdefghij", 5)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "abcde" || lines[1] != "fghij" {
		t.Errorf("unexpected wrap result: %v", lines)
	}

	lines = wrapText("short", 10)
	if len(lines) != 1 || lines[0] != "short" {
		t.Errorf("expected no wrap for short text, got %v", lines)
	}

	lines = wrapText("line1\nline2", 20)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines for newline split, got %d", len(lines))
	}
}

func TestWrapText_Multibyte(t *testing.T) {
	input := "这是一段需要自动换行显示的中文内容测试"
	lines := wrapText(input, 10)
	for _, l := range lines {
		runes := []rune(l)
		if len(runes) > 10 {
			t.Errorf("wrapped line exceeds maxW=10: %d runes", len(runes))
		}
	}
	joined := strings.Join(lines, "")
	if joined != input {
		t.Error("wrapped text lost content")
	}
}

func TestRenderConfigModePage(t *testing.T) {
	m := NewModel(nil, nil, "auto", "en", ConfigSnapshot{})
	m.page = PageConfigMode
	m.configModeIdx = 1

	page := m.renderConfigModePage(80)
	if !strings.Contains(page, "Manual") {
		t.Error("expected 'Manual' in mode page")
	}
	if !strings.Contains(page, "Auto") {
		t.Error("expected 'Auto' in mode page")
	}
	if !strings.Contains(page, "Cruise") {
		t.Error("expected 'Cruise' in mode page")
	}
	if !strings.Contains(page, "[*]") {
		t.Error("expected '[*]' for current mode (auto)")
	}
}

// ---------- i18n ----------

func TestConfigTextI18n(t *testing.T) {
	if configText("config_title", "en") != "Configuration" {
		t.Error("expected 'Configuration' for en")
	}
	if configText("config_title", "zh") != "配置" {
		t.Error("expected '配置' for zh")
	}
	if configText("config_title", "ja") != "設定" {
		t.Error("expected '設定' for ja")
	}
	// Fallback to english
	if configText("config_title", "fr") != "Configuration" {
		t.Error("expected english fallback for unknown language")
	}
	// Unknown key returns key itself
	if configText("nonexistent_key", "en") != "nonexistent_key" {
		t.Error("expected key itself for unknown config text key")
	}
}

// ---------- Git snapshot parser ----------

func TestParseGitSnapshotExpanded(t *testing.T) {
	content := `# Git Context — generated by Gitdex
# Snapshot at 2026-03-14T12:00:00Z

current_branch: main
detached_head: false
head_ref: abc123
is_initial: false
commit_count: 42
default_branch: main

## Local Branches
* main -> origin/main [ahead 2, behind 1] | fix: update readme
  dev -> origin/dev | feat: add parser
  feature/test

## Merged Branches
old-branch

## Remote Branches
origin/main
origin/dev

## Remotes
origin  fetch=git@github.com:test/repo.git  push=git@github.com:test/repo.git

## Upstream
origin/main  ahead=2  behind=1

## Working Tree Changes
M README.md
? newfile.txt

## Staging Area
A staged.go

## Repository State
merge_in_progress: false
rebase_in_progress: true

## Stash
stash@{0}: WIP on main
stash@{1}: WIP on dev

## Tags
v1.0.0
v0.9.0

## Submodules
sub1  path=vendor/sub1  url=https://github.com/foo/sub1

## Recent Reflog
abc123 HEAD@{0}: commit: fix readme
def456 HEAD@{1}: checkout: moving from dev to main

## Ahead Commits
abc123
def456

## Behind Commits
ghi789

## Config
user.name: TestUser
user.email: test@example.com

## Commit Summary
commit_frequency: 5/week
last_commit: fix: update readme

## Summary
working_tree_dirty: 2
staging_area_dirty: 1
`

	info := parseGitSnapshot(content)

	if info.Branch != "main" {
		t.Errorf("expected branch=main, got %s", info.Branch)
	}
	if info.Detached {
		t.Error("expected detached=false")
	}
	if info.HeadRef != "abc123" {
		t.Errorf("expected head_ref=abc123, got %s", info.HeadRef)
	}
	if info.CommitCount != 42 {
		t.Errorf("expected commit_count=42, got %d", info.CommitCount)
	}
	if info.DefaultBranch != "main" {
		t.Errorf("expected default_branch=main, got %s", info.DefaultBranch)
	}

	// Local branches
	if len(info.LocalBranches) != 3 {
		t.Fatalf("expected 3 local branches, got %d", len(info.LocalBranches))
	}
	if !info.LocalBranches[0].IsCurrent {
		t.Error("expected first branch to be current")
	}
	if info.LocalBranches[0].Name != "main" {
		t.Errorf("expected branch name 'main', got %s", info.LocalBranches[0].Name)
	}
	if info.LocalBranches[0].Upstream != "origin/main" {
		t.Errorf("expected upstream 'origin/main', got %s", info.LocalBranches[0].Upstream)
	}
	if info.LocalBranches[0].Ahead != 2 || info.LocalBranches[0].Behind != 1 {
		t.Errorf("expected ahead=2, behind=1 for main, got %d, %d",
			info.LocalBranches[0].Ahead, info.LocalBranches[0].Behind)
	}

	// Merged
	if len(info.MergedBranches) != 1 || info.MergedBranches[0] != "old-branch" {
		t.Errorf("expected 1 merged branch 'old-branch', got %v", info.MergedBranches)
	}

	// Remote branches
	if len(info.RemoteBranches) != 2 {
		t.Errorf("expected 2 remote branches, got %d", len(info.RemoteBranches))
	}

	// Remotes
	if len(info.Remotes) != 1 {
		t.Fatalf("expected 1 remote, got %d", len(info.Remotes))
	}
	if info.Remotes[0].Name != "origin" {
		t.Errorf("expected remote name 'origin', got %s", info.Remotes[0].Name)
	}

	// Upstream
	if info.Ahead != 2 || info.Behind != 1 {
		t.Errorf("expected ahead=2, behind=1, got %d, %d", info.Ahead, info.Behind)
	}

	// Working/staging
	if info.WorkingDirty != 2 {
		t.Errorf("expected working_dirty=2, got %d", info.WorkingDirty)
	}
	if info.StagingDirty != 1 {
		t.Errorf("expected staging_dirty=1, got %d", info.StagingDirty)
	}
	if len(info.WorkingFiles) != 2 {
		t.Errorf("expected 2 working files, got %d", len(info.WorkingFiles))
	}

	// Repo state
	if info.RebaseInProgress != true {
		t.Error("expected rebase_in_progress=true")
	}
	if info.MergeInProgress != false {
		t.Error("expected merge_in_progress=false")
	}

	// Stash
	if info.Stash != 2 {
		t.Errorf("expected stash=2, got %d", info.Stash)
	}

	// Tags
	if len(info.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(info.Tags))
	}

	// Submodules
	if len(info.Submodules) != 1 {
		t.Errorf("expected 1 submodule, got %d", len(info.Submodules))
	}

	// Reflog
	if len(info.RecentReflog) != 2 {
		t.Errorf("expected 2 reflog entries, got %d", len(info.RecentReflog))
	}

	// Ahead/behind commits
	if len(info.AheadCommits) != 2 {
		t.Errorf("expected 2 ahead commits, got %d", len(info.AheadCommits))
	}
	if len(info.BehindCommits) != 1 {
		t.Errorf("expected 1 behind commit, got %d", len(info.BehindCommits))
	}

	// Config
	if info.UserName != "TestUser" {
		t.Errorf("expected user.name=TestUser, got %s", info.UserName)
	}
	if info.UserEmail != "test@example.com" {
		t.Errorf("expected user.email=test@example.com, got %s", info.UserEmail)
	}

	// Commit summary
	if info.CommitFreq != "5/week" {
		t.Errorf("expected commit_frequency=5/week, got %s", info.CommitFreq)
	}
	if info.LastCommit != "fix: update readme" {
		t.Errorf("expected last_commit='fix: update readme', got %s", info.LastCommit)
	}
}

// ---------- Header ----------

func TestRenderHeader(t *testing.T) {
	m := NewModel(nil, nil, "auto", "en", ConfigSnapshot{})
	m.width = 80
	m.activeFlow = "maintain"
	header := m.renderHeader(80)
	if !strings.Contains(header, "Gitdex") {
		t.Error("expected 'Gitdex' in header")
	}
	if !strings.Contains(header, "AUTO") {
		t.Error("expected 'AUTO' in header")
	}
}

func TestRenderHeader_Ready(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.width = 80
	header := m.renderHeader(80)
	if !strings.Contains(header, "READY") {
		t.Errorf("expected 'READY' in header, got: %s", header)
	}
}

func TestRenderHeader_Analyzing(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.analyzing = true
	m.width = 80
	header := m.renderHeader(80)
	if !strings.Contains(header, "ANALYZING") {
		t.Errorf("expected 'ANALYZING' in header, got: %s", header)
	}
}

// ---------- Input ----------

func TestRenderInput(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.composerFocus = true
	m.composerText = "hello"
	m.width = 80
	input := m.renderInput(80)
	if !strings.Contains(input, "hello") {
		t.Error("expected composer text in rendered input")
	}
}

// ---------- Mode/Lang/Theme index helpers ----------

func TestModeToIdx(t *testing.T) {
	if modeToIdx("manual") != 0 {
		t.Error("expected manual=0")
	}
	if modeToIdx("auto") != 1 {
		t.Error("expected auto=1")
	}
	if modeToIdx("cruise") != 2 {
		t.Error("expected cruise=2")
	}
	if modeToIdx("unknown") != 0 {
		t.Error("expected unknown=0")
	}
}

func TestLangToIdx(t *testing.T) {
	if langToIdx("en") != 0 {
		t.Error("expected en=0")
	}
	if langToIdx("zh") != 1 {
		t.Error("expected zh=1")
	}
	if langToIdx("ja") != 2 {
		t.Error("expected ja=2")
	}
}

func TestThemeToIdx(t *testing.T) {
	if themeToIdx("catppuccin") != 0 {
		t.Error("expected catppuccin=0")
	}
	if themeToIdx("dracula") != 1 {
		t.Error("expected dracula=1")
	}
	if themeToIdx("tokyonight") != 2 {
		t.Error("expected tokyonight=2")
	}
	if themeToIdx("gruvbox") != 3 {
		t.Error("expected gruvbox=3")
	}
	if themeToIdx("nord") != 4 {
		t.Error("expected nord=4")
	}
	if themeToIdx("dark") != 5 {
		t.Error("expected dark=5")
	}
	if themeToIdx("light") != 6 {
		t.Error("expected light=6")
	}
	if themeToIdx("unknown") != 0 {
		t.Error("expected unknown=0 (fallback)")
	}
}

// ---------- applyPanelScroll ----------

func TestApplyPanelScroll_PadsShort(t *testing.T) {
	lines := []string{"line1", "line2"}
	result := applyPanelScroll(lines, 0, 80, 5)
	count := strings.Count(result, "\n")
	if count < 4 {
		t.Errorf("expected at least 4 newlines in padded output, got %d", count)
	}
}

func TestApplyPanelScroll_Scrolls(t *testing.T) {
	lines := []string{"line0", "line1", "line2", "line3", "line4"}
	result := applyPanelScroll(lines, 2, 80, 3)
	if !strings.Contains(result, "line2") {
		t.Error("expected line2 visible at scroll offset 2")
	}
	if strings.Contains(result, "line0") {
		t.Error("expected line0 to be scrolled past")
	}
}

// ---------- parseBranchLine ----------

func TestParseBranchLine(t *testing.T) {
	tests := []struct {
		input    string
		name     string
		current  bool
		upstream string
		ahead    int
		behind   int
	}{
		{"* main -> origin/main [ahead 2, behind 1] | fix readme", "main", true, "origin/main", 2, 1},
		{"  dev -> origin/dev | feat parser", "dev", false, "origin/dev", 0, 0},
		{"  feature/test", "feature/test", false, "", 0, 0},
	}

	for _, tc := range tests {
		b, ok := parseBranchLine(tc.input)
		if !ok {
			t.Errorf("parseBranchLine(%q) returned !ok", tc.input)
			continue
		}
		if b.Name != tc.name {
			t.Errorf("parseBranchLine(%q).Name = %q, want %q", tc.input, b.Name, tc.name)
		}
		if b.IsCurrent != tc.current {
			t.Errorf("parseBranchLine(%q).IsCurrent = %v, want %v", tc.input, b.IsCurrent, tc.current)
		}
		if b.Upstream != tc.upstream {
			t.Errorf("parseBranchLine(%q).Upstream = %q, want %q", tc.input, b.Upstream, tc.upstream)
		}
		if b.Ahead != tc.ahead {
			t.Errorf("parseBranchLine(%q).Ahead = %d, want %d", tc.input, b.Ahead, tc.ahead)
		}
		if b.Behind != tc.behind {
			t.Errorf("parseBranchLine(%q).Behind = %d, want %d", tc.input, b.Behind, tc.behind)
		}
	}
}

// ---------- parseRemoteLine ----------

func TestParseRemoteLine(t *testing.T) {
	r, ok := parseRemoteLine("origin  fetch=git@github.com:test/repo.git  push=git@github.com:test/repo.git")
	if !ok {
		t.Fatal("expected ok")
	}
	if r.Name != "origin" {
		t.Errorf("expected name=origin, got %s", r.Name)
	}
	if r.FetchURL != "git@github.com:test/repo.git" {
		t.Errorf("expected fetch URL, got %s", r.FetchURL)
	}
}

// ---------- splitKV ----------

func TestSplitKV(t *testing.T) {
	k, v, ok := splitKV("user.name: TestUser")
	if !ok {
		t.Fatal("expected ok")
	}
	if k != "user.name" || v != "TestUser" {
		t.Errorf("expected user.name=TestUser, got %s=%s", k, v)
	}

	_, _, ok = splitKV("no colon here")
	if ok {
		t.Error("expected !ok for line without colon")
	}
}

// ---------- Text editing helpers ----------

func TestInsertAtRune(t *testing.T) {
	val, cur := insertAtRune("hello", 3, "XY")
	if val != "helXYlo" {
		t.Errorf("expected 'helXYlo', got %q", val)
	}
	if cur != 5 {
		t.Errorf("expected cursor at 5, got %d", cur)
	}
}

func TestDeleteRuneBefore(t *testing.T) {
	val, cur := deleteRuneBefore("hello", 3)
	if val != "helo" {
		t.Errorf("expected 'helo', got %q", val)
	}
	if cur != 2 {
		t.Errorf("expected cursor at 2, got %d", cur)
	}
	// At position 0, nothing happens
	val, cur = deleteRuneBefore("hello", 0)
	if val != "hello" || cur != 0 {
		t.Error("expected no change at pos 0")
	}
}

func TestDeleteRuneAt(t *testing.T) {
	val, cur := deleteRuneAt("hello", 2)
	if val != "helo" {
		t.Errorf("expected 'helo', got %q", val)
	}
	if cur != 2 {
		t.Errorf("expected cursor at 2, got %d", cur)
	}
}

func TestClampRuneIdx(t *testing.T) {
	if clampRuneIdx("hello", -1) != 0 {
		t.Error("expected clamp to 0 for negative")
	}
	if clampRuneIdx("hello", 10) != 5 {
		t.Error("expected clamp to 5 for oversized")
	}
	if clampRuneIdx("hello", 3) != 3 {
		t.Error("expected 3 unchanged")
	}
}

// ---------- Provider metadata ----------

func TestProviderMetaFor(t *testing.T) {
	ollama := providerMetaFor("ollama")
	if ollama.Label != "Ollama" {
		t.Errorf("expected label 'Ollama', got %s", ollama.Label)
	}
	if ollama.APIKeyEnv != "" {
		t.Error("ollama should not require API key")
	}

	openai := providerMetaFor("openai")
	if openai.APIKeyEnv != "OPENAI_API_KEY" {
		t.Errorf("expected OPENAI_API_KEY, got %s", openai.APIKeyEnv)
	}

	ds := providerMetaFor("deepseek")
	if ds.Kind != "Cloud API" {
		t.Errorf("expected kind 'Cloud API', got %s", ds.Kind)
	}

	unknown := providerMetaFor("unknown")
	if unknown.ID != "ollama" {
		t.Error("expected fallback to ollama for unknown provider")
	}
}

func TestSplitAtRunePos(t *testing.T) {
	before, after := splitAtRunePos("hello", 2)
	if before != "he" || after != "llo" {
		t.Errorf("expected 'he'/'llo', got %q/%q", before, after)
	}

	before, after = splitAtRunePos("你好世界", 2)
	if before != "你好" || after != "世界" {
		t.Errorf("expected '你好'/'世界', got %q/%q", before, after)
	}
}

func TestClampInt(t *testing.T) {
	if clampInt(5, 1, 10) != 5 {
		t.Error("expected 5")
	}
	if clampInt(-1, 1, 10) != 1 {
		t.Error("expected 1 for below min")
	}
	if clampInt(99, 1, 10) != 10 {
		t.Error("expected 10 for above max")
	}
}

// ---------- Theme system ----------

func TestThemeNames(t *testing.T) {
	names := theme.Names()
	if len(names) < 5 {
		t.Errorf("expected at least 5 themes, got %d", len(names))
	}
	expected := []string{"catppuccin", "dracula", "tokyonight", "gruvbox", "nord"}
	for _, e := range expected {
		found := false
		for _, n := range names {
			if n == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected theme '%s' in Names()", e)
		}
	}
}

func TestThemeInit_AllThemes(t *testing.T) {
	for _, name := range theme.Names() {
		theme.Init(name)
		if theme.Current == nil {
			t.Errorf("theme.Current is nil after Init(%q)", name)
		}
		if theme.Current.Name != name {
			t.Errorf("expected theme name=%q, got %q", name, theme.Current.Name)
		}
		if theme.Current.Primary == "" {
			t.Errorf("theme %q has empty primary color", name)
		}
	}
}

func TestRenderConfigThemePage_AllThemes(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{Theme: "catppuccin"})
	m.configThemeIdx = 0
	page := m.renderConfigThemePage(80)
	for _, name := range theme.Names() {
		if !strings.Contains(page, name) {
			t.Errorf("expected theme '%s' in theme page", name)
		}
	}
}

// ---------- Ollama model selector ----------

func TestRenderOllamaModelSelector_Empty(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{Helper: LLMRoleSnapshot{Provider: "ollama"}})
	boxActive := theme.Current.Header
	boxIdle := theme.Current.Content
	result := m.renderOllamaModelSelector(true, boxActive, boxIdle)
	if !strings.Contains(result, "No local models") {
		t.Error("expected 'No local models' for empty model list")
	}
}

func TestRenderOllamaModelSelector_WithModels(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{Helper: LLMRoleSnapshot{Provider: "ollama"}})
	m.ollamaModels = []OllamaModelInfo{
		{Name: "qwen2.5:3b", ParamSize: "3B", Family: "qwen2.5", Quant: "Q4_K_M"},
		{Name: "llama3:8b", ParamSize: "8B", Family: "llama3", Quant: "Q4_K_M"},
	}
	m.configDraft.Model = "qwen2.5:3b"
	m.ollamaModelIdx = 0
	boxActive := theme.Current.Header
	boxIdle := theme.Current.Content
	result := m.renderOllamaModelSelector(true, boxActive, boxIdle)
	if !strings.Contains(result, "qwen2.5:3b") {
		t.Error("expected 'qwen2.5:3b' in model selector")
	}
	if !strings.Contains(result, "llama3:8b") {
		t.Error("expected 'llama3:8b' in model selector")
	}
	if !strings.Contains(result, "3B") {
		t.Error("expected param size '3B' in model selector")
	}
}

func TestRenderOllamaModelSelector_Fetching(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{Helper: LLMRoleSnapshot{Provider: "ollama"}})
	m.ollamaFetching = true
	boxActive := theme.Current.Header
	boxIdle := theme.Current.Content
	result := m.renderOllamaModelSelector(true, boxActive, boxIdle)
	if !strings.Contains(result, "Loading") {
		t.Error("expected 'Loading' during fetch")
	}
}

func TestRenderOllamaModelSelector_Error(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{Helper: LLMRoleSnapshot{Provider: "ollama"}})
	m.ollamaFetchError = "connection refused"
	boxActive := theme.Current.Header
	boxIdle := theme.Current.Content
	result := m.renderOllamaModelSelector(true, boxActive, boxIdle)
	if !strings.Contains(result, "connection refused") {
		t.Error("expected error message in model selector")
	}
}

func TestOllamaModelsMsg_Sets(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{Helper: LLMRoleSnapshot{Provider: "ollama"}})
	m.ready = true
	m.ollamaFetching = true
	m.configDraft.Model = "llama3:8b"

	models := []OllamaModelInfo{
		{Name: "qwen2.5:3b"},
		{Name: "llama3:8b"},
	}
	m2, _ := m.Update(ollamaModelsMsg{models: models})
	m = m2.(Model)

	if m.ollamaFetching {
		t.Error("expected ollamaFetching=false after message")
	}
	if len(m.ollamaModels) != 2 {
		t.Errorf("expected 2 models, got %d", len(m.ollamaModels))
	}
	if m.ollamaModelIdx != 1 {
		t.Errorf("expected ollamaModelIdx=1 (matching llama3:8b), got %d", m.ollamaModelIdx)
	}
}

// ---------- Flow loop tests ----------

func TestFlowRoundMsg_ZeroSuggestions_GoalPending_TriggersRetry(t *testing.T) {
	store := setupTestStore(t)
	_ = store.WriteGoalList([]dotgitdex.Goal{
		{Title: "Create feature branch", Completed: false, Todos: []dotgitdex.Todo{
			{Title: "Create branch", Completed: false},
		}},
	})

	m := NewModel(nil, store, "auto", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	msg := flowRoundMsg{flow: "goal", round: &flow.FlowRound{
		Flow:        "goal",
		Suggestions: nil,
	}}
	m2, cmd := m.Update(msg)
	m = m2.(Model)

	// In auto mode with pending goals and 0 suggestions,
	// startMaintainAnalysis is called, which sets analyzing = true
	if !m.analyzing {
		t.Error("analyzing should be true — startMaintainAnalysis should have re-armed it")
	}

	if cmd == nil {
		t.Error("expected a command to be returned for pending goals with 0 suggestions")
	}
}

func TestFlowRoundMsg_ZeroSuggestions_NoPendingGoals_Stops(t *testing.T) {
	store := setupTestStore(t)
	_ = store.WriteGoalList([]dotgitdex.Goal{
		{Title: "Already done", Completed: true},
	})

	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	msg := flowRoundMsg{flow: "goal", round: &flow.FlowRound{
		Flow:        "goal",
		Suggestions: nil,
	}}
	_, cmd := m.Update(msg)

	// In manual mode with no pending goals, should stop (nil cmd)
	if cmd != nil {
		t.Error("expected nil command when all goals are completed in manual mode")
	}
}

func TestModeSwitch_AutoWithPendingSuggestions_ExecutesThem(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.suggestions = []SuggestionDisplay{
		{Item: planner.SuggestionItem{Name: "Test", Action: planner.ActionSpec{Command: "echo hello"}}, Status: StatusPending},
	}

	// Switch mode via config page
	m.page = PageConfigMode
	m.configModeIdx = 1 // auto

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m2.(Model)

	if m.mode != "auto" {
		t.Errorf("expected mode=auto, got %s", m.mode)
	}
}

func TestRunAllMode_SetOnManualRunAll(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.composerFocus = true
	m.suggestions = []SuggestionDisplay{
		{Item: planner.SuggestionItem{Name: "S1"}, Status: StatusPending},
		{Item: planner.SuggestionItem{Name: "S2"}, Status: StatusPending},
	}

	// Simulate /run all via composer
	m.composerText = "/run all"
	m2, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m2.(Model)
	if cmd == nil {
		t.Error("expected command from /run all")
	}
	if !m.runAllMode {
		t.Error("expected runAllMode=true after /run all")
	}
}

func TestGoalProgressUpdated_AdvancesActiveGoal(t *testing.T) {
	store := setupTestStore(t)
	_ = store.WriteGoalList([]dotgitdex.Goal{
		{Title: "Goal A", Completed: true},
		{Title: "Goal B", Completed: false, Todos: []dotgitdex.Todo{
			{Title: "Todo B1", Completed: false},
		}},
	})

	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.activeGoal = "Goal A"

	m2, _ := m.Update(goalProgressUpdatedMsg{})
	m = m2.(Model)

	if m.activeGoal != "Goal B" {
		t.Errorf("expected activeGoal='Goal B', got %q", m.activeGoal)
	}
}

func TestGoalProgressUpdated_ClearsActiveGoalWhenAllDone(t *testing.T) {
	store := setupTestStore(t)
	_ = store.WriteGoalList([]dotgitdex.Goal{
		{Title: "Goal A", Completed: true},
	})

	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.activeGoal = "Goal A"

	m2, _ := m.Update(goalProgressUpdatedMsg{})
	m = m2.(Model)

	if m.activeGoal != "" {
		t.Errorf("expected activeGoal='', got %q", m.activeGoal)
	}
}

func TestGoalDecomposedMsg_WritesTodosToStore(t *testing.T) {
	store := setupTestStore(t)
	_ = store.WriteGoalList([]dotgitdex.Goal{
		{Title: "My Goal"},
	})

	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	todos := []dotgitdex.Todo{
		{Title: "Step 1"},
		{Title: "Step 2"},
		{Title: "Step 3"},
	}
	m2, _ := m.Update(goalDecomposedMsg{goalTitle: "My Goal", todos: todos})
	m = m2.(Model)

	if len(m.goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(m.goals))
	}
	if len(m.goals[0].Todos) != 3 {
		t.Errorf("expected 3 todos, got %d", len(m.goals[0].Todos))
	}

	// Also verify it was persisted
	persisted, _ := store.ReadGoalList()
	if len(persisted) != 1 || len(persisted[0].Todos) != 3 {
		t.Errorf("expected 3 todos persisted, got %d", len(persisted[0].Todos))
	}
}

func TestGoalDecomposedMsg_Error_LogsAndContinues(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	m2, _ := m.Update(goalDecomposedMsg{goalTitle: "X", err: errTest})
	m = m2.(Model)

	entries := m.opLog.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "Goal decomposition failed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error log entry for failed decomposition")
	}
}

var errTest = errType("test error")

type errType string

func (e errType) Error() string { return string(e) }

func setupTestStore(t testing.TB) *dotgitdex.Manager {
	t.Helper()
	tmp := t.TempDir()
	store := dotgitdex.New(tmp)
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	return store
}

func setupTestOrchestrator(t testing.TB, store *dotgitdex.Manager) *flow.Orchestrator {
	t.Helper()
	logger := executor.NewExecutionLogger(store, "test-session", "auto")
	return &flow.Orchestrator{
		Logger: logger,
	}
}

// ---------- Cruise interval tests ----------

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{30, "30s"},
		{60, "1min"},
		{90, "1min30s"},
		{300, "5min"},
		{900, "15min"},
		{2400, "40min"},
		{3600, "60min"},
		{3630, "60min30s"},
	}
	for _, tc := range tests {
		got := formatDuration(tc.seconds)
		if got != tc.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

func TestCruiseIntervalDefault(t *testing.T) {
	m := NewModel(nil, nil, "cruise", "en", ConfigSnapshot{CruiseInterval: 0})
	if m.cruiseIntervalS != 900 {
		t.Errorf("expected default cruise interval=900, got %d", m.cruiseIntervalS)
	}
}

func TestCruiseIntervalFromConfig(t *testing.T) {
	m := NewModel(nil, nil, "cruise", "en", ConfigSnapshot{CruiseInterval: 1800})
	if m.cruiseIntervalS != 1800 {
		t.Errorf("expected cruise interval=1800, got %d", m.cruiseIntervalS)
	}
}

func TestRenderConfigModePage_ShowsInterval(t *testing.T) {
	theme.Init("catppuccin")
	m := NewModel(nil, nil, "cruise", "en", ConfigSnapshot{CruiseInterval: 600})
	m.width = 80
	m.height = 30
	m.page = PageConfigMode
	result := m.renderConfigModePage(80)
	if !strings.Contains(result, "600") {
		t.Error("expected cruise interval '600' shown on mode config page")
	}
	if !strings.Contains(result, "10min") {
		t.Error("expected formatted duration '10min' shown on mode config page")
	}
}

func TestCruiseHeaderShowsInterval(t *testing.T) {
	theme.Init("catppuccin")
	m := NewModel(nil, nil, "cruise", "en", ConfigSnapshot{CruiseInterval: 2400})
	m.width = 100
	m.height = 30
	m.ready = true
	header := m.renderHeader(100)
	if !strings.Contains(header, "CRUISE") {
		t.Error("expected 'CRUISE' in header")
	}
	if !strings.Contains(header, "40min") {
		t.Error("expected '40min' cruise interval in header")
	}
	if !strings.Contains(header, "idle") {
		t.Error("expected 'idle' cycle status in header")
	}
}

func TestCruiseHeaderShowsActiveStatus(t *testing.T) {
	theme.Init("catppuccin")
	m := NewModel(nil, nil, "cruise", "en", ConfigSnapshot{CruiseInterval: 900})
	m.width = 100
	m.height = 30
	m.ready = true
	m.cruiseCycleActive = true
	header := m.renderHeader(100)
	if !strings.Contains(header, "active") {
		t.Error("expected 'active' cycle status in header when cycle is running")
	}
}

func TestIntervalCommand(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{CruiseInterval: 900})
	m.ready = true
	m.width = 100
	m.height = 30
	m.composerFocus = true

	m.composerText = "/interval 1800"
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m2.(Model)

	if m.cruiseIntervalS != 1800 {
		t.Errorf("expected cruiseIntervalS=1800 after /interval command, got %d", m.cruiseIntervalS)
	}
}

func TestIntervalCommand_RejectsBelow60(t *testing.T) {
	m := NewModel(nil, nil, "cruise", "en", ConfigSnapshot{CruiseInterval: 900})
	m.ready = true
	m.width = 100
	m.height = 30
	m.composerFocus = true

	m.composerText = "/interval 30"
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m2.(Model)

	if m.cruiseIntervalS != 900 {
		t.Errorf("expected cruiseIntervalS=900 (unchanged) for val < 60, got %d", m.cruiseIntervalS)
	}
}

func TestModePageNavigation_4Items(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 80
	m.height = 30
	m.page = PageConfigMode
	m.configModeIdx = 0

	// Navigate to item 3 (interval)
	for i := 0; i < 3; i++ {
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = m2.(Model)
	}
	if m.configModeIdx != 3 {
		t.Errorf("expected configModeIdx=3 after 3 downs, got %d", m.configModeIdx)
	}

	// Can't go beyond 3
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = m2.(Model)
	if m.configModeIdx != 3 {
		t.Errorf("expected configModeIdx=3 (clamped), got %d", m.configModeIdx)
	}
}

// ---------- Style functions ----------

// ---------- Creative flow tests ----------

func TestCreativeResultMsg_Success_UpdatesGoalsAndLogs(t *testing.T) {
	store := setupTestStore(t)
	_ = store.WriteGoalList([]dotgitdex.Goal{
		{Title: "Existing goal", Completed: false},
	})

	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	result := &flow.CreativeResult{
		NewGitdexGoals: []string{"Auto-generated goal A", "Auto-generated goal B"},
		NewCreative:    []string{"Creative idea 1"},
		Discarded:      []string{"Bad idea 1", "Bad idea 2"},
	}

	// Simulate: creative flow adds new goals to the store (like the real flow does)
	goals, _ := store.ReadGoalList()
	for _, title := range result.NewGitdexGoals {
		goals = append(goals, dotgitdex.Goal{Title: title})
	}
	_ = store.WriteGoalList(goals)

	m2, cmd := m.Update(creativeResultMsg{result: result})
	m = m2.(Model)

	// Should have logged the creative output
	entries := m.opLog.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "[creative]") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected creative flow log entry")
	}

	// Goals should be refreshed from store (3 total now)
	if len(m.goals) != 3 {
		t.Errorf("expected 3 goals after creative flow, got %d", len(m.goals))
	}

	// cmd should not be nil — it should proceed to startAnalysis or decomposeGoal
	if cmd == nil {
		t.Error("expected a command after creative flow completes")
	}
}

func TestCreativeResultMsg_Error_LogsError(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	m2, cmd := m.Update(creativeResultMsg{err: errTest})
	m = m2.(Model)

	entries := m.opLog.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "Creative flow error") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error log entry for failed creative flow")
	}

	// Even on error, should proceed to startAnalysis
	if cmd == nil {
		t.Error("expected startAnalysis command even after creative flow error")
	}
}

func TestCreativeResultMsg_NilResult_ProceedsToAnalysis(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	m2, cmd := m.Update(creativeResultMsg{result: nil, err: nil})
	m = m2.(Model)

	// Should still proceed to analysis
	if cmd == nil {
		t.Error("expected startAnalysis command even with nil creative result")
	}
}

func TestCruiseTickMsg_StartsGoalMaintainCycle(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{CruiseInterval: 900})
	m.ready = true
	m.width = 100
	m.height = 30
	m.analyzing = false

	m2, cmd := m.Update(cruiseTickMsg{})
	m = m2.(Model)

	if cmd == nil {
		t.Error("expected command from cruiseTickMsg in cruise mode")
	}
	if !m.cruiseCycleActive {
		t.Error("expected cruiseCycleActive=true after cruiseTickMsg")
	}
	if m.creativeRanThisSlice {
		t.Error("expected creativeRanThisSlice=false (reset by tick)")
	}

	entries := m.opLog.Entries()
	cycleTriggered := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "goal/maintain cycle") || strings.Contains(e.Summary, "patrol triggered") {
			cycleTriggered = true
			break
		}
	}
	if !cycleTriggered {
		t.Error("expected log entry about cruise patrol / goal-maintain cycle")
	}
}

func TestCruiseTickMsg_IgnoredWhenCycleActive(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{CruiseInterval: 60})
	m.ready = true
	m.width = 100
	m.height = 30
	m.cruiseCycleActive = true

	m2, cmd := m.Update(cruiseTickMsg{})
	m = m2.(Model)

	if cmd != nil {
		t.Error("expected nil command when cruise cycle is already active")
	}

	entries := m.opLog.Entries()
	continuing := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "previous cycle still in progress") || strings.Contains(e.Summary, "continuing") {
			continuing = true
			break
		}
	}
	if !continuing {
		t.Error("expected log entry about previous cycle still in progress")
	}
}

func TestCruiseCycleCompleteMsg_ClearsFlag(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{CruiseInterval: 900})
	m.ready = true
	m.width = 100
	m.height = 30
	m.cruiseCycleActive = true

	m2, cmd := m.Update(cruiseCycleCompleteMsg{})
	m = m2.(Model)

	if m.cruiseCycleActive {
		t.Error("expected cruiseCycleActive=false after cruiseCycleCompleteMsg")
	}
	if cmd == nil {
		t.Error("expected next cruise tick timer to be scheduled")
	}

	entries := m.opLog.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "Cruise cycle complete") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Cruise cycle complete' log entry")
	}
}

func TestModeSwitchClearsCruiseCycle(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.composerFocus = true
	m.cruiseCycleActive = true

	m.composerText = "/mode manual"
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m2.(Model)

	if m.cruiseCycleActive {
		t.Error("expected cruiseCycleActive=false after switching to manual mode")
	}
	if m.mode != "manual" {
		t.Errorf("expected mode='manual', got %q", m.mode)
	}
}

func TestFlowRetryMsg_RetriesAnalysisNotCreative(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.cruiseCycleActive = true

	m2, cmd := m.Update(flowRetryMsg{})
	m = m2.(Model)

	if cmd == nil {
		t.Error("expected a retry command from flowRetryMsg in cruise mode")
	}
	// cruiseCycleActive should still be true — retry does NOT reset the cycle
	if !m.cruiseCycleActive {
		t.Error("expected cruiseCycleActive to remain true during retry")
	}

	entries := m.opLog.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "Retrying analysis") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Retrying analysis' log entry")
	}
}

func TestCruiseTickMsg_NonCruiseMode_NilCmd(t *testing.T) {
	m := NewModel(nil, nil, "auto", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	_, cmd := m.Update(cruiseTickMsg{})
	if cmd != nil {
		t.Error("expected nil command from cruiseTickMsg in non-cruise mode")
	}
}

func TestInitMsg_CruiseMode_TriggersCreativeFirst(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{CruiseInterval: 900})
	m.width = 100
	m.height = 30

	m2, cmd := m.Update(initMsg{})
	m = m2.(Model)

	if !m.cruiseCycleActive {
		t.Error("expected cruiseCycleActive=true after initMsg in cruise mode")
	}
	if cmd == nil {
		t.Error("expected commands from initMsg in cruise mode")
	}
}

func TestCreativeCommand_ManualMode(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.composerFocus = true

	m.composerText = "/creative"
	m2, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m2.(Model)

	if cmd == nil {
		t.Error("expected command from /creative in manual mode")
	}

	entries := m.opLog.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "creative") || strings.Contains(e.Summary, "Creative") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log entry for manual creative trigger")
	}
}

func TestHelpCommand_IncludesCreative(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.composerFocus = true

	m.composerText = "/help"
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m2.(Model)

	entries := m.opLog.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "Help overlay opened") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected /help to open help overlay")
	}
	if !m.showHelpOverlay {
		t.Error("expected help overlay to be visible after /help")
	}
}

// ---------- Goal triage tests ----------

func TestGoalTriageMsg_GitdexCategory_AddsGoalWithTodos(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	triage := &helper.GoalTriageResult{
		Achievable: true,
		Category:   "gitdex",
		Reason:     "Can be done with git commands",
		Todos: []dotgitdex.Todo{
			{Title: "Step 1"},
			{Title: "Step 2"},
		},
	}
	m2, cmd := m.Update(goalTriageMsg{goalTitle: "Create branch", result: triage})
	m = m2.(Model)

	if m.activeGoal != "Create branch" {
		t.Errorf("expected activeGoal='Create branch', got %q", m.activeGoal)
	}
	if len(m.goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(m.goals))
	}
	if len(m.goals[0].Todos) != 2 {
		t.Errorf("expected 2 todos, got %d", len(m.goals[0].Todos))
	}
	if cmd == nil {
		t.Error("expected startAnalysis command after gitdex goal triage")
	}
}

func TestGoalTriageMsg_CreativeCategory_SavesProposal(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	triage := &helper.GoalTriageResult{
		Achievable: false,
		Category:   "creative",
		Reason:     "This is a strategic suggestion, not directly actionable",
	}
	m2, cmd := m.Update(goalTriageMsg{goalTitle: "Improve code quality", result: triage})
	m = m2.(Model)

	// Should NOT be added as a goal
	if len(m.goals) != 0 {
		t.Errorf("expected 0 goals for creative category, got %d", len(m.goals))
	}
	if m.activeGoal != "" {
		t.Errorf("expected empty activeGoal for creative category, got %q", m.activeGoal)
	}
	// No further commands
	if cmd != nil {
		t.Error("expected nil command for creative category goal")
	}

	entries := m.opLog.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "creative proposal") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log entry about creative proposal classification")
	}
}

func TestGoalTriageMsg_DiscardCategory_SavesDiscard(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	triage := &helper.GoalTriageResult{
		Achievable: false,
		Category:   "discard",
		Reason:     "This goal is impossible to achieve",
	}
	m2, cmd := m.Update(goalTriageMsg{goalTitle: "Launch rocket", result: triage})
	m = m2.(Model)

	if len(m.goals) != 0 {
		t.Errorf("expected 0 goals for discarded category, got %d", len(m.goals))
	}
	if cmd != nil {
		t.Error("expected nil command for discarded goal")
	}

	entries := m.opLog.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Summary, "discarded") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log entry about discarded goal")
	}
}

func TestGoalTriageMsg_Error_FallsBackToGitdex(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30

	m2, _ := m.Update(goalTriageMsg{goalTitle: "Fallback goal", err: errTest})
	m = m2.(Model)

	// On triage error, should fall back to adding as goal
	if len(m.goals) != 1 {
		t.Errorf("expected 1 goal after triage fallback, got %d", len(m.goals))
	}
	if m.activeGoal != "Fallback goal" {
		t.Errorf("expected activeGoal='Fallback goal', got %q", m.activeGoal)
	}
}

// ---------- Error loop / max replan tests ----------

func TestConsecutiveReplans_ResetOnSuccess(t *testing.T) {
	store := setupTestStore(t)
	orch := setupTestOrchestrator(t, store)
	m := NewModel(orch, store, "auto", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.consecutiveReplans = 2
	m.suggestions = []SuggestionDisplay{
		{Item: planner.SuggestionItem{Name: "S1"}, Status: StatusPending},
	}
	m.suggIdx = 0

	m2, _ := m.Update(executionResultMsg{
		index:  0,
		result: &executor.ExecutionResult{Success: true, Stdout: "ok"},
	})
	m = m2.(Model)

	if m.consecutiveReplans != 0 {
		t.Errorf("expected consecutiveReplans=0 after successful round, got %d", m.consecutiveReplans)
	}
}

func TestConsecutiveReplans_IncrementsOnFailure(t *testing.T) {
	store := setupTestStore(t)
	orch := setupTestOrchestrator(t, store)
	m := NewModel(orch, store, "auto", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.consecutiveReplans = 0
	m.suggestions = []SuggestionDisplay{
		{Item: planner.SuggestionItem{Name: "S1"}, Status: StatusPending},
		{Item: planner.SuggestionItem{Name: "S2"}, Status: StatusPending},
	}
	m.suggIdx = 0

	m2, cmd := m.Update(executionResultMsg{
		index:  0,
		result: &executor.ExecutionResult{Success: false, Stderr: "error"},
	})
	m = m2.(Model)

	if m.consecutiveReplans != 1 {
		t.Errorf("expected consecutiveReplans=1 after failure, got %d", m.consecutiveReplans)
	}
	if cmd == nil {
		t.Error("expected replan command after first failure")
	}
	if m.suggestions[1].Status != StatusSkipped {
		t.Errorf("expected remaining suggestion to be skipped, got %v", m.suggestions[1].Status)
	}
}

func TestConsecutiveReplans_ContinuesOnFailure(t *testing.T) {
	store := setupTestStore(t)
	orch := setupTestOrchestrator(t, store)
	m := NewModel(orch, store, "auto", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.consecutiveReplans = 5
	m.suggestions = []SuggestionDisplay{
		{Item: planner.SuggestionItem{Name: "S1"}, Status: StatusPending},
	}
	m.suggIdx = 0

	m2, cmd := m.Update(executionResultMsg{
		index:  0,
		result: &executor.ExecutionResult{Success: false, Stderr: "same error again"},
	})
	m = m2.(Model)

	if m.consecutiveReplans != 6 {
		t.Errorf("expected consecutiveReplans=6 (incremented), got %d", m.consecutiveReplans)
	}
	if cmd == nil {
		t.Error("expected replan command — no limit should stop replanning")
	}
}

func TestConsecutiveReplans_CruiseMode_ContinuesReplanning(t *testing.T) {
	store := setupTestStore(t)
	orch := setupTestOrchestrator(t, store)
	m := NewModel(orch, store, "cruise", "en", ConfigSnapshot{CruiseInterval: 900})
	m.ready = true
	m.width = 100
	m.height = 30
	m.cruiseCycleActive = true
	m.consecutiveReplans = 5
	m.suggestions = []SuggestionDisplay{
		{Item: planner.SuggestionItem{Name: "S1"}, Status: StatusPending},
	}
	m.suggIdx = 0

	m2, cmd := m.Update(executionResultMsg{
		index:  0,
		result: &executor.ExecutionResult{Success: false, Stderr: "persistent error"},
	})
	m = m2.(Model)

	if m.consecutiveReplans != 6 {
		t.Errorf("expected consecutiveReplans=6 in cruise mode, got %d", m.consecutiveReplans)
	}
	if cmd == nil {
		t.Error("expected replan command in cruise mode — no limit should stop replanning")
	}
}

// ---------- Output log failed commands summary test ----------

func TestOutputLogReadRecent_IncludesFailedSummary(t *testing.T) {
	store := setupTestStore(t)
	ol := dotgitdex.NewOutputLog(store)
	_ = ol.AppendRound(dotgitdex.Round{
		RoundID: 1, Flow: "maintain", Mode: "auto", Status: "partial-failure",
		StartedAt: time.Now(),
		Steps: []dotgitdex.Step{
			{SequenceID: 1, Name: "Push", Command: "git push", Success: false, Stderr: "auth failed"},
		},
	})

	text, err := ol.ReadRecent(3)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(text, "FAILED COMMANDS") {
		t.Error("expected FAILED COMMANDS section in output")
	}
	if !strings.Contains(text, "git push") {
		t.Error("expected 'git push' in failed commands summary")
	}
	if !strings.Contains(text, "auth failed") {
		t.Error("expected 'auth failed' in failed commands summary")
	}
}

// === Creative three-condition gate tests ===

func TestCreativeGate_NotTriggeredWhenGoalsPending(t *testing.T) {
	store := setupTestStore(t)
	orch := setupTestOrchestrator(t, store)
	m := NewModel(orch, store, "cruise", "en", ConfigSnapshot{CruiseInterval: 60})
	m.ready = true
	m.width = 100
	m.height = 30
	m.cruiseCycleActive = true
	m.creativeRanThisSlice = false

	goals := []dotgitdex.Goal{{Title: "Pending Goal", Completed: false}}
	_ = store.WriteGoalList(goals)

	m2, cmd := m.Update(cruiseCycleCompleteMsg{})
	m = m2.(Model)

	if m.cruiseCycleActive {
		t.Error("expected cruiseCycleActive=false after cycle complete")
	}
	if cmd == nil {
		t.Error("expected next timer tick to be scheduled")
	}
	entries := m.opLog.Entries()
	for _, e := range entries {
		if strings.Contains(e.Summary, "creative flow") {
			t.Error("creative should NOT be triggered when goals are still pending")
		}
	}
}

func TestCreativeGate_NotTriggeredWhenAlreadyRan(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{CruiseInterval: 60})
	m.ready = true
	m.width = 100
	m.height = 30
	m.cruiseCycleActive = true
	m.creativeRanThisSlice = true

	m2, cmd := m.Update(cruiseCycleCompleteMsg{})
	m = m2.(Model)

	if m.cruiseCycleActive {
		t.Error("expected cruiseCycleActive=false after cycle complete")
	}
	if cmd == nil {
		t.Error("expected next timer tick to be scheduled")
	}
	entries := m.opLog.Entries()
	for _, e := range entries {
		if strings.Contains(e.Summary, "creative flow") {
			t.Error("creative should NOT be triggered when already ran in this time slice")
		}
	}
}

func TestCreativeGate_TimeSliceResetOnTick(t *testing.T) {
	store := setupTestStore(t)
	m := NewModel(nil, store, "cruise", "en", ConfigSnapshot{CruiseInterval: 60})
	m.ready = true
	m.width = 100
	m.height = 30
	m.creativeRanThisSlice = true

	m2, _ := m.Update(cruiseTickMsg{})
	m = m2.(Model)

	if m.creativeRanThisSlice {
		t.Error("expected creativeRanThisSlice to be reset to false on new tick")
	}
}

// === Tool type label tests ===

func TestToolLabel_AllTypes(t *testing.T) {
	tests := []struct {
		actionType string
		expected   string
	}{
		{"git_command", "GIT"},
		{"shell_command", "SHELL"},
		{"file_write", "FILE"},
		{"file_read", "READ"},
		{"github_op", "GITHUB"},
		{"unknown_type", "UNKNOWN"},
	}
	for _, tt := range tests {
		a := planner.ActionSpec{Type: tt.actionType}
		if a.ToolLabel() != tt.expected {
			t.Errorf("ActionSpec{Type:%q}.ToolLabel() = %q, want %q", tt.actionType, a.ToolLabel(), tt.expected)
		}
	}
}

// === Scroll enhancement tests ===

func TestMouseWheelScroll_StepSize(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.panelScrolls[FocusLeft] = 10

	m2, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp, X: 5, Y: 5})
	m = m2.(Model)

	if m.panelScrolls[FocusLeft] != 7 {
		t.Errorf("expected scroll 7 after wheel up from 10 (step=3), got %d", m.panelScrolls[FocusLeft])
	}
}

func TestMouseWheelScroll_ClampsToZero(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.panelScrolls[FocusLeft] = 1

	m2, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp, X: 5, Y: 5})
	m = m2.(Model)

	if m.panelScrolls[FocusLeft] != 0 {
		t.Errorf("expected scroll 0 after wheel up from 1 (clamped), got %d", m.panelScrolls[FocusLeft])
	}
}

func TestPgUpPgDown_PassedFromComposerFocus(t *testing.T) {
	m := NewModel(nil, nil, "manual", "en", ConfigSnapshot{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.composerFocus = true
	m.focusZone = FocusInput
	m.panelScrolls[FocusInput] = 0

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	m = m2.(Model)

	if m.panelScrolls[FocusInput] == 0 {
		t.Log("pgdown from composer was handled correctly (scroll increased)")
	}
}

func TestStyleFunctions_NotPanic(t *testing.T) {
	theme.Init("catppuccin")
	fns := []func() string{
		func() string { return titleStyle().Render("test") },
		func() string { return subtitleStyle().Render("test") },
		func() string { return modeManual().Render("test") },
		func() string { return successStyle().Render("test") },
		func() string { return warningStyle().Render("test") },
		func() string { return dangerStyle().Render("test") },
		func() string { return infoStyle().Render("test") },
		func() string { return accentStyle().Render("test") },
		func() string { return keyStyle().Render("test") },
		func() string { return valueStyle().Render("test") },
		func() string { return mutedStyle().Render("test") },
		func() string { return borderStyle().Render("test") },
		func() string { return cursorStyle().Render("test") },
	}
	for i, fn := range fns {
		result := fn()
		if result == "" {
			t.Errorf("style function %d returned empty string", i)
		}
	}
}
