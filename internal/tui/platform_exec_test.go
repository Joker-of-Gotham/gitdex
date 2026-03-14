package tui

import (
	"context"
	"encoding/json"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	platformruntime "github.com/Joker-of-Gotham/gitdex/internal/platform/runtime"
)

type fakeAdminExecutor struct {
	capabilityID string
	inspectFn    func(context.Context, platform.AdminInspectRequest) (*platform.AdminSnapshot, error)
	mutateFn     func(context.Context, platform.AdminMutationRequest) (*platform.AdminMutationResult, error)
	validateFn   func(context.Context, platform.AdminValidationRequest) (*platform.AdminValidationResult, error)
	rollbackFn   func(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error)
}

func (f fakeAdminExecutor) CapabilityID() string {
	if f.capabilityID != "" {
		return f.capabilityID
	}
	return "pages"
}

func (f fakeAdminExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	if f.inspectFn != nil {
		return f.inspectFn(ctx, req)
	}
	return &platform.AdminSnapshot{CapabilityID: f.CapabilityID(), State: json.RawMessage(`{"status":"ok"}`)}, nil
}

func (f fakeAdminExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	if f.mutateFn != nil {
		return f.mutateFn(ctx, req)
	}
	return &platform.AdminMutationResult{
		CapabilityID: f.CapabilityID(),
		Operation:    req.Operation,
		ResourceID:   "github-pages",
		Before: &platform.AdminSnapshot{
			CapabilityID: f.CapabilityID(),
			ResourceID:   "github-pages",
			State:        json.RawMessage(`{"build_type":"legacy"}`),
		},
		After: &platform.AdminSnapshot{
			CapabilityID: f.CapabilityID(),
			ResourceID:   "github-pages",
			State:        json.RawMessage(`{"build_type":"workflow"}`),
		},
	}, nil
}

func (f fakeAdminExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if f.validateFn != nil {
		return f.validateFn(ctx, req)
	}
	return &platform.AdminValidationResult{
		OK:         true,
		Summary:    "validated",
		ResourceID: req.ResourceID,
		Snapshot: &platform.AdminSnapshot{
			CapabilityID: f.CapabilityID(),
			ResourceID:   req.ResourceID,
			State:        json.RawMessage(`{"validated":true}`),
		},
	}, nil
}

func (f fakeAdminExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if f.rollbackFn != nil {
		return f.rollbackFn(ctx, req)
	}
	return &platform.AdminRollbackResult{
		OK:      true,
		Summary: "rolled back",
		Snapshot: &platform.AdminSnapshot{
			CapabilityID: f.CapabilityID(),
			ResourceID:   req.Mutation.ResourceID,
			State:        json.RawMessage(`{"build_type":"legacy"}`),
		},
	}, nil
}

func TestUpdateMain_AcceptsPlatformSuggestion(t *testing.T) {
	m := NewModel()
	m.gitState = &status.GitState{}
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return &platformruntime.Bundle{
			Platform: platform.PlatformGitHub,
			Executors: map[string]platform.AdminExecutor{
				"pages": fakeAdminExecutor{},
			},
		}, nil
	}
	m.suggestions = []git.Suggestion{{
		Action:      "Enable workflow-based Pages build",
		Reason:      "Need Pages on Actions",
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "pages",
			Flow:         "mutate",
			Operation:    "update",
			Payload:      json.RawMessage(`{"build_type":"workflow"}`),
		},
	}}
	m.suggExecState = make([]git.ExecState, len(m.suggestions))
	m.suggExecMsg = make([]string, len(m.suggestions))

	model, cmd := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "y"}))
	updated := model.(Model)
	if cmd == nil {
		t.Fatalf("expected execute command")
	}
	msg := cmd()
	model, _ = updated.Update(msg)
	updated = model.(Model)

	if updated.lastCommand.ResultKind != resultKindPlatformAdmin {
		t.Fatalf("expected platform result kind, got %s", updated.lastCommand.ResultKind)
	}
	if updated.lastPlatform == nil || updated.lastPlatform.Mutation == nil {
		t.Fatalf("expected last platform mutation to be recorded")
	}
	if updated.lastCommand.PlatformCapability != "pages" {
		t.Fatalf("unexpected capability: %s", updated.lastCommand.PlatformCapability)
	}
}

func TestUpdateMain_ValidateAndRollbackLatestPlatformMutation(t *testing.T) {
	validateCalls := 0
	rollbackCalls := 0

	m := NewModel()
	m.gitState = &status.GitState{}
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return &platformruntime.Bundle{
			Platform: platform.PlatformGitHub,
			Executors: map[string]platform.AdminExecutor{
				"pages": fakeAdminExecutor{
					validateFn: func(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
						validateCalls++
						return &platform.AdminValidationResult{
							OK:         true,
							Summary:    "validated",
							ResourceID: req.ResourceID,
						}, nil
					},
					rollbackFn: func(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
						rollbackCalls++
						return &platform.AdminRollbackResult{
							OK:      true,
							Summary: "rolled back",
						}, nil
					},
				},
			},
		}, nil
	}
	m.lastPlatform = &platformActionState{
		CapabilityID: "pages",
		Scope:        map[string]string{"scope": "repo"},
		Mutation: &platform.AdminMutationResult{
			CapabilityID: "pages",
			Operation:    "update",
			ResourceID:   "github-pages",
		},
	}

	model, cmd := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "v"}))
	updated := model.(Model)
	if cmd == nil {
		t.Fatalf("expected validate command")
	}
	msg := cmd()
	model, _ = updated.Update(msg)
	updated = model.(Model)
	if validateCalls != 1 || updated.lastCommand.PlatformFlow != "validate" {
		t.Fatalf("validate flow did not run")
	}
	if updated.lastCommand.PlatformOperation != "update" {
		t.Fatalf("expected validate to retain operation, got %s", updated.lastCommand.PlatformOperation)
	}

	model, cmd = updated.updateMain(tea.KeyPressMsg(tea.Key{Text: "b"}))
	updated = model.(Model)
	if cmd == nil {
		t.Fatalf("expected rollback command")
	}
	msg = cmd()
	model, _ = updated.Update(msg)
	updated = model.(Model)
	if rollbackCalls != 1 || updated.lastCommand.PlatformFlow != "rollback" {
		t.Fatalf("rollback flow did not run")
	}
	if updated.lastCommand.PlatformOperation != "update" {
		t.Fatalf("expected rollback to retain operation, got %s", updated.lastCommand.PlatformOperation)
	}
}

func TestBuildPlatformLedgerEntryUsesActualAdapterMeta(t *testing.T) {
	req := platformExecRequest{
		Op: &git.PlatformExecInfo{
			CapabilityID: "pages",
			Flow:         "mutate",
			Operation:    "update",
			ResourceID:   "github-pages",
		},
	}
	msg := platformExecResultMsg{
		Platform: platform.PlatformGitHub,
		Request:  req,
		Mutation: &platform.AdminMutationResult{
			CapabilityID: "pages",
			Operation:    "update",
			ResourceID:   "github-pages",
			ExecMeta: platform.ExecutionMeta{
				Adapter:  platform.AdapterBrowser,
				Rollback: platform.RollbackCompensating,
				Coverage: platform.CoveragePartial,
			},
		},
	}
	entry := buildPlatformLedgerEntry(platform.PlatformGitHub, req, msg, "step-1")
	if entry.ExecMeta.Adapter != platform.AdapterBrowser {
		t.Fatalf("expected browser adapter in ledger entry, got %+v", entry.ExecMeta)
	}
	trace := platformTraceFromResult(msg)
	if trace.PlatformAdapter != string(platform.AdapterBrowser) {
		t.Fatalf("expected browser adapter in trace, got %+v", trace)
	}
}

func TestBuildPlatformLedgerEntryRetainsValidationMetadata(t *testing.T) {
	req := platformExecRequest{
		Op: &git.PlatformExecInfo{
			CapabilityID: "pages",
			Flow:         "validate",
			Operation:    "update",
			ResourceID:   "github-pages",
		},
	}
	msg := platformExecResultMsg{
		Platform: platform.PlatformGitHub,
		Request:  req,
		Validation: &platform.AdminValidationResult{
			OK:         false,
			Summary:    "browser-backed stub requires operator validation",
			ResourceID: "github-pages",
			Metadata: map[string]string{
				"adapter_backed":               string(platform.AdapterBrowser),
				"browser_driver":               "playwright",
				"operator_validation_required": "true",
			},
			ExecMeta: platform.ExecutionMeta{
				Adapter:  platform.AdapterBrowser,
				Rollback: platform.RollbackNotSupported,
				Coverage: platform.CoveragePartial,
			},
		},
	}
	entry := buildPlatformLedgerEntry(platform.PlatformGitHub, req, msg, "step-validate")
	if entry.Metadata["browser_driver"] != "playwright" {
		t.Fatalf("expected validation metadata in ledger entry, got %+v", entry.Metadata)
	}
}

func TestBuildPlatformLedgerEntryRetainsRollbackMetadata(t *testing.T) {
	req := platformExecRequest{
		Op: &git.PlatformExecInfo{
			CapabilityID: "pages",
			Flow:         "rollback",
			Operation:    "update",
			ResourceID:   "github-pages",
		},
	}
	msg := platformExecResultMsg{
		Platform: platform.PlatformGitHub,
		Request:  req,
		Rollback: &platform.AdminRollbackResult{
			OK:      false,
			Summary: "browser-backed stub recorded manual recovery path",
			Metadata: map[string]string{
				"adapter_backed":             string(platform.AdapterBrowser),
				"browser_driver":             "playwright",
				"manual_completion_required": "true",
			},
			ExecMeta: platform.ExecutionMeta{
				Adapter:  platform.AdapterBrowser,
				Rollback: platform.RollbackNotSupported,
				Coverage: platform.CoveragePartial,
			},
		},
	}
	entry := buildPlatformLedgerEntry(platform.PlatformGitHub, req, msg, "step-rollback")
	if entry.Metadata["manual_completion_required"] != "true" {
		t.Fatalf("expected rollback metadata in ledger entry, got %+v", entry.Metadata)
	}
}

func TestExecutePlatformRequestBlocksInspectOnlyMutationDiagnostics(t *testing.T) {
	m := NewModel()
	m.gitState = &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:owner/repo.git",
		}},
	}
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		t.Fatal("resolve should not be called for blocked diagnostics")
		return nil, nil
	}

	cmd := m.executePlatformRequest(platformExecRequest{
		Op: &git.PlatformExecInfo{
			CapabilityID: "code_scanning_tool_settings",
			Flow:         "mutate",
			Operation:    "update",
		},
	})
	msg, ok := cmd().(platformExecResultMsg)
	if !ok {
		t.Fatalf("expected platform exec result")
	}
	if msg.Err == nil {
		t.Fatal("expected diagnostic block error")
	}
	if msg.Diagnostics.Decision != platform.DiagnosticBlocked {
		t.Fatalf("expected blocked diagnostics, got %+v", msg.Diagnostics)
	}
}

func TestExecutePlatformRequestAutoRepairsTokensBeforeMutation(t *testing.T) {
	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "feature/pages"},
		RepoConfig:  git.RepoConfig{DefaultBranch: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:owner/repo.git",
		}},
	}
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return &platformruntime.Bundle{
			Platform: platform.PlatformGitHub,
			Executors: map[string]platform.AdminExecutor{
				"pages": fakeAdminExecutor{
					mutateFn: func(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
						return &platform.AdminMutationResult{
							CapabilityID: "pages",
							Operation:    req.Operation,
							ResourceID:   "github-pages",
							After: &platform.AdminSnapshot{
								CapabilityID: "pages",
								ResourceID:   "github-pages",
								State:        req.Payload,
							},
						}, nil
					},
				},
			},
		}, nil
	}

	cmd := m.executePlatformRequest(platformExecRequest{
		Op: &git.PlatformExecInfo{
			CapabilityID: "pages",
			Flow:         "mutate",
			Operation:    "update",
			Payload:      json.RawMessage(`{"source":{"branch":"<default_branch>"}}`),
		},
	})
	msg, ok := cmd().(platformExecResultMsg)
	if !ok {
		t.Fatalf("expected platform exec result")
	}
	if msg.Err != nil {
		t.Fatal(msg.Err)
	}
	if msg.Diagnostics.Decision != platform.DiagnosticAutoRepair {
		t.Fatalf("expected auto repair diagnostics, got %+v", msg.Diagnostics)
	}
	if msg.Request.Op == nil {
		t.Fatal("expected repaired request")
	}
	if got := string(msg.Request.Op.Payload); got != `{"source":{"branch":"main"}}` {
		t.Fatalf("expected repaired payload, got %s", got)
	}
}
