package tui

import (
	"os"
	"strings"
	"testing"

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

	out := m.renderOperationLogPanel(60)
	assert.Contains(t, out, "Operation Log (expanded)")
	assert.True(t, strings.Contains(out, "pgup/pgdn"))
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

	out := m.renderObservabilityPanel(72)
	assert.Contains(t, out, "Timeline")
	assert.Contains(t, out, "Command succeeded: git status --short")
	assert.Contains(t, out, "Commit changes and push upstream")
}
