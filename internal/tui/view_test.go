package tui

import (
	"os"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	_ = i18n.Init("en")
	os.Exit(m.Run())
}

func TestAnalysisHasFailure(t *testing.T) {
	assert.True(t, analysisHasFailure("AI error: timeout"))
	assert.True(t, analysisHasFailure("AI returned output that could not be parsed into the required JSON suggestion format"))
	assert.False(t, analysisHasFailure("Repository is clean and synchronized"))
}

func TestRepositoryLooksClean(t *testing.T) {
	assert.True(t, repositoryLooksClean(&status.GitState{}))

	assert.False(t, repositoryLooksClean(&status.GitState{
		WorkingTree: []git.FileStatus{{Path: "a.txt", WorktreeCode: git.StatusUntracked}},
	}))

	assert.False(t, repositoryLooksClean(&status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       "<url>",
			PushURLValid:  false,
			FetchURL:      "<url>",
			FetchURLValid: false,
		}},
	}))

	assert.False(t, repositoryLooksClean(&status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:                "origin",
			PushURL:             "https://example.com/repo.git",
			PushURLValid:        true,
			FetchURL:            "https://example.com/repo.git",
			FetchURLValid:       true,
			ReachabilityChecked: true,
			Reachable:           false,
		}},
	}))

	assert.True(t, repositoryLooksClean(&status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:                "origin",
			PushURL:             "git@github.com:user/repo.git",
			PushURLValid:        true,
			FetchURL:            "git@github.com:user/repo.git",
			FetchURLValid:       true,
			ReachabilityChecked: false,
			Reachable:           false,
		}},
	}))
}

func TestRenderMainContent_NarrowFallbackShowsCompactSummary(t *testing.T) {
	m := NewModel()
	m.width = 70
	m.height = 20
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "master"},
		WorkingTree: []git.FileStatus{{Path: "a.txt", WorktreeCode: git.StatusUntracked}},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURL:      "git@github.com:user/repo.git",
			PushURL:       "git@github.com:user/repo.git",
			FetchURLValid: true,
			PushURLValid:  true,
		}},
	}
	m.llmAnalysis = "AI says there are untracked files."
	m.opLog = oplog.New(10)
	m.opLog.Add(oplog.Entry{Summary: "State refreshed"})

	out := m.renderMainContent(16)
	assert.Contains(t, out, "[areas]")
	assert.Contains(t, out, "Operation Log")
	assert.NotContains(t, out, "Git Areas")
}

func TestRenderMainContent_WideLayoutShowsTreePanel(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.height = 30
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "master"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURL:      "git@github.com:user/repo.git",
			PushURL:       "git@github.com:user/repo.git",
			FetchURLValid: true,
			PushURLValid:  true,
		}},
	}
	m.llmAnalysis = "Repository synchronized."
	m.opLog = oplog.New(10)
	m.opLog.Add(oplog.Entry{Summary: "LLM analysis started"})

	out := m.renderMainContent(20)
	assert.Contains(t, out, "Git Areas")
	assert.Contains(t, out, "Operation Log")
}

func TestRenderOperationLogPanel_ExpandedShowsScrollHint(t *testing.T) {
	m := NewModel()
	m.logExpanded = true
	m.logScrollOffset = 2
	m.opLog = oplog.New(10)
	for i := 0; i < 5; i++ {
		m.opLog.Add(oplog.Entry{Summary: "entry"})
	}

	out := m.renderOperationLogPanel(60, 14)
	assert.Contains(t, out, "Operation Log (expanded)")
	assert.True(t, strings.Contains(out, "pgup/pgdn"))
	assert.Equal(t, 14, lipgloss.Height(out))
}

func TestRenderObservabilityPanel_TimelineShowsDetails(t *testing.T) {
	m := NewModel()
	m.obsTab = observabilityTimeline
	m.opLog = oplog.New(10)
	m.opLog.Add(oplog.Entry{
		Type:    oplog.EntryCmdSuccess,
		Summary: "Command succeeded: git status --short",
		Detail:  "M README.md",
	})
	m.opLog.Add(oplog.Entry{
		Type:    oplog.EntryLLMOutput,
		Summary: "LLM output: 2 suggestion(s)",
		Detail:  "Commit changes and push upstream",
	})

	out := m.renderObservabilityPanel(72, 18)
	assert.Contains(t, out, "Timeline")
	assert.Contains(t, out, "Command succeeded: git status --short")
	assert.Contains(t, out, "Commit changes and push upstream")
	assert.Equal(t, 18, lipgloss.Height(out))
}

func TestRenderObservabilityPanel_NarrowWidthKeepsSingleTabRow(t *testing.T) {
	m := NewModel()
	m.obsTab = observabilityThinking
	out := m.renderObservabilityPanel(36, 16)

	assert.Equal(t, 16, lipgloss.Height(out))
	assert.Contains(t, out, "<")
	assert.Contains(t, out, "Thinking")
}

func TestScrollPaneBy_LogDownMovesTowardNewerEntries(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.height = 36
	m.logExpanded = true
	m.logScrollOffset = 5
	m.opLog = oplog.New(100)
	for i := 0; i < 30; i++ {
		m.opLog.Add(oplog.Entry{Summary: "entry"})
	}

	updated := m.scrollPaneBy(scrollPaneLog, 1)
	assert.Equal(t, 4, updated.logScrollOffset)

	updated = updated.scrollPaneBy(scrollPaneLog, -1)
	assert.Equal(t, 5, updated.logScrollOffset)
}

func TestScrollPaneBy_ObservabilityDownMovesForward(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.height = 36
	m.obsTab = observabilityTimeline
	m.opLog = oplog.New(100)
	for i := 0; i < 30; i++ {
		m.opLog.Add(oplog.Entry{Summary: "timeline", Detail: "detail"})
	}

	updated := m.scrollPaneBy(scrollPaneObservability, 3)
	assert.Equal(t, 3, updated.obsScroll)

	updated = updated.scrollPaneBy(scrollPaneObservability, -2)
	assert.Equal(t, 1, updated.obsScroll)
}

func TestRenderCommandInspectorShowsStructuredFileResult(t *testing.T) {
	m := NewModel()
	m.lastCommand = commandTrace{
		Title:         "README.md",
		Status:        "file success",
		ResultKind:    resultKindFileWrite,
		FilePath:      "README.md",
		FileOperation: "update",
		BeforeContent: "old",
		AfterContent:  "new",
	}

	out := m.renderCommandInspector(90)
	assert.Contains(t, out, "File mutation")
	assert.Contains(t, out, "Before")
	assert.Contains(t, out, "After")
}

func TestRenderCommandInspectorShowsPreparedPlatformRequest(t *testing.T) {
	m := NewModel()
	m.gitState = &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURL:      "git@github.com:user/repo.git",
			PushURL:       "git@github.com:user/repo.git",
			FetchURLValid: true,
			PushURLValid:  true,
		}},
	}
	m.lastPlatformOp = &git.PlatformExecInfo{
		CapabilityID: "pages",
		Flow:         "inspect",
		Query:        map[string]string{"view": "latest_build"},
	}

	out := m.renderCommandInspector(80)
	assert.Contains(t, out, "Prepared platform request")
	assert.Contains(t, out, "latest_build")
	assert.Contains(t, out, "Coverage:")
	assert.Contains(t, out, "partial_mutate")
	assert.Contains(t, out, "Press e to edit and retry")
}

func TestRenderWorkflowSelectScreenShowsCoverageSummary(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.height = 40
	m.screen = screenWorkflowSelect
	m.gitState = &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURL:      "git@github.com:user/repo.git",
			PushURL:       "git@github.com:user/repo.git",
			FetchURLValid: true,
			PushURLValid:  true,
		}},
	}
	m.workflows = []workflowDefinition{{
		ID:           "submit_pr",
		Label:        "Submit PR",
		Goal:         "Create and merge a pull request",
		Capabilities: []string{"pull_request", "pr_review"},
	}}
	out := m.renderWorkflowSelectScreen()
	assert.Contains(t, out, "coverage:")
	assert.Contains(t, out, "partial_mutate=1")
	assert.Contains(t, out, "full=1")
}

func TestRenderActionBarIncludesSlashEditPlatformHint(t *testing.T) {
	m := NewModel()
	m.width = 220
	m.lastPlatformOp = &git.PlatformExecInfo{
		CapabilityID: "pages",
		Flow:         "inspect",
	}

	out := m.renderActionBar()
	assert.Contains(t, out, "/edit")
}

func TestRenderActionBarIncludesSlashEditFileHint(t *testing.T) {
	m := NewModel()
	m.width = 220
	m.lastCommand = commandTrace{
		ResultKind:    resultKindFileWrite,
		FilePath:      "README.md",
		FileOperation: "update",
		AfterContent:  "content",
	}

	out := m.renderActionBar()
	assert.Contains(t, out, "/edit")
}

func TestRenderSuggestionCardsKeepsPlatformMetadataCollapsedByDefault(t *testing.T) {
	m := NewModel()
	m.width = 140
	m.suggestions = []git.Suggestion{{
		Action:      "Publish release",
		Reason:      "Finalize release assets and publish the draft",
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "release",
			Flow:         "mutate",
			Operation:    "publish_draft",
			ResourceID:   "v1.0.0",
		},
	}}
	m.suggExecState = []git.ExecState{git.ExecPending}
	out := m.renderSuggestionCards(120)
	assert.Contains(t, out, "Change release via publish draft")
	assert.NotContains(t, out, "coverage:")
	assert.NotContains(t, out, "rollback:")
	assert.NotContains(t, out, "approval:")
}
