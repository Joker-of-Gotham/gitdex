package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/app/autonomyexec"
	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/llm/adapter"
	"github.com/your-org/gitdex/internal/platform/config"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/storage"
	"github.com/your-org/gitdex/internal/tui/components"
	"github.com/your-org/gitdex/internal/tui/layout"
	"github.com/your-org/gitdex/internal/tui/panes"
	"github.com/your-org/gitdex/internal/tui/views"
)

func TestNew(t *testing.T) {
	m := New()
	if m.focus != FocusComposer {
		t.Errorf("expected initial focus on Composer, got %d", m.focus)
	}
}

func TestModel_Init(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	m := New()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if !model.ready {
		t.Error("expected ready after WindowSizeMsg")
	}
	if model.dims.Breakpoint != layout.Standard {
		t.Errorf("expected Standard breakpoint for 120 width, got %v", model.dims.Breakpoint)
	}
}

func TestModel_Update_Quit_CtrlC(t *testing.T) {
	m := New()
	m.ready = true
	msg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("expected quit command from Ctrl+C")
	}
}

func TestModel_Update_Quit_NotInContent(t *testing.T) {
	m := New()
	m.ready = true
	m.focus = FocusComposer
	msg := tea.KeyPressMsg{Code: 'q'}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("expected quit command from 'q' when not in content")
	}
}

func TestModel_Update_QBlocked_InContent(t *testing.T) {
	m := New()
	m.ready = true
	m.setFocusArea(FocusContent)
	msg := tea.KeyPressMsg{Code: 'q'}
	_, cmd := m.Update(msg)
	if cmd != nil {
		t.Error("'q' should not quit when focus is on content")
	}
}

func TestModel_Update_ToggleHelp(t *testing.T) {
	m := New()
	m.ready = true
	msg := tea.KeyPressMsg{Code: '?'}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if !model.showHelp {
		t.Error("expected help to be shown")
	}
}

func TestModel_Update_ToggleFocus(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(160, 40)
	m.resizeAll()

	initial := m.focus
	msg := tea.KeyPressMsg{Code: '\t'}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.focus == initial {
		t.Error("focus should have toggled")
	}
}

func TestModel_Update_TabStaysInSettingsContent(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(160, 40)
	m.resizeAll()
	m.switchView(views.ViewSettings)
	m.setFocusArea(FocusContent)

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	model := updated.(Model)
	if model.focus != FocusContent {
		t.Fatalf("focus after tab in settings = %d, want FocusContent", model.focus)
	}
	if model.settingsView.InspectorData().CurrentSection != "Model Runtime" {
		t.Fatalf("section after tab = %q, want Model Runtime", model.settingsView.InspectorData().CurrentSection)
	}
}

func TestModel_SetSummary(t *testing.T) {
	m := New()
	summary := &repo.RepoSummary{
		Owner:        "test-owner",
		Repo:         "test-repo",
		OverallLabel: repo.Healthy,
		Timestamp:    time.Now(),
		Local:        repo.LocalState{Label: repo.Healthy},
		Remote:       repo.RemoteState{Label: repo.Healthy},
	}
	m.SetSummary(summary)
}

func TestModel_View_NotReady(t *testing.T) {
	m := New()
	v := m.View()
	if v.Content == "" {
		t.Error("expected initializing message")
	}
}

func TestModel_View_Ready(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()
	v := m.View()
	if v.Content == "" {
		t.Error("expected app content")
	}
}

func TestModel_SwitchView(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	if m.router.ActiveID() != views.ViewDashboard {
		t.Error("initial view should be Dashboard")
	}

	m.switchView(views.ViewChat)
	if m.router.ActiveID() != views.ViewChat {
		t.Error("should have switched to Chat")
	}
}

func TestModel_SwitchAllViews(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	ids := []views.ID{
		views.ViewDashboard,
		views.ViewChat,
		views.ViewExplorer,
		views.ViewWorkspace,
		views.ViewSettings,
	}
	for _, id := range ids {
		m.switchView(id)
		if m.router.ActiveID() != id {
			t.Errorf("switchView(%s): got %q", id, m.router.ActiveID())
		}
	}
}

func TestModel_CycleTheme(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	origPalette := m.paletteName
	m.cycleTheme()
	if m.paletteName == origPalette {
		t.Error("cycleTheme should change palette name")
	}
	if m.theme == nil {
		t.Error("theme should not be nil after cycling")
	}
}

func TestModel_CycleTheme_WrapsAround(t *testing.T) {
	m := New()
	m.ready = true
	for i := 0; i < 10; i++ {
		m.cycleTheme()
	}
	if m.theme == nil {
		t.Error("theme should not be nil after multiple cycles")
	}
}

func TestModel_NavSelectMsg(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(160, 40)
	m.resizeAll()

	msg := panes.NavSelectMsg{Item: panes.NavItem{Label: "Explorer", Path: "explorer"}}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.router.ActiveID() != views.ViewExplorer {
		t.Error("NavSelectMsg should switch to Explorer view")
	}
}

func TestModel_ThemePointerShared(t *testing.T) {
	m := New()
	themeAddr := m.theme
	m.cycleTheme()
	if m.theme != themeAddr {
		t.Error("theme pointer should remain the same after cycling (shared pointer)")
	}
}

func TestNavPathToViewID(t *testing.T) {
	tests := map[string]views.ID{
		"dashboard": views.ViewDashboard,
		"chat":      views.ViewChat,
		"explorer":  views.ViewExplorer,
		"workspace": views.ViewWorkspace,
		"settings":  views.ViewSettings,
	}
	for path, want := range tests {
		if got := navPathToViewID(path); got != want {
			t.Errorf("navPathToViewID(%q) = %q, want %q", path, got, want)
		}
	}
	if navPathToViewID("unknown") != "" {
		t.Error("navPathToViewID(unknown) should return empty")
	}
}

func TestModel_Update_ConfigSaveMsg(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	msg := views.ConfigSaveMsg{
		Fields: []views.ConfigField{
			{Key: "llm.provider", Label: "LLM Provider", Value: "deepseek"},
			{Key: "llm.api_key", Label: "API Key", Value: "sk-secret", Secret: true},
		},
	}
	m.Update(msg)
}

func TestModel_Update_FileContentMsg(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	msg := views.FileContentMsg{Path: "main.go", Content: "package main\nfunc main() {}"}
	m.Update(msg)
}

func TestModel_Update_FileDiffMsg(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	msg := views.FileDiffMsg{Diff: "--- a/main.go\n+++ b/main.go\n@@ -1,1 +1,1 @@\n-old\n+new"}
	m.Update(msg)
}

func TestModel_Update_RequestFileContentMsg(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	msg := views.RequestFileContentMsg{Path: "nonexistent_file.go"}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("RequestFileContentMsg should return a loading command")
	}
}

func TestModel_Update_RequestFileDiffMsg(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	msg := views.RequestFileDiffMsg{Path: "main.go"}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("RequestFileDiffMsg should return a loading command")
	}
}

func TestModel_Update_RequestPRAndIssueDetailMsgs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/7", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"number": 7,
			"title":  "Improve sync",
			"state":  "open",
			"user":   map[string]any{"login": "alice"},
			"body":   "Detailed body",
			"labels": []map[string]any{{"name": "infra"}},
		})
	})
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/7/reviews", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{"state": "APPROVED", "user": map[string]any{"login": "reviewer"}}})
	})
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/7/files", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{"filename": "main.go", "status": "modified", "additions": 3, "deletions": 1}})
	})
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/comments", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{"body": "looks good", "user": map[string]any{"login": "commenter"}}})
	})
	mux.HandleFunc("GET /api/v3/repos/owner/repo/issues/9", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"number":    9,
			"title":     "Track drift",
			"state":     "open",
			"user":      map[string]any{"login": "bob"},
			"body":      "Issue body",
			"labels":    []map[string]any{{"name": "bug"}},
			"assignees": []map[string]any{{"login": "ops"}},
			"milestone": map[string]any{"title": "M1"},
		})
	})
	mux.HandleFunc("GET /api/v3/repos/owner/repo/issues/9/comments", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{"body": "investigating", "user": map[string]any{"login": "alice"}}})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client, err := ghclient.NewClientWithBaseURL(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	m := New()
	m.ready = true
	m.dims = layout.Classify(140, 40)
	m.resizeAll()
	m.ghClient = client
	m.activeRepo = &repo.RepoContext{Owner: "owner", Name: "repo", FullName: "owner/repo"}

	_, prCmd := m.Update(views.RequestPRDetailMsg{Number: 7})
	if prCmd == nil {
		t.Fatal("RequestPRDetailMsg should return a loading command")
	}
	prMsg := prCmd()
	if _, ok := prMsg.(views.PRDetailMsg); !ok {
		t.Fatalf("pr detail cmd returned %T", prMsg)
	}

	_, issueCmd := m.Update(views.RequestIssueDetailMsg{Number: 9})
	if issueCmd == nil {
		t.Fatal("RequestIssueDetailMsg should return a loading command")
	}
	issueMsg := issueCmd()
	if _, ok := issueMsg.(views.IssueDetailMsg); !ok {
		t.Fatalf("issue detail cmd returned %T", issueMsg)
	}
}

func TestModel_Update_RequestPRActionMsg(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/issues/7/comments", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"html_url": "https://example.test/pr/7#issuecomment-1",
			"body":     "ship it",
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client, err := ghclient.NewClientWithBaseURL(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	m := New()
	m.ready = true
	m.dims = layout.Classify(140, 40)
	m.resizeAll()
	m.ghClient = client
	m.activeRepo = &repo.RepoContext{Owner: "owner", Name: "repo", FullName: "owner/repo"}

	_, cmd := m.Update(views.RequestPRActionMsg{Number: 7, Kind: views.PRActionComment, Body: "ship it"})
	if cmd == nil {
		t.Fatal("RequestPRActionMsg should return a command")
	}
	msg := cmd()
	result, ok := msg.(views.PRActionResultMsg)
	if !ok {
		t.Fatalf("PR action cmd returned %T", msg)
	}
	if result.Err != nil {
		t.Fatalf("PR action result err = %v", result.Err)
	}
	if !strings.Contains(result.Message, "Commented on PR #7") {
		t.Fatalf("unexpected PR action message: %q", result.Message)
	}
}

func TestModel_Update_RequestIssueActionMsg(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/repos/owner/repo/issues/9", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"number": 9,
			"labels": []map[string]any{{"name": "bug"}, {"name": "triage"}},
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client, err := ghclient.NewClientWithBaseURL(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	m := New()
	m.ready = true
	m.dims = layout.Classify(140, 40)
	m.resizeAll()
	m.ghClient = client
	m.activeRepo = &repo.RepoContext{Owner: "owner", Name: "repo", FullName: "owner/repo"}

	_, cmd := m.Update(views.RequestIssueActionMsg{Number: 9, Kind: views.IssueActionLabel, Values: []string{"bug", "triage"}})
	if cmd == nil {
		t.Fatal("RequestIssueActionMsg should return a command")
	}
	msg := cmd()
	result, ok := msg.(views.IssueActionResultMsg)
	if !ok {
		t.Fatalf("Issue action cmd returned %T", msg)
	}
	if result.Err != nil {
		t.Fatalf("Issue action result err = %v", result.Err)
	}
	if !strings.Contains(result.Message, "Updated labels on issue #9") {
		t.Fatalf("unexpected issue action message: %q", result.Message)
	}
}

func TestModel_Update_RequestWorkflowDispatchMsg(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/actions/workflows/123/dispatches", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client, err := ghclient.NewClientWithBaseURL(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	m := New()
	m.ready = true
	m.dims = layout.Classify(140, 40)
	m.resizeAll()
	m.ghClient = client
	m.activeRepo = &repo.RepoContext{Owner: "owner", Name: "repo", FullName: "owner/repo", DefaultBranch: "main"}

	_, cmd := m.Update(views.RequestWorkflowDispatchMsg{WorkflowID: 123, Ref: "main"})
	if cmd == nil {
		t.Fatal("RequestWorkflowDispatchMsg should return a command")
	}
	msg := cmd()
	result, ok := msg.(views.WorkflowDispatchResultMsg)
	if !ok {
		t.Fatalf("workflow dispatch cmd returned %T", msg)
	}
	if result.Err != nil {
		t.Fatalf("workflow dispatch result err = %v", result.Err)
	}
	if !strings.Contains(result.Message, "Dispatched workflow 123") {
		t.Fatalf("unexpected workflow dispatch message: %q", result.Message)
	}
}

func TestModel_Update_RequestWorkflowActionMsg(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/actions/runs/321/rerun", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("POST /api/v3/repos/owner/repo/actions/runs/321/cancel", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "cancellation requested"})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client, err := ghclient.NewClientWithBaseURL(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	m := New()
	m.ready = true
	m.dims = layout.Classify(140, 40)
	m.resizeAll()
	m.ghClient = client
	m.activeRepo = &repo.RepoContext{Owner: "owner", Name: "repo", FullName: "owner/repo", DefaultBranch: "main"}

	_, rerunCmd := m.Update(views.RequestWorkflowActionMsg{RunID: 321, Kind: views.WorkflowActionRerun})
	if rerunCmd == nil {
		t.Fatal("RequestWorkflowActionMsg rerun should return a command")
	}
	rerunMsg := rerunCmd()
	rerunResult, ok := rerunMsg.(views.WorkflowActionResultMsg)
	if !ok {
		t.Fatalf("workflow action rerun cmd returned %T", rerunMsg)
	}
	if rerunResult.Err != nil {
		t.Fatalf("workflow rerun result err = %v", rerunResult.Err)
	}
	if !strings.Contains(rerunResult.Message, "Rerun requested for workflow run 321") {
		t.Fatalf("unexpected workflow rerun message: %q", rerunResult.Message)
	}

	_, cancelCmd := m.Update(views.RequestWorkflowActionMsg{RunID: 321, Kind: views.WorkflowActionCancel})
	if cancelCmd == nil {
		t.Fatal("RequestWorkflowActionMsg cancel should return a command")
	}
	cancelMsg := cancelCmd()
	cancelResult, ok := cancelMsg.(views.WorkflowActionResultMsg)
	if !ok {
		t.Fatalf("workflow action cancel cmd returned %T", cancelMsg)
	}
	if cancelResult.Err != nil {
		t.Fatalf("workflow cancel result err = %v", cancelResult.Err)
	}
	if !strings.Contains(cancelResult.Message, "Cancel requested for workflow run 321") {
		t.Fatalf("unexpected workflow cancel message: %q", cancelResult.Message)
	}
}

func TestModel_Update_RequestCommitDetailAndBranchCheckoutMsgs(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile go.mod: %v", err)
	}

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	run("init", "-b", "main")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")
	run("add", "go.mod")
	run("commit", "-m", "initial commit")

	m := New()
	m.ready = true
	m.dims = layout.Classify(140, 40)
	m.resizeAll()
	m.activeRepo = &repo.RepoContext{
		Name:       "repo",
		FullName:   "owner/repo",
		LocalPaths: []string{root},
		IsLocal:    true,
	}

	logMsg := m.loadCommitLog()()
	commitData, ok := logMsg.(views.CommitLogDataMsg)
	if !ok || len(commitData.Commits) == 0 {
		t.Fatalf("commit log msg = %#v", logMsg)
	}

	_, detailCmd := m.Update(views.RequestCommitDetailMsg{Hash: commitData.Commits[0].Hash})
	if detailCmd == nil {
		t.Fatal("RequestCommitDetailMsg should return a loading command")
	}
	if _, ok := detailCmd().(views.CommitDetailMsg); !ok {
		t.Fatalf("detail cmd returned %T", detailCmd())
	}

	run("checkout", "-b", "feature/demo")
	run("checkout", "main")

	_, checkoutCmd := m.Update(views.RequestBranchCheckoutMsg{Name: "feature/demo"})
	if checkoutCmd == nil {
		t.Fatal("RequestBranchCheckoutMsg should return a command")
	}
	msg := checkoutCmd()
	if _, ok := msg.(views.BranchCheckoutResultMsg); !ok {
		t.Fatalf("checkout cmd returned %T", msg)
	}
}

func TestModel_Update_RequestCommitActionMsg(t *testing.T) {
	root := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	run("init", "-b", "main")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")

	target := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(target, []byte("one\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	run("add", "notes.txt")
	run("commit", "-m", "initial")

	if err := os.WriteFile(target, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatalf("WriteFile second: %v", err)
	}
	run("add", "notes.txt")
	run("commit", "-m", "second")

	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v\n%s", err, string(out))
	}
	sha := strings.TrimSpace(string(out))

	m := New()
	m.ready = true
	m.dims = layout.Classify(140, 40)
	m.resizeAll()
	m.activeRepo = &repo.RepoContext{
		Name:       "repo",
		FullName:   "owner/repo",
		LocalPaths: []string{root},
		IsLocal:    true,
	}

	_, cmd := m.Update(views.RequestCommitActionMsg{Hash: sha, Kind: views.CommitActionRevert})
	if cmd == nil {
		t.Fatal("RequestCommitActionMsg should return a command")
	}
	msg := cmd()
	result, ok := msg.(views.CommitActionResultMsg)
	if !ok {
		t.Fatalf("commit action cmd returned %T", msg)
	}
	if result.Err != nil {
		t.Fatalf("commit action result err = %v", result.Err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	if normalized != "one\n" {
		t.Fatalf("unexpected file content after revert: %q", string(data))
	}
}

func TestModel_Update_RequestBranchActionMsg(t *testing.T) {
	root := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	run("init", "-b", "main")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	run("add", "README.md")
	run("commit", "-m", "initial")

	m := New()
	m.ready = true
	m.dims = layout.Classify(140, 40)
	m.resizeAll()
	m.activeRepo = &repo.RepoContext{
		Name:       "repo",
		FullName:   "owner/repo",
		LocalPaths: []string{root},
		IsLocal:    true,
	}

	_, cmd := m.Update(views.RequestBranchActionMsg{Kind: views.BranchActionCreate, Name: "feature/demo", Target: "main"})
	if cmd == nil {
		t.Fatal("RequestBranchActionMsg should return a command")
	}
	msg := cmd()
	result, ok := msg.(views.BranchActionResultMsg)
	if !ok {
		t.Fatalf("branch action cmd returned %T", msg)
	}
	if result.Err != nil {
		t.Fatalf("branch action result err = %v", result.Err)
	}

	out, err := exec.Command("git", "-C", root, "branch", "--list", "feature/demo").CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --list: %v\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "feature/demo") {
		t.Fatalf("branch not created, output: %q", string(out))
	}
}

func TestModel_Update_PullsDataMsg(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	msg := views.PullsDataMsg{Items: []repo.PullRequestSummary{
		{Number: 1, Title: "test PR", Author: "user"},
	}}
	m.Update(msg)
}

func TestModel_Update_IssuesDataMsg(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	msg := views.IssuesDataMsg{Items: []repo.IssueSummary{
		{Number: 10, Title: "test issue", Author: "dev", State: "open"},
	}}
	m.Update(msg)
}

func TestModel_Update_FileTreeDataMsg(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	root := views.BuildFileTree([]string{"cmd/main.go", "internal/app.go"})
	msg := views.FileTreeDataMsg{Root: root}
	m.Update(msg)
}

func TestModel_Update_CommitAndBranchMsgs(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	m.Update(views.CommitLogDataMsg{Commits: []views.CommitEntry{{Hash: "abc1234", Author: "alice", Date: "2026-03-21", Message: "init"}}})
	m.Update(views.BranchTreeDataMsg{Branches: []views.BranchEntry{{Name: "main", IsCurrent: true}}})
}

func TestModel_Update_RequestFileEditAndSaveMsg(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "main.go")
	if err := os.WriteFile(target, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()
	m.activeRepo = &repo.RepoContext{
		Name:       "repo",
		FullName:   "owner/repo",
		LocalPaths: []string{root},
		IsLocal:    true,
	}

	_, editCmd := m.Update(views.RequestFileEditMsg{Path: "main.go"})
	if editCmd == nil {
		t.Fatal("RequestFileEditMsg should return a loading command")
	}

	_, saveCmd := m.Update(views.RequestFileSaveMsg{Path: target, Content: "package main\n\nfunc main() {}\n"})
	if saveCmd == nil {
		t.Fatal("RequestFileSaveMsg should return a save command")
	}
	msg := saveCmd()
	saved, ok := msg.(views.FileSavedMsg)
	if !ok {
		t.Fatalf("save cmd returned %T", msg)
	}
	if saved.Err != nil {
		t.Fatalf("save err = %v", saved.Err)
	}
}

func TestModel_FocusModeUpdatesStatusBar(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	m.toggleFocus()
	if m.focus != FocusContent {
		t.Errorf("after toggle from Composer: focus = %d, want FocusContent", m.focus)
	}
}

func TestModel_FocusCycle_AllModes(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(160, 40)
	m.resizeAll()

	if m.focus != FocusComposer {
		t.Fatal("initial focus should be Composer")
	}

	// Inspector is visible by default now, so full cycle: Composer -> Content -> Inspector -> Composer
	m.toggleFocus()
	if m.focus != FocusContent {
		t.Errorf("step 1: want FocusContent, got %d", m.focus)
	}

	m.toggleFocus()
	if m.focus != FocusInspector {
		t.Errorf("step 2 (inspector visible by default): want FocusInspector, got %d", m.focus)
	}

	m.toggleFocus()
	if m.focus != FocusComposer {
		t.Errorf("step 3: want FocusComposer, got %d", m.focus)
	}

	// Now hide inspector and test shortened cycle
	if m.inspectorPane != nil {
		m.inspectorPane.Toggle()
	}

	m.toggleFocus()
	if m.focus != FocusContent {
		t.Errorf("step 4: want FocusContent, got %d", m.focus)
	}

	m.toggleFocus()
	if m.focus != FocusComposer {
		t.Errorf("step 5 (inspector hidden): want FocusComposer, got %d", m.focus)
	}
}

func TestModel_SetFocusArea_DirectCtrl(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(160, 40)
	m.resizeAll()

	m.setFocusArea(FocusContent)
	if m.focus != FocusContent {
		t.Error("direct setFocusArea to Content failed")
	}
	if m.composer.Focused() {
		t.Error("composer should not be focused when content is")
	}

	m.setFocusArea(FocusComposer)
	if !m.composer.Focused() {
		t.Error("composer should be focused after setFocusArea(Composer)")
	}
}

func TestModel_StatusDataMsg_RoutesToDashboard(t *testing.T) {
	m := New()
	m.ready = true
	m.dims = layout.Classify(120, 40)
	m.resizeAll()

	summary := &repo.RepoSummary{
		Owner:        "org",
		Repo:         "repo",
		OverallLabel: repo.Healthy,
	}
	m.Update(views.StatusDataMsg{Summary: summary})
}

func TestModel_LoadFileTree_ReturnsCmd(t *testing.T) {
	m := New()
	cmd := m.loadFileTree()
	if cmd == nil {
		t.Error("loadFileTree should return a non-nil command")
	}
}

func TestModel_LoadFileContent_ReturnsCmd(t *testing.T) {
	m := New()
	cmd := m.loadFileContent("nonexistent.go")
	if cmd == nil {
		t.Error("loadFileContent should return a non-nil command")
	}
}

func TestModel_LoadFileDiff_ReturnsCmd(t *testing.T) {
	m := New()
	cmd := m.loadFileDiff("nonexistent.go")
	if cmd == nil {
		t.Error("loadFileDiff should return a non-nil command")
	}
}

func TestModel_HandleSubmit_IntentExecutesAndRefreshes(t *testing.T) {
	root := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	run("init", "-b", "main")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile README: %v", err)
	}
	run("add", "README.md")
	run("commit", "-m", "seed")

	provider, err := storage.NewProvider(storage.Config{Type: storage.BackendMemory})
	if err != nil {
		t.Fatalf("new memory provider: %v", err)
	}
	defer func() { _ = provider.Close() }()

	restore := autonomyexec.SetProviderOverrideForTest(&adapter.MockProvider{
		ChatCompletionFn: func(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			return &adapter.ChatResponse{
				Content: `{
  "description": "add notes file",
  "steps": [
    {"order": 1, "action": "file.write", "args": {"path": "notes/tui.txt", "content": "hello from tui\n"}, "reversible": true, "description": "write file"},
    {"order": 2, "action": "git.add", "args": {"path": "notes/tui.txt"}, "reversible": true, "description": "stage file"},
    {"order": 3, "action": "git.commit", "args": {"message": "tui autonomy commit"}, "reversible": false, "description": "commit"}
  ],
  "risk_level": "high",
  "rationale": "test"
}`,
			}, nil
		},
	})
	defer restore()

	m := New()
	m.SetBootstrapApp(bootstrap.App{
		RepoRoot: root,
		Config: config.Config{
			FileConfig: config.FileConfig{
				Output:  "text",
				Storage: config.StorageConfig{Type: string(storage.BackendMemory)},
				LLM: config.LLMConfig{
					Provider: "openai",
					Model:    "gpt-4o-mini",
					APIKey:   "test-key",
				},
			},
			Paths: config.ConfigPaths{
				WorkingDir:         root,
				RepositoryRoot:     root,
				RepositoryDetected: true,
			},
		},
		StorageProvider: provider,
	})
	m.activeRepo = &repo.RepoContext{
		Owner:      "owner",
		Name:       "repo",
		FullName:   "owner/repo",
		LocalPaths: []string{root},
		IsLocal:    true,
	}
	m.ready = true
	m.dims = layout.Classify(140, 40)
	m.resizeAll()

	cmd := m.handleSubmit(components.SubmitMsg{Input: "!add a notes file", IsIntent: true})
	if cmd == nil {
		t.Fatal("intent submit should return a command")
	}

	msg := cmd()
	ar, ok := msg.(autonomyResultMsg)
	if !ok {
		t.Fatalf("intent cmd returned %T", msg)
	}
	if ar.err != nil {
		t.Fatalf("intent execution error: %v", ar.err)
	}

	updated, followCmd := m.Update(msg)
	model := updated.(Model)
	if followCmd == nil {
		t.Fatal("autonomy result should trigger refresh commands")
	}

	content, err := os.ReadFile(filepath.Join(root, "notes", "tui.txt"))
	if err != nil {
		t.Fatalf("expected written file: %v", err)
	}
	if string(content) != "hello from tui\n" {
		t.Fatalf("unexpected file content: %q", string(content))
	}

	logCmd := exec.Command("git", "log", "-1", "--pretty=%s")
	logCmd.Dir = root
	out, err := logCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "tui autonomy commit" {
		t.Fatalf("latest commit = %q", got)
	}

	found := false
	for _, message := range model.chatView.Messages() {
		if strings.Contains(message.Content, "Autonomy Mode: execute") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("chat view should include autonomy execution summary")
	}
	if !model.explorerView.Files().Editable() {
		t.Fatal("files view should remain editable for local repo")
	}
}

func TestModel_Update_RequestFileOpCreateFile(t *testing.T) {
	root := t.TempDir()
	m := New()
	m.activeRepo = &repo.RepoContext{
		Owner:      "owner",
		Name:       "repo",
		FullName:   "owner/repo",
		LocalPaths: []string{root},
		IsLocal:    true,
	}

	_, cmd := m.Update(views.RequestFileOpMsg{Kind: views.FileOpCreateFile, Target: "notes/new.txt"})
	if cmd == nil {
		t.Fatal("RequestFileOpMsg should return a command")
	}
	msg := cmd()
	result, ok := msg.(views.FileOpResultMsg)
	if !ok {
		t.Fatalf("file op result = %T", msg)
	}
	if result.Err != nil {
		t.Fatalf("file op error = %v", result.Err)
	}
	if _, err := os.Stat(filepath.Join(root, "notes", "new.txt")); err != nil {
		t.Fatalf("created file missing: %v", err)
	}
}

func TestModel_Update_RequestFileOpMoveAndDelete(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "main.go")
	if err := os.WriteFile(source, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}

	m := New()
	m.activeRepo = &repo.RepoContext{
		Owner:      "owner",
		Name:       "repo",
		FullName:   "owner/repo",
		LocalPaths: []string{root},
		IsLocal:    true,
	}

	_, moveCmd := m.Update(views.RequestFileOpMsg{Kind: views.FileOpMove, Path: "main.go", Target: "cmd/main.go"})
	moveMsg := moveCmd()
	moveResult, ok := moveMsg.(views.FileOpResultMsg)
	if !ok {
		t.Fatalf("move result = %T", moveMsg)
	}
	if moveResult.Err != nil {
		t.Fatalf("move error = %v", moveResult.Err)
	}
	if _, err := os.Stat(filepath.Join(root, "cmd", "main.go")); err != nil {
		t.Fatalf("moved file missing: %v", err)
	}

	_, deleteCmd := m.Update(views.RequestFileOpMsg{Kind: views.FileOpDelete, Path: "cmd/main.go"})
	deleteMsg := deleteCmd()
	deleteResult, ok := deleteMsg.(views.FileOpResultMsg)
	if !ok {
		t.Fatalf("delete result = %T", deleteMsg)
	}
	if deleteResult.Err != nil {
		t.Fatalf("delete error = %v", deleteResult.Err)
	}
	if _, err := os.Stat(filepath.Join(root, "cmd", "main.go")); !os.IsNotExist(err) {
		t.Fatalf("file should be deleted, stat err = %v", err)
	}
}

func TestModel_Update_RequestFileOpRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	m := New()
	m.activeRepo = &repo.RepoContext{
		Owner:      "owner",
		Name:       "repo",
		FullName:   "owner/repo",
		LocalPaths: []string{root},
		IsLocal:    true,
	}

	_, cmd := m.Update(views.RequestFileOpMsg{Kind: views.FileOpCreateFile, Target: "../escape.txt"})
	msg := cmd()
	result, ok := msg.(views.FileOpResultMsg)
	if !ok {
		t.Fatalf("file op result = %T", msg)
	}
	if result.Err == nil {
		t.Fatal("expected path escape to be rejected")
	}
}
