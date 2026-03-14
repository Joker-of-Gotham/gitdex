package tui

import (
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func BenchmarkRenderMainLayoutHeavy(b *testing.B) {
	m := newHeavyRenderModel()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = m.renderMainLayoutWithRegions(36)
	}
}

func BenchmarkRenderSuggestionCardsCompactHeavy(b *testing.B) {
	m := newHeavyRenderModel()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.renderSuggestionCardsCompactWithRegions(92)
	}
}

func BenchmarkUpdateMainPromptTypingHeavy(b *testing.B) {
	m := newHeavyRenderModel()
	m.screen = screenMain
	m.composerFocused = true
	m.composerInput = "/config "
	m.composerCursor = len(m.composerInput)
	key := tea.KeyPressMsg(tea.Key{Text: "s"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model, _ := m.updateMain(key)
		m = model.(Model)
		m.composerInput = "/config "
		m.composerCursor = len(m.composerInput)
	}
}

func TestRenderMainLayoutPerformanceBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping perf budget in short mode")
	}
	m := newHeavyRenderModel()
	const iterations = 120
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, _, _ = m.renderMainLayoutWithRegions(36)
	}
	elapsed := time.Since(start)
	perRender := elapsed / iterations
	t.Logf("render_main_layout total=%s per_render=%s", elapsed, perRender)
	if perRender > 30*time.Millisecond {
		t.Fatalf("render_main_layout too slow: %s per render", perRender)
	}
}

func TestPromptTypingPerformanceBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping perf budget in short mode")
	}
	m := newHeavyRenderModel()
	m.screen = screenMain
	m.composerFocused = true
	m.composerInput = "/config "
	m.composerCursor = len(m.composerInput)
	const iterations = 240
	start := time.Now()
	for i := 0; i < iterations; i++ {
		model, _ := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "s"}))
		m = model.(Model)
		m.composerInput = "/config "
		m.composerCursor = len(m.composerInput)
	}
	elapsed := time.Since(start)
	perKey := elapsed / iterations
	t.Logf("prompt_typing total=%s per_key=%s", elapsed, perKey)
	if perKey > 8*time.Millisecond {
		t.Fatalf("prompt typing too slow: %s per key", perKey)
	}
}

func TestUIClickPerformanceBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping perf budget in short mode")
	}
	m := newHeavyRenderModel()
	const iterations = 240
	start := time.Now()
	for i := 0; i < iterations; i++ {
		model, _ := m.handleUIClick(uiClickMsg{action: "observability_tab", index: i % len(observabilityTabs())})
		m = model.(Model)
	}
	elapsed := time.Since(start)
	perClick := elapsed / iterations
	t.Logf("ui_click total=%s per_click=%s", elapsed, perClick)
	if perClick > 5*time.Millisecond {
		t.Fatalf("ui click too slow: %s per click", perClick)
	}
}

func newHeavyRenderModel() Model {
	m := NewModel()
	m.ready = true
	m.screen = screenMain
	m.width = 160
	m.height = 48
	m.composerFocused = true
	m.composerInput = "/config "
	m.composerCursor = len(m.composerInput)
	m.session.ActiveGoal = "Review repository readiness and next actions"
	m.llmGoalStatus = "in_progress"
	m.llmAnalysis = "Review repository readiness, surface missing configuration, and avoid speculative platform actions unless the repository provides clear evidence."
	m.commandResponseTitle = "Configuration"
	m.commandResponseBody = "Platform access:\n- GitHub: missing token | gh: CLI unavailable | browser: disabled\nQuick actions:\n/settings\n/config status\n/provider"
	m.lastCommand = commandTrace{
		Title:              "pages / inspect",
		Status:             "platform unavailable",
		Output:             "GitHub Pages unavailable: missing API token, gh CLI not found, browser adapter disabled",
		ResultKind:         resultKindPlatformAdmin,
		PlatformCapability: "pages",
		PlatformFlow:       "inspect",
	}
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main", Upstream: "origin/main", Ahead: 2},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURL:      "https://github.com/example/repo.git",
			PushURL:       "https://github.com/example/repo.git",
			FetchURLValid: true,
			PushURLValid:  true,
		}},
		LocalBranches:  []string{"main", "feature/test"},
		RemoteBranches: []string{"origin/main"},
	}
	for i := 0; i < 16; i++ {
		m.gitState.WorkingTree = append(m.gitState.WorkingTree, git.FileStatus{
			Path:         fmt.Sprintf("internal/file_%02d.go", i),
			WorktreeCode: git.StatusModified,
		})
	}
	for i := 0; i < 8; i++ {
		m.gitState.StagingArea = append(m.gitState.StagingArea, git.FileStatus{
			Path:        fmt.Sprintf("cmd/staged_%02d.go", i),
			StagingCode: git.StatusModified,
		})
	}
	for i := 0; i < 12; i++ {
		m.suggestions = append(m.suggestions, git.Suggestion{
			Action:      fmt.Sprintf("Inspect repository readiness %02d", i),
			Reason:      "Inspect current repository and platform readiness before proposing writes.",
			Interaction: git.PlatformExec,
			PlatformOp: &git.PlatformExecInfo{
				CapabilityID: "pages",
				Flow:         "inspect",
				Query:        map[string]string{"view": "latest_build"},
			},
			RiskLevel: git.RiskCaution,
		})
		m.suggExecState = append(m.suggExecState, git.ExecPending)
		m.suggExecMsg = append(m.suggExecMsg, "Platform access needs configuration. Run /config status.")
	}
	m.opLog = oplog.New(200)
	for i := 0; i < 80; i++ {
		m.opLog.Add(oplog.Entry{
			Summary: fmt.Sprintf("entry %03d", i),
			Detail:  "render and interaction performance sample",
		})
	}
	return m
}
