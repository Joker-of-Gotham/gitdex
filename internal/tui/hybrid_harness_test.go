package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	platformruntime "github.com/Joker-of-Gotham/gitdex/internal/platform/runtime"
)

type hybridHarness struct {
	t      *testing.T
	root   string
	repo   *gitSandboxRepo
	replay *platformReplayServer
	model  Model
}

type gitSandboxRepo struct {
	Root      string
	RemoteURL string
}

type replayRequest struct {
	Method string
	Path   string
	Body   map[string]any
}

type replayHandler func(map[string]any) (int, any)

type platformReplayServer struct {
	t        *testing.T
	server   *httptest.Server
	mu       sync.Mutex
	handlers map[string]replayHandler
	requests []replayRequest
}

type replayAdminExecutor struct {
	baseURL      string
	httpClient   *http.Client
	capabilityID string
}

func newHybridHarness(t *testing.T, capabilityIDs ...string) *hybridHarness {
	t.Helper()
	root := t.TempDir()
	return newHybridHarnessFromRoot(t, root, capabilityIDs...)
}

func newHybridHarnessFromRoot(t *testing.T, root string, capabilityIDs ...string) *hybridHarness {
	t.Helper()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	repo := ensureGitSandboxRepo(t, filepath.Join(root, "repo"))
	replay := newPlatformReplayServer(t)
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(repo.Root); err != nil {
		t.Fatalf("chdir sandbox repo: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
		replay.Close()
	})

	executors := make(map[string]platform.AdminExecutor, len(capabilityIDs))
	for _, capabilityID := range capabilityIDs {
		capabilityID = strings.TrimSpace(capabilityID)
		if capabilityID == "" {
			continue
		}
		executors[capabilityID] = replayAdminExecutor{
			baseURL:      replay.URL(),
			httpClient:   replay.server.Client(),
			capabilityID: capabilityID,
		}
	}

	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main", Upstream: "origin/main"},
		RepoConfig:  git.RepoConfig{DefaultBranch: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       repo.RemoteURL,
			FetchURL:      repo.RemoteURL,
			PushURLValid:  true,
			FetchURLValid: true,
		}},
	}
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return &platformruntime.Bundle{
			Platform:        platform.PlatformGitHub,
			RemoteURL:       repo.RemoteURL,
			Executors:       executors,
			Adapter:         platform.AdapterAPI,
			ExecutorAdapter: platform.NewDirectAdapterExecutor(platform.AdapterAPI),
		}, nil
	}

	m.reconcileRepoScopedState()

	return &hybridHarness{
		t:      t,
		root:   root,
		repo:   repo,
		replay: replay,
		model:  m,
	}
}

func ensureGitSandboxRepo(t *testing.T, root string) *gitSandboxRepo {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir sandbox repo: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for hybrid sandbox tests")
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		return &gitSandboxRepo{
			Root:      root,
			RemoteURL: "git@github.com:Joker-of-Gotham/gitdex.git",
		}
	}
	runGit(t, root, "init", "-b", "main")
	runGit(t, root, "config", "user.email", "gitdex@example.com")
	runGit(t, root, "config", "user.name", "Gitdex Test")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# sandbox\n"), 0o600); err != nil {
		t.Fatalf("write sandbox file: %v", err)
	}
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "init sandbox")
	runGit(t, root, "remote", "add", "origin", "git@github.com:Joker-of-Gotham/gitdex.git")
	return &gitSandboxRepo{
		Root:      root,
		RemoteURL: "git@github.com:Joker-of-Gotham/gitdex.git",
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func newPlatformReplayServer(t *testing.T) *platformReplayServer {
	t.Helper()
	p := &platformReplayServer{
		t:        t,
		handlers: map[string]replayHandler{},
	}
	p.server = httptest.NewServer(http.HandlerFunc(p.serveHTTP))
	return p
}

func (p *platformReplayServer) Handle(method, path string, handler replayHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[strings.ToUpper(strings.TrimSpace(method))+" "+strings.TrimSpace(path)] = handler
}

func (p *platformReplayServer) Requests() []replayRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]replayRequest, len(p.requests))
	copy(out, p.requests)
	return out
}

func (p *platformReplayServer) URL() string {
	return p.server.URL
}

func (p *platformReplayServer) Close() {
	if p.server != nil {
		p.server.Close()
	}
}

func (p *platformReplayServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	body := map[string]any{}
	if data, err := io.ReadAll(r.Body); err == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &body); err != nil {
			p.t.Fatalf("decode replay body %s %s: %v", r.Method, r.URL.Path, err)
		}
	}
	key := strings.ToUpper(strings.TrimSpace(r.Method)) + " " + strings.TrimSpace(r.URL.Path)

	p.mu.Lock()
	p.requests = append(p.requests, replayRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Body:   body,
	})
	handler := p.handlers[key]
	p.mu.Unlock()

	if handler == nil {
		http.Error(w, "replay handler not found", http.StatusNotFound)
		return
	}
	code, response := handler(body)
	writeJSON(p.t, w, code, response)
}

func (e replayAdminExecutor) CapabilityID() string {
	return e.capabilityID
}

func (e replayAdminExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	var out platform.AdminSnapshot
	if err := e.post(ctx, "/inspect/"+e.capabilityID, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (e replayAdminExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	var out platform.AdminMutationResult
	if err := e.post(ctx, "/mutate/"+e.capabilityID+"/"+strings.TrimSpace(req.Operation), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (e replayAdminExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	op := ""
	if req.Mutation != nil {
		op = strings.TrimSpace(req.Mutation.Operation)
	}
	var out platform.AdminValidationResult
	path := "/validate/" + e.capabilityID
	if op != "" {
		path += "/" + op
	}
	if err := e.post(ctx, path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (e replayAdminExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	op := ""
	if req.Mutation != nil {
		op = strings.TrimSpace(req.Mutation.Operation)
	}
	var out platform.AdminRollbackResult
	path := "/rollback/" + e.capabilityID
	if op != "" {
		path += "/" + op
	}
	if err := e.post(ctx, path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (e replayAdminExecutor) post(ctx context.Context, path string, payload any, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(e.baseURL, "/")+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (h *hybridHarness) execute(req platformExecRequest) Model {
	h.t.Helper()
	cmd := h.model.executePlatformRequest(req)
	if cmd == nil {
		h.t.Fatal("expected platform command")
	}
	msg := cmd()
	model, _ := h.model.Update(msg)
	h.model = model.(Model)
	return h.model
}

func (h *hybridHarness) press(text string) Model {
	h.t.Helper()
	model, cmd := h.model.updateMain(tea.KeyPressMsg(tea.Key{Text: text}))
	h.model = model.(Model)
	if cmd != nil {
		msg := cmd()
		model, _ = h.model.Update(msg)
		h.model = model.(Model)
	}
	return h.model
}

func TestHybridHarness_ReleaseAssetLifecycle(t *testing.T) {
	h := newHybridHarness(t, "release")
	h.replay.Handle("POST", "/mutate/release/create", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminMutationResult{
			CapabilityID: "release",
			Operation:    "create",
			ResourceID:   "11",
			After: &platform.AdminSnapshot{
				CapabilityID: "release",
				ResourceID:   "11",
				State:        rawJSON(t, map[string]any{"id": 11, "tag_name": "v1.0.0", "draft": true}),
			},
		}
	})
	h.replay.Handle("POST", "/mutate/release/asset_upload", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminMutationResult{
			CapabilityID: "release",
			Operation:    "asset_upload",
			ResourceID:   "99",
			Metadata:     map[string]string{"release_id": "11", "rollback_grade": "reversible"},
			After: &platform.AdminSnapshot{
				CapabilityID: "release",
				ResourceID:   "99",
				State:        rawJSON(t, map[string]any{"id": 99, "name": "gitdex.txt"}),
			},
		}
	})
	h.replay.Handle("POST", "/inspect/release", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminSnapshot{
			CapabilityID: "release",
			ResourceID:   "11",
			State:        rawJSON(t, []map[string]any{{"id": 99, "name": "gitdex.txt"}}),
		}
	})
	h.replay.Handle("POST", "/mutate/release/publish_draft", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminMutationResult{
			CapabilityID: "release",
			Operation:    "publish_draft",
			ResourceID:   "11",
			After: &platform.AdminSnapshot{
				CapabilityID: "release",
				ResourceID:   "11",
				State:        rawJSON(t, map[string]any{"id": 11, "tag_name": "v1.0.0", "draft": false}),
			},
		}
	})
	h.replay.Handle("POST", "/validate/release/publish_draft", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminValidationResult{
			OK:         true,
			Summary:    "release published",
			ResourceID: "11",
			Snapshot: &platform.AdminSnapshot{
				CapabilityID: "release",
				ResourceID:   "11",
				State:        rawJSON(t, map[string]any{"id": 11, "draft": false}),
			},
		}
	})
	h.replay.Handle("POST", "/mutate/release/asset_delete", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminMutationResult{
			CapabilityID: "release",
			Operation:    "asset_delete",
			ResourceID:   "99",
			Metadata: map[string]string{
				"release_id":               "11",
				"asset_name":               "gitdex.txt",
				"stored_bytes_ref":         "release-asset:test:gitdex.txt",
				"rollback_grade":           "reversible",
				"partial_restore_required": "false",
			},
			Before: &platform.AdminSnapshot{
				CapabilityID: "release",
				ResourceID:   "99",
				State:        rawJSON(t, map[string]any{"id": 99, "name": "gitdex.txt"}),
			},
		}
	})
	h.replay.Handle("POST", "/rollback/release/asset_delete", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminRollbackResult{
			OK:      true,
			Summary: "deleted release asset restored",
			Snapshot: &platform.AdminSnapshot{
				CapabilityID: "release",
				ResourceID:   "99",
				State:        rawJSON(t, map[string]any{"id": 99, "name": "gitdex.txt", "restored": true}),
			},
		}
	})

	h.execute(platformExecRequest{Op: &git.PlatformExecInfo{CapabilityID: "release", Flow: "mutate", Operation: "create", Payload: rawJSON(t, map[string]any{"tag_name": "v1.0.0", "draft": true})}})
	h.execute(platformExecRequest{Op: &git.PlatformExecInfo{CapabilityID: "release", Flow: "mutate", Operation: "asset_upload", Scope: map[string]string{"release_id": "11"}, Payload: rawJSON(t, map[string]any{"name": "gitdex.txt"})}})
	h.execute(platformExecRequest{Op: &git.PlatformExecInfo{CapabilityID: "release", Flow: "inspect", Query: map[string]string{"view": "assets", "release_id": "11"}}})
	h.execute(platformExecRequest{Op: &git.PlatformExecInfo{CapabilityID: "release", Flow: "mutate", Operation: "publish_draft", ResourceID: "11"}})
	h.execute(platformExecRequest{Op: &git.PlatformExecInfo{CapabilityID: "release", Flow: "validate", Operation: "publish_draft", ResourceID: "11"}, Mutation: cloneMutation(h.model.lastPlatform.Mutation)})
	h.execute(platformExecRequest{Op: &git.PlatformExecInfo{CapabilityID: "release", Flow: "mutate", Operation: "asset_delete", ResourceID: "99", Scope: map[string]string{"release_id": "11", "asset_id": "99"}}})
	h.execute(platformExecRequest{Op: &git.PlatformExecInfo{CapabilityID: "release", Flow: "rollback", Operation: "asset_delete", ResourceID: "99"}, Mutation: cloneMutation(h.model.lastPlatform.Mutation)})

	requests := h.replay.Requests()
	if len(requests) != 7 {
		t.Fatalf("expected 7 replayed requests, got %d", len(requests))
	}
	if h.model.lastCommand.PlatformFlow != "rollback" || !strings.Contains(h.model.lastCommand.Output, "restored") {
		t.Fatalf("expected restored rollback trace, got %+v", h.model.lastCommand)
	}
	if _, err := os.Stat(filepath.Join(h.repo.Root, ".git")); err != nil {
		t.Fatalf("expected git sandbox repo, got %v", err)
	}
}

func TestHybridHarness_PagesDomainBuildLifecycle(t *testing.T) {
	h := newHybridHarness(t, "pages")
	h.replay.Handle("POST", "/mutate/pages/update", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminMutationResult{
			CapabilityID: "pages",
			Operation:    "update",
			ResourceID:   "github-pages",
			After: &platform.AdminSnapshot{
				CapabilityID: "pages",
				ResourceID:   "github-pages",
				State: rawJSON(t, map[string]any{
					"cname":                  "localhost",
					"build_type":             "workflow",
					"protected_domain_state": "verified",
					"https_enforced":         true,
					"https_certificate":      map[string]any{"state": "approved"},
				}),
			},
		}
	})
	h.replay.Handle("POST", "/mutate/pages/build", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminMutationResult{
			CapabilityID: "pages",
			Operation:    "build",
			ResourceID:   "github-pages",
			After: &platform.AdminSnapshot{
				CapabilityID: "pages",
				ResourceID:   "github-pages",
				State:        rawJSON(t, map[string]any{"status": "built"}),
			},
		}
	})
	h.replay.Handle("POST", "/validate/pages/build", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminValidationResult{
			OK:         true,
			Summary:    "pages validated | DNS validated | readiness validated",
			ResourceID: "github-pages",
			Snapshot: &platform.AdminSnapshot{
				CapabilityID: "pages",
				ResourceID:   "github-pages",
				State:        rawJSON(t, map[string]any{"status": "built", "protected_domain_state": "verified"}),
			},
		}
	})

	h.execute(platformExecRequest{Op: &git.PlatformExecInfo{CapabilityID: "pages", Flow: "mutate", Operation: "update", Payload: rawJSON(t, map[string]any{"cname": "localhost", "https_enforced": true})}})
	h.execute(platformExecRequest{Op: &git.PlatformExecInfo{CapabilityID: "pages", Flow: "mutate", Operation: "build"}})
	h.execute(platformExecRequest{Op: &git.PlatformExecInfo{CapabilityID: "pages", Flow: "validate", Operation: "build", ResourceID: "github-pages"}, Mutation: cloneMutation(h.model.lastPlatform.Mutation)})

	if h.model.lastCommand.PlatformFlow != "validate" || !strings.Contains(h.model.lastCommand.Output, "readiness") {
		t.Fatalf("expected pages readiness validation, got %+v", h.model.lastCommand)
	}
	if len(h.replay.Requests()) != 3 {
		t.Fatalf("expected 3 replayed requests, got %d", len(h.replay.Requests()))
	}
}

func TestHybridHarness_WorkflowDeadLetterRetryAndCompensate(t *testing.T) {
	h := newHybridHarness(t, "pages")
	h.replay.Handle("POST", "/rollback/pages", func(body map[string]any) (int, any) {
		return http.StatusOK, &platform.AdminRollbackResult{
			OK:      true,
			Summary: "compensated pages configuration",
			Snapshot: &platform.AdminSnapshot{
				CapabilityID: "pages",
				ResourceID:   "github-pages",
				State:        rawJSON(t, map[string]any{"build_type": "legacy", "restored": true}),
			},
		}
	})

	h.model.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "pages_setup",
		WorkflowLabel: "Pages Setup",
		Goal:          "Recover Pages health",
		Capabilities:  []string{"pages"},
		Steps: []prompt.WorkflowOrchestrationStep{{
			Title:      "Fix Pages config",
			Capability: "pages",
			Flow:       "mutate",
			Operation:  "update",
			ResourceID: "github-pages",
			Rollback:   rawJSON(t, map[string]any{"build_type": "legacy"}),
		}},
	}
	h.model.syncWorkflowFlowFromPlan()
	h.model.workflowFlow.Steps[0].Status = workflowFlowDeadLetter
	h.model.workflowFlow.Steps[0].DeadLetter = "certificate validation failed"
	h.model.workflowFlow.Steps[0].DeadLetterRef = h.model.recordDeadLetterEntry(&h.model.workflowFlow.Steps[0], h.model.workflowFlow.Steps[0].DeadLetter)
	h.model.refreshWorkflowRunState("")

	h.press("X")
	if h.model.workflowFlow.Steps[0].Status != workflowFlowReady {
		t.Fatalf("expected operator retry to restore ready state, got %s", h.model.workflowFlow.Steps[0].Status)
	}

	h.model.workflowFlow.Steps[0].Status = workflowFlowDeadLetter
	h.model.workflowFlow.Steps[0].DeadLetter = "certificate validation failed"
	h.model.workflowFlow.Steps[0].DeadLetterRef = h.model.recordDeadLetterEntry(&h.model.workflowFlow.Steps[0], h.model.workflowFlow.Steps[0].DeadLetter)
	h.model.refreshWorkflowRunState("")
	h.press("C")

	if h.model.workflowFlow.Steps[0].Status != workflowFlowCompensated {
		t.Fatalf("expected compensation to update status, got %s", h.model.workflowFlow.Steps[0].Status)
	}
	if h.model.lastCommand.PlatformFlow != "rollback" {
		t.Fatalf("expected rollback flow, got %+v", h.model.lastCommand)
	}
	if len(h.replay.Requests()) != 1 {
		t.Fatalf("expected single replayed compensation request, got %d", len(h.replay.Requests()))
	}
}

func TestHybridHarness_CheckpointRestoreFlowResume(t *testing.T) {
	root := t.TempDir()
	h := newHybridHarnessFromRoot(t, root)
	h.model.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "pages_setup",
		WorkflowLabel: "Pages Setup",
		Goal:          "Keep Pages healthy",
		Capabilities:  []string{"pages"},
		Steps: []prompt.WorkflowOrchestrationStep{{
			Title:      "Trigger Pages build",
			Capability: "pages",
			Flow:       "mutate",
			Operation:  "build",
		}},
	}
	h.model.syncWorkflowFlowFromPlan()
	h.model.workflowFlow.Steps[0].Status = workflowFlowPaused
	h.model.refreshWorkflowRunState("operator pause")
	h.model.persistAutomationCheckpoint()

	restored := newHybridHarnessFromRoot(t, root)
	if restored.model.workflowFlow == nil || restored.model.workflowFlow.Steps[0].Status != workflowFlowPaused {
		t.Fatalf("expected paused step after checkpoint restore, got %+v", restored.model.workflowFlow)
	}
	restored.press("R")
	if restored.model.workflowFlow.Steps[0].Status != workflowFlowReady {
		t.Fatalf("expected restored flow to resume, got %s", restored.model.workflowFlow.Steps[0].Status)
	}
	if restored.model.workflowFlow.Health != "approval_pending" {
		t.Fatalf("expected approval pending health, got %s", restored.model.workflowFlow.Health)
	}
}

func rawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal raw json: %v", err)
	}
	return data
}

func writeJSON(t *testing.T, w http.ResponseWriter, code int, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if value == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode replay response: %v", err)
	}
}
