package tui

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/engine"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	platformruntime "github.com/Joker-of-Gotham/gitdex/internal/platform/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type hybridLLMProvider struct {
	text string
}

func (h hybridLLMProvider) Name() string { return "hybrid" }
func (h hybridLLMProvider) Generate(context.Context, llm.GenerateRequest) (*llm.GenerateResponse, error) {
	return &llm.GenerateResponse{Text: h.text}, nil
}
func (h hybridLLMProvider) GenerateStream(context.Context, llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk)
	close(ch)
	return ch, nil
}
func (h hybridLLMProvider) IsAvailable(context.Context) bool { return true }
func (h hybridLLMProvider) ModelInfo(context.Context) (*llm.ModelInfo, error) {
	return &llm.ModelInfo{Name: "hybrid"}, nil
}
func (h hybridLLMProvider) ListModels(context.Context) ([]llm.ModelInfo, error) {
	return []llm.ModelInfo{{Name: "hybrid", Provider: "openai"}}, nil
}
func (h hybridLLMProvider) SetModel(string)                       {}
func (h hybridLLMProvider) SetModelForRole(llm.ModelRole, string) {}

func TestHybridE2E_WorkflowLLMPlatformValidateRollback(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	provider := hybridLLMProvider{text: `{
  "analysis":"Pages workflow ready",
  "goal_status":"in_progress",
  "suggestions":[
    {
      "action":"Enable Pages workflow build",
      "reason":"Apply the Pages mutation before validation.",
      "interaction":"platform_exec",
      "capability_id":"pages",
      "flow":"mutate",
      "operation":"update",
      "payload":{"build_type":"workflow"},
      "validate_payload":{"build_type":"workflow"},
      "rollback_payload":{"build_type":"legacy"}
    }
  ]
}`}

	cfg := config.DefaultConfig()
	m := NewModel()
	m = m.SetLLMConfig(cfg.LLM)
	m.llmProvider = provider
	m.pipeline = engine.NewPipelineWithLLM("zen", provider, cfg.LLM)
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "feature/pages"},
		RepoConfig:  git.RepoConfig{DefaultBranch: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       "git@github.com:Joker-of-Gotham/gitdex.git",
			FetchURL:      "git@github.com:Joker-of-Gotham/gitdex.git",
			PushURLValid:  true,
			FetchURLValid: true,
		}},
	}
	m.workflows = []workflowDefinition{{
		ID:           "pages_setup",
		Label:        "Pages Setup",
		Goal:         "Configure Pages",
		Capabilities: []string{"pages"},
	}}
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return &platformruntime.Bundle{
			Platform: platform.PlatformGitHub,
			Executors: map[string]platform.AdminExecutor{
				"pages": fakeAdminExecutor{},
			},
			Adapter: platform.AdapterAPI,
		}, nil
	}
	m.screen = screenWorkflowSelect

	model, cmd := m.updateWorkflowSelect(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.(Model).Update(msg)
	afterLLM := model.(Model)
	require.Len(t, afterLLM.suggestions, 1)
	assert.Equal(t, git.PlatformExec, afterLLM.suggestions[0].Interaction)

	model, cmd = afterLLM.updateMain(tea.KeyPressMsg(tea.Key{Text: "y"}))
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.(Model).Update(msg)
	afterMutate := model.(Model)
	require.NotNil(t, afterMutate.lastPlatform)
	assert.Equal(t, "mutate", afterMutate.lastCommand.PlatformFlow)

	model, cmd = afterMutate.updateMain(tea.KeyPressMsg(tea.Key{Text: "v"}))
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.(Model).Update(msg)
	afterValidate := model.(Model)
	assert.Equal(t, "validate", afterValidate.lastCommand.PlatformFlow)

	model, cmd = afterValidate.updateMain(tea.KeyPressMsg(tea.Key{Text: "b"}))
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.(Model).Update(msg)
	afterRollback := model.(Model)
	assert.Equal(t, "rollback", afterRollback.lastCommand.PlatformFlow)
}

func TestHybridE2E_AutomationCheckpointRestoresLedgerAndFlow(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	fp := "https://github.com/example/pages-repo.git"
	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{Name: "origin", FetchURL: fp, PushURL: fp}},
	}
	m.loadedCheckpointRepo = m.repoFingerprint()
	m.session.ActiveGoal = "Keep Pages healthy"
	m.workflowPlan = &prompt.WorkflowOrchestration{
		WorkflowID:    "pages_setup",
		WorkflowLabel: "Pages Setup",
		Goal:          "Keep Pages healthy",
		Capabilities:  []string{"pages"},
		Steps: []prompt.WorkflowOrchestrationStep{{
			Title:      "Inspect Pages",
			Capability: "pages",
			Flow:       "inspect",
			Query:      map[string]string{"view": "site"},
		}},
	}
	m.syncWorkflowFlowFromPlan()
	m.mutationLedger = []platform.MutationLedgerEntry{{
		ID:           "ledger-1",
		Platform:     "github",
		CapabilityID: "pages",
		Flow:         "mutate",
		Operation:    "update",
		ResourceID:   "github-pages",
		Request:      json.RawMessage(`{"capability_id":"pages"}`),
	}}
	m.persistAutomationCheckpoint()

	restored := NewModel()
	restored.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{Name: "origin", FetchURL: fp, PushURL: fp}},
	}
	restored.reconcileRepoScopedState()
	require.NotNil(t, restored.workflowFlow)
	require.Len(t, restored.mutationLedger, 1)
	assert.Equal(t, "Keep Pages healthy", restored.session.ActiveGoal)
	assert.Equal(t, "ledger-1", restored.mutationLedger[0].ID)
}

func TestHybridE2E_CheckpointRestoreFlowResume(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	fp := "https://github.com/example/pages-repo.git"
	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{Name: "origin", FetchURL: fp, PushURL: fp}},
	}
	m.loadedCheckpointRepo = m.repoFingerprint()
	m.workflowPlan = &prompt.WorkflowOrchestration{
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
	m.syncWorkflowFlowFromPlan()
	require.NotNil(t, m.workflowFlow)
	m.workflowFlow.Steps[0].Status = workflowFlowPaused
	m.refreshWorkflowRunState("operator pause")
	m.persistAutomationCheckpoint()

	restored := NewModel()
	restored.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{Name: "origin", FetchURL: fp, PushURL: fp}},
	}
	restored.reconcileRepoScopedState()
	require.NotNil(t, restored.workflowFlow)
	require.Equal(t, workflowFlowPaused, restored.workflowFlow.Steps[0].Status)

	model, _ := restored.updateMain(tea.KeyPressMsg(tea.Key{Text: "R"}))
	resumed := model.(Model)
	assert.Equal(t, workflowFlowReady, resumed.workflowFlow.Steps[0].Status)
	assert.Equal(t, "approval_pending", resumed.workflowFlow.Health)
}

func TestHybridE2E_ReleaseAssetRollbackFlow(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "release/v1"},
		RepoConfig:  git.RepoConfig{DefaultBranch: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       "git@github.com:Joker-of-Gotham/gitdex.git",
			FetchURL:      "git@github.com:Joker-of-Gotham/gitdex.git",
			PushURLValid:  true,
			FetchURLValid: true,
		}},
	}
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return &platformruntime.Bundle{
			Platform: platform.PlatformGitHub,
			Executors: map[string]platform.AdminExecutor{
				"release": fakeAdminExecutor{
					capabilityID: "release",
					mutateFn: func(context.Context, platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
						return &platform.AdminMutationResult{
							CapabilityID: "release",
							Operation:    "asset_delete",
							ResourceID:   "asset-1",
							Metadata: map[string]string{
								"rollback_grade":           "reversible",
								"stored_bytes_ref":         "release-asset:test:gitdex.txt",
								"asset_name":               "gitdex.txt",
								"partial_restore_required": "false",
							},
							Before: &platform.AdminSnapshot{
								CapabilityID: "release",
								ResourceID:   "asset-1",
								State:        json.RawMessage(`{"name":"gitdex.txt"}`),
							},
						}, nil
					},
					rollbackFn: func(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
						return &platform.AdminRollbackResult{
							OK:      true,
							Summary: "deleted release asset restored",
							Snapshot: &platform.AdminSnapshot{
								CapabilityID: "release",
								ResourceID:   "asset-1",
								State:        json.RawMessage(`{"name":"gitdex.txt","restored":true}`),
							},
						}, nil
					},
				},
			},
			Adapter: platform.AdapterAPI,
		}, nil
	}
	m.suggestions = []git.Suggestion{{
		Action:      "Delete stale release asset",
		Reason:      "Clean up and keep rollback path available",
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "release",
			Flow:         "mutate",
			Operation:    "asset_delete",
			ResourceID:   "asset-1",
			Scope:        map[string]string{"asset_id": "asset-1", "release_id": "11"},
		},
	}}
	m.suggExecState = make([]git.ExecState, 1)
	m.suggExecMsg = make([]string, 1)

	model, cmd := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "y"}))
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.(Model).Update(msg)
	updated := model.(Model)
	require.NotNil(t, updated.lastPlatform)
	assert.Equal(t, "reversible", updated.lastPlatform.Mutation.Metadata["rollback_grade"])

	model, cmd = updated.updateMain(tea.KeyPressMsg(tea.Key{Text: "b"}))
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.(Model).Update(msg)
	rolledBack := model.(Model)
	assert.Equal(t, "rollback", rolledBack.lastCommand.PlatformFlow)
	assert.Contains(t, rolledBack.lastCommand.Output, "restored")
}

func TestHybridE2E_PagesDomainBuildLifecycle(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	m := NewModel()
	m.gitState = &status.GitState{
		LocalBranch: git.BranchInfo{Name: "feature/pages"},
		RepoConfig:  git.RepoConfig{DefaultBranch: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       "git@github.com:Joker-of-Gotham/gitdex.git",
			FetchURL:      "git@github.com:Joker-of-Gotham/gitdex.git",
			PushURLValid:  true,
			FetchURLValid: true,
		}},
	}
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return &platformruntime.Bundle{
			Platform: platform.PlatformGitHub,
			Executors: map[string]platform.AdminExecutor{
				"pages": fakeAdminExecutor{
					capabilityID: "pages",
					mutateFn: func(_ context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
						return &platform.AdminMutationResult{
							CapabilityID: "pages",
							Operation:    req.Operation,
							ResourceID:   "github-pages",
							After: &platform.AdminSnapshot{
								CapabilityID: "pages",
								ResourceID:   "github-pages",
								State:        json.RawMessage(`{"cname":"localhost","https_enforced":true,"https_certificate":{"state":"approved"},"protected_domain_state":"verified","status":"built"}`),
							},
						}, nil
					},
					validateFn: func(_ context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
						return &platform.AdminValidationResult{
							OK:         true,
							Summary:    "pages validated | DNS validated | readiness validated",
							ResourceID: "github-pages",
							Snapshot: &platform.AdminSnapshot{
								CapabilityID: "pages",
								ResourceID:   "github-pages",
								State:        json.RawMessage(`{"cname":"localhost","https_enforced":true,"https_certificate":{"state":"approved"},"protected_domain_state":"verified","status":"built"}`),
							},
						}, nil
					},
				},
			},
			Adapter: platform.AdapterAPI,
		}, nil
	}
	m.suggestions = []git.Suggestion{{
		Action:      "Configure Pages domain",
		Reason:      "Set domain and then validate DNS/HTTPS readiness",
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID:    "pages",
			Flow:            "mutate",
			Operation:       "update",
			Payload:         json.RawMessage(`{"cname":"localhost","https_enforced":true}`),
			ValidatePayload: json.RawMessage(`{"cname":"localhost","https_enforced":true}`),
		},
	}}
	m.suggExecState = make([]git.ExecState, 1)
	m.suggExecMsg = make([]string, 1)

	model, cmd := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "y"}))
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.(Model).Update(msg)
	updated := model.(Model)

	model, cmd = updated.updateMain(tea.KeyPressMsg(tea.Key{Text: "v"}))
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.(Model).Update(msg)
	validated := model.(Model)
	assert.Equal(t, "validate", validated.lastCommand.PlatformFlow)
	assert.Contains(t, validated.lastCommand.Output, "readiness")
}

func TestHybridE2E_WorkflowDeadLetterRetryAndCompensate(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	m := NewModel()
	m.gitState = &status.GitState{}
	m.workflowPlan = &prompt.WorkflowOrchestration{
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
			Rollback:   json.RawMessage(`{"build_type":"legacy"}`),
		}},
	}
	m.syncWorkflowFlowFromPlan()
	require.NotNil(t, m.workflowFlow)
	m.workflowFlow.Steps[0].Status = workflowFlowDeadLetter
	m.workflowFlow.Steps[0].DeadLetter = "certificate validation failed"
	m.workflowFlow.Steps[0].DeadLetterRef = m.recordDeadLetterEntry(&m.workflowFlow.Steps[0], m.workflowFlow.Steps[0].DeadLetter)
	m.refreshWorkflowRunState("")
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return &platformruntime.Bundle{
			Platform: platform.PlatformGitHub,
			Executors: map[string]platform.AdminExecutor{
				"pages": fakeAdminExecutor{
					capabilityID: "pages",
					rollbackFn: func(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
						return &platform.AdminRollbackResult{
							OK:      true,
							Summary: "compensated pages configuration",
							Snapshot: &platform.AdminSnapshot{
								CapabilityID: "pages",
								ResourceID:   "github-pages",
								State:        json.RawMessage(`{"build_type":"legacy","restored":true}`),
							},
						}, nil
					},
				},
			},
			Adapter: platform.AdapterAPI,
		}, nil
	}

	model, _ := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "X"}))
	retried := model.(Model)
	assert.False(t, retried.workflowHasDeadLetters())
	assert.Equal(t, workflowFlowReady, retried.workflowFlow.Steps[0].Status)

	retried.workflowFlow.Steps[0].Status = workflowFlowDeadLetter
	retried.workflowFlow.Steps[0].DeadLetter = "certificate validation failed"
	retried.workflowFlow.Steps[0].DeadLetterRef = retried.recordDeadLetterEntry(&retried.workflowFlow.Steps[0], retried.workflowFlow.Steps[0].DeadLetter)
	retried.refreshWorkflowRunState("")

	model, cmd := retried.updateMain(tea.KeyPressMsg(tea.Key{Text: "C"}))
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.(Model).Update(msg)
	compensated := model.(Model)
	assert.Equal(t, "rollback", compensated.lastCommand.PlatformFlow)
	assert.Equal(t, workflowFlowCompensated, compensated.workflowFlow.Steps[0].Status)
	assert.Contains(t, compensated.lastCommand.Output, "compensated")
}

func TestHybridE2E_TrustedUntrustedAutomationSplit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	base := NewModel()
	base.gitState = &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:owner/repo.git",
		}},
	}
	base.automation = config.AutomationConfig{
		Mode:           config.AutomationModeAuto,
		Enabled:        true,
		Unattended:     true,
		AutoAcceptSafe: true,
	}
	base.session.ActiveGoal = "Keep Pages configuration healthy"
	base.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return &platformruntime.Bundle{
			Platform: platform.PlatformGitHub,
			Executors: map[string]platform.AdminExecutor{
				"pages": fakeAdminExecutor{
					capabilityID: "pages",
				},
			},
			Adapter: platform.AdapterAPI,
		}, nil
	}
	base.suggestions = []git.Suggestion{{
		Action:      "Enable workflow-based Pages build",
		Reason:      "Apply the Pages mutation.",
		RiskLevel:   git.RiskSafe,
		Interaction: git.PlatformExec,
		PlatformOp: &git.PlatformExecInfo{
			CapabilityID: "pages",
			Flow:         "mutate",
			Operation:    "update",
			Payload:      json.RawMessage(`{"build_type":"workflow"}`),
		},
	}}
	base.suggExecState = make([]git.ExecState, 1)
	base.suggExecMsg = make([]string, 1)

	untrusted, cmd, ok := base.autoExecuteNextSafeSuggestion(false)
	require.False(t, ok)
	require.Nil(t, cmd)
	assert.Equal(t, 0, untrusted.autoSteps)

	trusted := base
	trusted.automation.TrustedMode = true
	model, cmd, ok := trusted.autoExecuteNextSafeSuggestion(false)
	require.True(t, ok)
	require.NotNil(t, cmd)
	msg := cmd()
	nextModel, _ := model.Update(msg)
	updated := nextModel.(Model)
	assert.Equal(t, "mutate", updated.lastCommand.PlatformFlow)
	assert.Equal(t, "pages", updated.lastCommand.PlatformCapability)
}
