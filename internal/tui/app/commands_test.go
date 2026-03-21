package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/your-org/gitdex/internal/gitops"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/views"
)

func TestFileCommands_CreateEditAndDelete(t *testing.T) {
	root := t.TempDir()

	m := New()
	m.activeRepo = &repo.RepoContext{
		Name:       "repo",
		FullName:   "owner/repo",
		LocalPaths: []string{root},
		IsLocal:    true,
	}
	m.registerRepoCommands()

	createResult := m.cmdHandlers["new"]("notes/todo.txt")
	if !strings.Contains(createResult, "已创建文件") {
		t.Fatalf("create result = %q", createResult)
	}

	target := filepath.Join(root, "notes", "todo.txt")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("created file missing: %v", err)
	}

	editResult := m.cmdHandlers["edit"]("notes/todo.txt -- hello")
	if !strings.Contains(editResult, "已写入文件") {
		t.Fatalf("edit result = %q", editResult)
	}

	appendResult := m.cmdHandlers["edit"]("notes/todo.txt ++ world")
	if !strings.Contains(appendResult, "已追加内容") {
		t.Fatalf("append result = %q", appendResult)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "helloworld" {
		t.Fatalf("content = %q, want %q", string(content), "helloworld")
	}

	blockedDelete := m.cmdHandlers["rm"]("notes/todo.txt")
	if !strings.Contains(blockedDelete, "删除已拦截") {
		t.Fatalf("blocked delete result = %q", blockedDelete)
	}

	deleteResult := m.cmdHandlers["rm"]("--confirm notes/todo.txt")
	if !strings.Contains(deleteResult, "已删除") {
		t.Fatalf("delete result = %q", deleteResult)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, stat err = %v", err)
	}
}

func TestFileCommands_MkdirMoveAndCopy(t *testing.T) {
	root := t.TempDir()

	m := New()
	m.activeRepo = &repo.RepoContext{
		Name:       "repo",
		FullName:   "owner/repo",
		LocalPaths: []string{root},
		IsLocal:    true,
	}
	m.registerRepoCommands()

	mkdirResult := m.cmdHandlers["mkdir"]("docs/specs")
	if !strings.Contains(mkdirResult, "Created directory") {
		t.Fatalf("mkdir result = %q", mkdirResult)
	}

	sourceFile := filepath.Join(root, "docs", "specs", "a.txt")
	if err := os.WriteFile(sourceFile, []byte("alpha"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	copyResult := m.cmdHandlers["cp"]("docs/specs/a.txt docs/specs/b.txt")
	if !strings.Contains(copyResult, "Copied") {
		t.Fatalf("copy result = %q", copyResult)
	}
	copied, err := os.ReadFile(filepath.Join(root, "docs", "specs", "b.txt"))
	if err != nil {
		t.Fatalf("ReadFile copied: %v", err)
	}
	if string(copied) != "alpha" {
		t.Fatalf("copied content = %q", string(copied))
	}

	moveResult := m.cmdHandlers["mv"]("docs/specs/b.txt docs/specs/c.txt")
	if !strings.Contains(moveResult, "Moved") {
		t.Fatalf("move result = %q", moveResult)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "specs", "b.txt")); !os.IsNotExist(err) {
		t.Fatalf("source should be moved away, stat err = %v", err)
	}
	moved, err := os.ReadFile(filepath.Join(root, "docs", "specs", "c.txt"))
	if err != nil {
		t.Fatalf("ReadFile moved: %v", err)
	}
	if string(moved) != "alpha" {
		t.Fatalf("moved content = %q", string(moved))
	}
}

func TestHandleIssueCreate_UsesGitHubAPI(t *testing.T) {
	var sawBody struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"number":   42,
			"title":    sawBody.Title,
			"html_url": "https://example.test/owner/repo/issues/42",
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client, err := ghclient.NewClientWithBaseURL(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	m := New()
	m.ghClient = client
	m.activeRepo = &repo.RepoContext{Owner: "owner", Name: "repo", FullName: "owner/repo"}
	m.registerRepoCommands()

	result := m.handleIssueCreate("Fix login -- body text")
	if !strings.Contains(result, "已创建 Issue #42") {
		t.Fatalf("result = %q", result)
	}
	if sawBody.Title != "Fix login" || sawBody.Body != "body text" {
		t.Fatalf("request payload = %+v", sawBody)
	}
}

func TestHandleReleaseCreate_UsesGitHubAPI(t *testing.T) {
	var sawBody struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Body    string `json:"body"`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":  sawBody.TagName,
			"html_url":  "https://example.test/owner/repo/releases/tag/" + sawBody.TagName,
			"draft":     false,
			"published": true,
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client, err := ghclient.NewClientWithBaseURL(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	m := New()
	m.ghClient = client
	m.activeRepo = &repo.RepoContext{Owner: "owner", Name: "repo", FullName: "owner/repo"}
	m.registerRepoCommands()

	result := m.handleReleaseCreate("v1.2.3 Release 1.2.3 -- ship it")
	if !strings.Contains(result, "已创建 Release v1.2.3") {
		t.Fatalf("result = %q", result)
	}
	if sawBody.TagName != "v1.2.3" || sawBody.Name != "Release 1.2.3" || sawBody.Body != "ship it" {
		t.Fatalf("request payload = %+v", sawBody)
	}
}

func TestActionsRunCommand_UsesGitHubAPI(t *testing.T) {
	var sawRequest struct {
		Ref string `json:"ref"`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/actions/workflows/123/dispatches", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&sawRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client, err := ghclient.NewClientWithBaseURL(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	m := New()
	m.ghClient = client
	m.activeRepo = &repo.RepoContext{
		Owner:         "owner",
		Name:          "repo",
		FullName:      "owner/repo",
		DefaultBranch: "main",
	}
	m.registerRepoCommands()

	result := m.cmdHandlers["actions"]("run 123")
	if !strings.Contains(result, "已触发 workflow 123") {
		t.Fatalf("result = %q", result)
	}
	if sawRequest.Ref != "main" {
		t.Fatalf("dispatch ref = %q, want %q", sawRequest.Ref, "main")
	}
}

func TestCloneCommand_QueuesCloneAndPromotesRepoToLocal(t *testing.T) {
	root := t.TempDir()

	m := New()
	m.activeRepo = &repo.RepoContext{
		Owner:         "owner",
		Name:          "repo",
		FullName:      "owner/repo",
		DefaultBranch: "main",
		IsLocal:       false,
		IsReadOnly:    true,
	}
	m.cloneRemote = func(ctx context.Context, url, dir string, opts gitops.CloneOptions) error {
		if url != "https://github.com/owner/repo.git" {
			t.Fatalf("clone url = %q", url)
		}
		if dir != root {
			t.Fatalf("clone dir = %q, want %q", dir, root)
		}
		return nil
	}
	m.registerRepoCommands()

	result := m.cmdHandlers["clone"](root)
	if !strings.Contains(result, "开始克隆 owner/repo") {
		t.Fatalf("clone result = %q", result)
	}
	cmd := m.drainPostCommand()
	if cmd == nil {
		t.Fatal("expected clone command to be queued")
	}
	msg := cmd()
	cloneMsg, ok := msg.(views.CloneRepoResultMsg)
	if !ok {
		t.Fatalf("queued command returned %T, want views.CloneRepoResultMsg", msg)
	}
	updated, follow := m.Update(cloneMsg)
	if follow == nil {
		t.Fatal("clone completion should trigger follow-up refresh commands")
	}
	model := updated.(Model)
	if model.activeRepo == nil || !model.activeRepo.IsLocal {
		t.Fatal("active repo should be promoted to local after clone completes")
	}
	if model.activeRepo.LocalPath() != root {
		t.Fatalf("local path = %q, want %q", model.activeRepo.LocalPath(), root)
	}
	followMsg := follow()
	if _, ok := followMsg.(tea.BatchMsg); !ok {
		t.Fatalf("follow command returned %T, want tea.BatchMsg", followMsg)
	}
}
