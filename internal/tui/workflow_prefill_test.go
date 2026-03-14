package tui

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLLMProvider struct{}

func (stubLLMProvider) Name() string { return "stub" }
func (stubLLMProvider) Generate(context.Context, llm.GenerateRequest) (*llm.GenerateResponse, error) {
	return &llm.GenerateResponse{Text: `{"analysis":"ok","suggestions":[]}`}, nil
}
func (stubLLMProvider) GenerateStream(context.Context, llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk)
	close(ch)
	return ch, nil
}
func (stubLLMProvider) IsAvailable(context.Context) bool { return true }
func (stubLLMProvider) ModelInfo(context.Context) (*llm.ModelInfo, error) {
	return &llm.ModelInfo{Name: "stub"}, nil
}
func (stubLLMProvider) ListModels(context.Context) ([]llm.ModelInfo, error) {
	return []llm.ModelInfo{{Name: "stub", Provider: "openai"}}, nil
}
func (stubLLMProvider) SetModel(string)                       {}
func (stubLLMProvider) SetModelForRole(llm.ModelRole, string) {}

func TestUpdateWorkflowSelectBuildsWorkflowOrchestrationAndStartsLLM(t *testing.T) {
	m := NewModel()
	m.screen = screenWorkflowSelect
	m.llmProvider = stubLLMProvider{}
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
		Label:        "Pages / 静态站点",
		Goal:         "Plan Pages deployment",
		Capabilities: []string{"pages", "branch_rulesets"},
		Prefill: []workflowPrefillDefinition{
			{
				CapabilityID: "pages",
				Flow:         "inspect",
				Query:        map[string]string{"view": "latest_build"},
			},
			{
				CapabilityID: "branch_rulesets",
				Flow:         "inspect",
				Query:        map[string]string{"view": "branch_rules", "branch": "<default_branch>"},
			},
		},
	}}

	model, cmd := m.updateWorkflowSelect(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	updated := model.(Model)

	require.NotNil(t, cmd)
	require.NotNil(t, updated.workflowPlan)
	require.NotNil(t, updated.workflowFlow)
	require.Len(t, updated.workflowPlan.Steps, 2)
	require.Len(t, updated.workflowFlow.Steps, 2)
	assert.Equal(t, screenMain, updated.screen)
	assert.Equal(t, "pages", updated.workflowPlan.Steps[0].Capability)
	assert.Equal(t, "latest_build", updated.workflowPlan.Steps[0].Query["view"])
	assert.Equal(t, "main", updated.workflowPlan.Steps[1].Query["branch"])
	assert.Contains(t, updated.llmAnalysis, "Sending 2 schema-backed platform steps")
}

func TestBuildWorkflowOrchestrationFallsBackToSchemaHints(t *testing.T) {
	wf := workflowDefinition{
		ID:           "release",
		Label:        "发布版本",
		Goal:         "Prepare release notes and release assets",
		Capabilities: []string{"release"},
	}
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       "git@github.com:Joker-of-Gotham/gitdex.git",
			FetchURL:      "git@github.com:Joker-of-Gotham/gitdex.git",
			PushURLValid:  true,
			FetchURLValid: true,
		}},
	}

	flow := buildWorkflowOrchestration(wf, state)
	require.NotNil(t, flow)
	require.NotEmpty(t, flow.Steps)
	assert.Equal(t, "release", flow.Steps[0].Capability)
	assert.Equal(t, "inspect", flow.Steps[0].Flow)
}

func TestBuildWorkflowOrchestrationIncludesStructuredPayloads(t *testing.T) {
	wf := workflowDefinition{
		ID:           "pages_setup",
		Label:        "Pages",
		Goal:         "Configure Pages",
		Capabilities: []string{"pages"},
		Prefill: []workflowPrefillDefinition{{
			CapabilityID: "pages",
			Flow:         "mutate",
			Operation:    "update",
			Payload:      []byte(`{"build_type":"workflow","source":{"branch":"<default_branch>","path":"/"}}`),
			Validate:     []byte(`{"build_type":"workflow"}`),
			Rollback:     []byte(`{"build_type":"legacy"}`),
		}},
	}
	state := &status.GitState{
		RepoConfig:  git.RepoConfig{DefaultBranch: "main"},
		LocalBranch: git.BranchInfo{Name: "feature/pages"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       "git@github.com:Joker-of-Gotham/gitdex.git",
			FetchURL:      "git@github.com:Joker-of-Gotham/gitdex.git",
			PushURLValid:  true,
			FetchURLValid: true,
		}},
	}

	flow := buildWorkflowOrchestration(wf, state)
	require.NotNil(t, flow)
	require.Len(t, flow.Steps, 1)
	assert.JSONEq(t, `{"build_type":"workflow","source":{"branch":"main","path":"/"}}`, string(flow.Steps[0].Payload))
	assert.JSONEq(t, `{"build_type":"workflow"}`, string(flow.Steps[0].Validate))
	assert.JSONEq(t, `{"build_type":"legacy"}`, string(flow.Steps[0].Rollback))
}

func TestLoadWorkflowDefinitionsReleaseIncludesLifecyclePrefill(t *testing.T) {
	defs := loadWorkflowDefinitions()
	var release workflowDefinition
	found := false
	for _, item := range defs {
		if item.ID != "release" {
			continue
		}
		release = item
		found = true
		break
	}
	require.True(t, found, "release workflow definition missing")

	state := &status.GitState{
		RepoConfig:  git.RepoConfig{DefaultBranch: "main"},
		LocalBranch: git.BranchInfo{Name: "release/v1"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       "git@github.com:Joker-of-Gotham/gitdex.git",
			FetchURL:      "git@github.com:Joker-of-Gotham/gitdex.git",
			PushURLValid:  true,
			FetchURLValid: true,
		}},
	}

	flow := buildWorkflowOrchestration(release, state)
	require.NotNil(t, flow)
	require.GreaterOrEqual(t, len(flow.Steps), 6)

	var (
		sawCreate      bool
		sawAssetUpload bool
		sawPublish     bool
		sawAssetDelete bool
	)
	for _, step := range flow.Steps {
		switch step.Operation {
		case "create":
			sawCreate = true
			assert.JSONEq(t, `{"tag_name":"<release_tag>","target_commitish":"main","name":"<release_name>","body":"<release_notes>","draft":true}`, string(step.Payload))
			assert.JSONEq(t, `{"draft":true}`, string(step.Validate))
		case "asset_upload":
			sawAssetUpload = true
			assert.JSONEq(t, `{"file_path":"<release_asset_path>","asset_name":"<release_asset_name>","label":"<release_asset_label>"}`, string(step.Payload))
			assert.JSONEq(t, `{"delete_uploaded_asset":true}`, string(step.Rollback))
		case "publish_draft":
			sawPublish = true
			assert.Equal(t, "<draft_release_id>", step.ResourceID)
			assert.JSONEq(t, `{"draft":false}`, string(step.Validate))
			assert.JSONEq(t, `{"draft":true}`, string(step.Rollback))
		case "asset_delete":
			sawAssetDelete = true
			assert.Equal(t, "<draft_release_id>", step.Scope["release_id"])
			assert.JSONEq(t, `{"restore_deleted_asset":true}`, string(step.Rollback))
		}
	}
	assert.True(t, sawCreate)
	assert.True(t, sawAssetUpload)
	assert.True(t, sawPublish)
	assert.True(t, sawAssetDelete)
}

func TestLoadWorkflowDefinitionsPagesIncludesDomainAndReadinessValidation(t *testing.T) {
	defs := loadWorkflowDefinitions()
	var pages workflowDefinition
	found := false
	for _, item := range defs {
		if item.ID != "pages_setup" {
			continue
		}
		pages = item
		found = true
		break
	}
	require.True(t, found, "pages workflow definition missing")

	state := &status.GitState{
		RepoConfig:  git.RepoConfig{DefaultBranch: "main"},
		LocalBranch: git.BranchInfo{Name: "feature/pages"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       "git@github.com:Joker-of-Gotham/gitdex.git",
			FetchURL:      "git@github.com:Joker-of-Gotham/gitdex.git",
			PushURLValid:  true,
			FetchURLValid: true,
		}},
	}

	flow := buildWorkflowOrchestration(pages, state)
	require.NotNil(t, flow)
	require.GreaterOrEqual(t, len(flow.Steps), 10)

	var (
		sawDomainInspect       bool
		sawDNSInspect          bool
		sawBuildValidate       bool
		sawDomainValidate      bool
		sawCertificateValidate bool
	)
	for _, step := range flow.Steps {
		switch {
		case step.Flow == "inspect" && step.Query["view"] == "domain":
			sawDomainInspect = true
		case step.Flow == "inspect" && step.Query["view"] == "dns":
			sawDNSInspect = true
		case step.Flow == "validate" && step.Operation == "build":
			sawBuildValidate = true
			assert.JSONEq(t, `{"status":"built"}`, string(step.Payload))
		case step.Flow == "validate" && step.Operation == "update" && strings.Contains(string(step.Payload), `"protected_domain_state":"verified"`):
			sawDomainValidate = true
		case step.Flow == "validate" && step.Operation == "update":
			var payload map[string]any
			require.NoError(t, json.Unmarshal(step.Payload, &payload))
			certificate, _ := payload["https_certificate"].(map[string]any)
			if state, _ := certificate["state"].(string); state == "approved" {
				sawCertificateValidate = true
			}
		}
	}
	assert.True(t, sawDomainInspect)
	assert.True(t, sawDNSInspect)
	assert.True(t, sawBuildValidate)
	assert.True(t, sawDomainValidate)
	assert.True(t, sawCertificateValidate)
}

func TestBuildWorkflowOrchestrationKeepsDistinctValidateStepsWithDifferentPayloads(t *testing.T) {
	wf := workflowDefinition{
		ID:           "pages_setup",
		Label:        "Pages",
		Goal:         "Configure Pages",
		Capabilities: []string{"pages"},
		Prefill: []workflowPrefillDefinition{
			{
				CapabilityID: "pages",
				Flow:         "validate",
				Operation:    "update",
				Payload:      []byte(`{"protected_domain_state":"verified"}`),
			},
			{
				CapabilityID: "pages",
				Flow:         "validate",
				Operation:    "update",
				Payload:      []byte(`{"https_certificate":{"state":"approved"}}`),
			},
		},
	}
	state := &status.GitState{
		RepoConfig:  git.RepoConfig{DefaultBranch: "main"},
		LocalBranch: git.BranchInfo{Name: "feature/pages"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			PushURL:       "git@github.com:Joker-of-Gotham/gitdex.git",
			FetchURL:      "git@github.com:Joker-of-Gotham/gitdex.git",
			PushURLValid:  true,
			FetchURLValid: true,
		}},
	}

	flow := buildWorkflowOrchestration(wf, state)
	require.NotNil(t, flow)
	require.Len(t, flow.Steps, 2)
	assert.JSONEq(t, `{"protected_domain_state":"verified"}`, string(flow.Steps[0].Payload))
	assert.JSONEq(t, `{"https_certificate":{"state":"approved"}}`, string(flow.Steps[1].Payload))
}
