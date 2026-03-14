package tui

import (
	"context"
	"encoding/json"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	platformruntime "github.com/Joker-of-Gotham/gitdex/internal/platform/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMain_WorkflowOperatorShortcuts(t *testing.T) {
	m := NewModel()
	m.screen = screenMain
	m.gitState = &status.GitState{}
	m.workflowFlow = &workflowFlowState{
		WorkflowID: "pages_setup",
		Steps: []workflowFlowStep{
			{
				Index:  0,
				Status: workflowFlowRunning,
				Step: prompt.WorkflowOrchestrationStep{
					Title:      "Deploy Pages",
					Capability: "pages",
					Flow:       "mutate",
					Operation:  "update",
					ResourceID: "github-pages",
					Rollback:   json.RawMessage(`{"build_type":"legacy"}`),
				},
			},
			{
				Index:      1,
				Status:     workflowFlowDeadLetter,
				DeadLetter: "validation failed",
				Step: prompt.WorkflowOrchestrationStep{
					Title:      "Validate Pages",
					Capability: "pages",
					Flow:       "mutate",
					Operation:  "update",
					ResourceID: "github-pages",
					Rollback:   json.RawMessage(`{"build_type":"legacy"}`),
				},
			},
		},
	}
	m.resolveAdminBundle = func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error) {
		return &platformruntime.Bundle{
			Platform: platform.PlatformGitHub,
			Executors: map[string]platform.AdminExecutor{
				"pages": fakeAdminExecutor{
					rollbackFn: func(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
						return &platform.AdminRollbackResult{OK: true, Summary: "compensated"}, nil
					},
				},
			},
			Adapter: platform.AdapterAPI,
		}, nil
	}

	model, _ := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "P"}))
	paused := model.(Model)
	assert.True(t, paused.workflowHasPausedSteps())

	model, _ = paused.updateMain(tea.KeyPressMsg(tea.Key{Text: "R"}))
	resumed := model.(Model)
	assert.False(t, resumed.workflowHasPausedSteps())
	assert.Equal(t, 0, resumed.workflowFlow.SelectedStepIndex)

	model, _ = resumed.updateMain(tea.KeyPressMsg(tea.Key{Text: ">"}))
	selected := model.(Model)
	assert.Equal(t, 1, selected.workflowFlow.SelectedStepIndex)

	model, _ = selected.updateMain(tea.KeyPressMsg(tea.Key{Text: "X"}))
	retried := model.(Model)
	assert.False(t, retried.workflowHasDeadLetters())

	// Recreate a dead-letter step so compensation path stays available.
	retried.workflowFlow.Steps[1].Status = workflowFlowDeadLetter
	model, cmd := retried.updateMain(tea.KeyPressMsg(tea.Key{Text: "C"}))
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.(Model).Update(msg)
	compensated := model.(Model)
	assert.Equal(t, resultKindPlatformAdmin, compensated.lastCommand.ResultKind)
	assert.Equal(t, "rollback", compensated.lastCommand.PlatformFlow)

	retried.workflowFlow.Steps[1].Status = workflowFlowDeadLetter
	retried.workflowFlow.Steps[1].DeadLetterRef = &DeadLetterEntry{Identity: retried.workflowFlow.Steps[1].Identity}
	model, _ = retried.updateMain(tea.KeyPressMsg(tea.Key{Text: "A"}))
	acked := model.(Model)
	assert.True(t, acked.workflowFlow.Steps[1].DeadLetterRef.Acked)

	acked.workflowFlow.Steps[1].Status = workflowFlowDeadLetter
	model, _ = acked.updateMain(tea.KeyPressMsg(tea.Key{Text: "K"}))
	skipped := model.(Model)
	assert.Equal(t, workflowFlowSkipped, skipped.workflowFlow.Steps[1].Status)
}

func TestUpdateMain_ClearsSelectedAutomationLock(t *testing.T) {
	m := NewModel()
	m.screen = screenMain
	m.workflowFlow = &workflowFlowState{
		SelectedStepIndex: 0,
		Steps: []workflowFlowStep{{
			Index:  0,
			Status: workflowFlowRunning,
			Policy: WorkflowStepPolicy{
				ConcurrencyKey: "pages:main",
			},
			Step: prompt.WorkflowOrchestrationStep{
				Title:      "Deploy Pages",
				Capability: "pages",
				Flow:       "mutate",
			},
		}},
	}
	m.automationLocks = map[string]string{"pages:main": "pages mutate"}
	m.refreshWorkflowRunState("")

	model, _ := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "u"}))
	updated := model.(Model)
	assert.Empty(t, updated.automationLocks)
	assert.Empty(t, updated.workflowFlow.ActiveLocks)
}

func TestUpdateMain_RecoversObserveOnlyAutomation(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.screen = screenMain
	m.workflowFlow = &workflowFlowState{
		SelectedStepIndex: 0,
		Steps: []workflowFlowStep{{
			Index:  0,
			Status: workflowFlowPaused,
			Step: prompt.WorkflowOrchestrationStep{
				Title:      "Validate Pages",
				Capability: "pages",
				Flow:       "validate",
			},
		}},
	}
	m.automationObserveOnly = true
	m.automationFailures = map[string]int{"pages": 3}

	model, _ := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "H"}))
	updated := model.(Model)
	assert.False(t, updated.automationObserveOnly)
	assert.Empty(t, updated.automationFailures)
	assert.False(t, updated.lastRecovery.IsZero())
}

func TestUpdateMain_ApprovesSelectedWorkflowStep(t *testing.T) {
	configureAutomationStateEnv(t)

	m := NewModel()
	m.screen = screenMain
	m.workflowFlow = &workflowFlowState{
		SelectedStepIndex: 0,
		Steps: []workflowFlowStep{{
			Index:  0,
			Status: workflowFlowReady,
			Policy: WorkflowStepPolicy{
				ApprovalRequired: true,
			},
			ApprovalState: "required",
			Step: prompt.WorkflowOrchestrationStep{
				Title:      "Publish release",
				Capability: "release",
				Flow:       "mutate",
				Operation:  "publish_draft",
			},
		}},
	}

	model, _ := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "Y"}))
	updated := model.(Model)
	assert.Equal(t, "approved", updated.workflowFlow.Steps[0].ApprovalState)
	assert.False(t, updated.workflowFlow.Steps[0].ApprovedAt.IsZero())
	assert.Equal(t, "clear", updated.workflowFlow.ApprovalState)
}
