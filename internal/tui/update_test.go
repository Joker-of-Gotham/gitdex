package tui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/ollama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleInputPaste(t *testing.T) {
	m := Model{
		screen:        screenInput,
		inputFields:   []git.InputField{{Label: "Remote URL", ArgIndex: 4}},
		inputValues:   []string{""},
		inputIdx:      0,
		inputCursorAt: 0,
	}

	model, _ := m.Update(tea.PasteMsg{Content: "git@github.com:user/repo.git"})
	updated, ok := model.(Model)
	assert.True(t, ok)

	assert.Equal(t, "git@github.com:user/repo.git", updated.inputValues[0])
	assert.Equal(t, len("git@github.com:user/repo.git"), updated.inputCursorAt)
}

func TestUpdateInputUsesKeyText(t *testing.T) {
	m := Model{
		screen:        screenInput,
		inputFields:   []git.InputField{{Label: "Remote URL", ArgIndex: 4}},
		inputValues:   []string{""},
		inputIdx:      0,
		inputCursorAt: 0,
	}

	msg := tea.KeyPressMsg(tea.Key{Text: "https://github.com/user/repo.git"})
	model, _ := m.Update(msg)
	updated, ok := model.(Model)
	assert.True(t, ok)

	assert.Equal(t, "https://github.com/user/repo.git", updated.inputValues[0])
}

func TestUpdateInputBackspaceHandlesUnicode(t *testing.T) {
	m := Model{
		screen:        screenInput,
		inputFields:   []git.InputField{{Label: "Comment", ArgIndex: 1}},
		inputValues:   []string{"你好世界"},
		inputIdx:      0,
		inputCursorAt: runeLen("你好世界"),
	}

	model, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace, Text: "backspace"}))
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Equal(t, "你好世", updated.inputValues[0])
	assert.Equal(t, runeLen("你好世"), updated.inputCursorAt)
}

func TestUpdateMain_InfoOnlyAdvancesAndRefreshes(t *testing.T) {
	m := NewModel()
	m.suggestions = []git.Suggestion{
		{Action: "Review advisory", Reason: "Inspect the current plan", Interaction: git.InfoOnly},
		{Action: "Commit", Command: []string{"git", "commit", "-m", "test"}, Interaction: git.AutoExec},
	}
	m.suggExecState = make([]git.ExecState, len(m.suggestions))
	m.suggExecMsg = make([]string, len(m.suggestions))

	model, cmd := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "y"}))
	updated, ok := model.(Model)
	assert.True(t, ok)

	assert.NotNil(t, cmd)
	assert.Equal(t, 1, updated.suggIdx)
	assert.Equal(t, git.ExecDone, updated.suggExecState[0])
	assert.Equal(t, "Review advisory", updated.lastCommand.Title)
	assert.Equal(t, "advisory viewed", updated.lastCommand.Status)
}

func TestUpdateGoalInputEscapeCancels(t *testing.T) {
	m := NewModel()
	m.screen = screenGoalInput
	m.goalInput = "修复乱码"
	m.goalCursorAt = runeLen(m.goalInput)

	model, _ := m.updateGoalInput(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Equal(t, screenMain, updated.screen)
	assert.Equal(t, "修复乱码", updated.goalInput)
}

func TestPersistProviderConfigUpdatesRuntimeProvider(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	cfg := config.DefaultConfig()
	config.Set(cfg)

	m := NewModel()
	m = m.SetLLMConfig(cfg.LLM)
	m = m.openProviderConfig(selectPrimary)
	m.providerDraft.Provider = "openai"
	m.providerDraft.Model = "gpt-4.1-mini"
	m.providerDraft.Endpoint = "https://api.openai.com/v1"
	m.providerDraft.APIKey = "test-key"

	model, cmd := m.persistProviderConfig()
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Nil(t, cmd)
	assert.Equal(t, "openai", updated.primaryProvider)
	assert.Equal(t, "gpt-4.1-mini", updated.selectedPrimary)
	assert.NotNil(t, updated.llmProvider)
	assert.Equal(t, screenModelSelect, updated.screen)
	assert.Equal(t, selectSecondary, updated.modelSelectPhase)
}

func TestUpdateMain_ModelShortcutOpensAISetup(t *testing.T) {
	m := NewModel()
	m.screen = screenMain
	m.primaryProvider = "openai"

	model, _ := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "m"}))
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Equal(t, screenModelSelect, updated.screen)
	assert.Equal(t, modelSelectProviders, updated.modelSelectMode)
	assert.Equal(t, providerOptionIndex("openai"), updated.modelCursor)
}

func TestUpdateModelSelect_OpenAIProviderOpensConfig(t *testing.T) {
	m := NewModel()
	m = m.openModelSetup(selectPrimary)
	m.modelCursor = providerOptionIndex("openai")

	model, _ := m.updateModelSelect(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Equal(t, screenProviderConfig, updated.screen)
	assert.Equal(t, "openai", updated.providerDraft.Provider)
	assert.Equal(t, providerFieldModel, updated.providerField)
}

func TestUpdateModelSelect_OllamaUsesModelList(t *testing.T) {
	m := NewModel()
	m.primaryProvider = "ollama"
	m.availModels = []llm.ModelInfo{{Name: "qwen2.5:3b", Provider: "ollama"}}
	m.availModelsSource = "ollama"
	m = m.openModelSetup(selectPrimary)
	m.modelCursor = providerOptionIndex("ollama")

	model, _ := m.updateModelSelect(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Equal(t, screenModelSelect, updated.screen)
	assert.Equal(t, modelSelectModels, updated.modelSelectMode)
	assert.Equal(t, "ollama", updated.modelListProvider)
}

func TestUpdateModelSelect_OllamaDoesNotReuseCloudModels(t *testing.T) {
	m := NewModel()
	m.secondaryProvider = "ollama"
	m.availModels = []llm.ModelInfo{{Name: "deepseek-chat", Provider: "deepseek"}}
	m.availModelsSource = "deepseek"
	m = m.openModelSetup(selectSecondary)
	m.modelCursor = providerOptionIndex("ollama")

	model, cmd := m.updateModelSelect(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	updated, ok := model.(Model)
	require.True(t, ok)

	require.NotNil(t, cmd)
	msg := cmd()
	setup, ok := msg.(setupProviderModelsMsg)
	require.True(t, ok)
	assert.Equal(t, "ollama", setup.provider)
	assert.Equal(t, screenModelSelect, updated.screen)
	assert.Equal(t, modelSelectProviders, updated.modelSelectMode)
}

func TestUpdateModelSelect_PrimaryLocalSelectionAdvancesToSecondarySetup(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	cfg := config.DefaultConfig()
	config.Set(cfg)

	m := NewModel()
	m = m.SetLLMConfig(cfg.LLM)
	m.availModels = []llm.ModelInfo{{Name: "qwen2.5:3b", Provider: "ollama"}}
	m.availModelsSource = "ollama"
	m = m.openLocalModelSelection(selectPrimary, "ollama")

	model, cmd := m.updateModelSelect(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Nil(t, cmd)
	assert.Equal(t, screenModelSelect, updated.screen)
	assert.Equal(t, modelSelectProviders, updated.modelSelectMode)
	assert.Equal(t, selectSecondary, updated.modelSelectPhase)
}

func TestPersistRoleModelSelectionSwitchesPrimaryProvider(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	cfg := config.DefaultConfig()
	cfg.LLM.Provider = "openai"
	cfg.LLM.APIKey = "test-key"
	cfg.LLM.APIKeyEnv = ""
	cfg.LLM.Primary.Provider = "openai"
	cfg.LLM.Primary.Model = "gpt-4.1-mini"
	cfg.LLM.Primary.Endpoint = "https://api.openai.com/v1"
	cfg.LLM.Primary.APIKey = "test-key"
	config.Set(cfg)

	m := NewModel()
	m = m.SetLLMConfig(cfg.LLM)

	updated, err := m.persistRoleModelSelection(selectPrimary, "ollama", "qwen2.5:3b")
	require.NoError(t, err)

	assert.Equal(t, "ollama", updated.primaryProvider)
	assert.Equal(t, "qwen2.5:3b", updated.selectedPrimary)
	assert.NotNil(t, updated.llmProvider)
}

func TestPersistProviderConfig_PrimaryCloudAdvancesToSecondarySetup(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	cfg := config.DefaultConfig()
	config.Set(cfg)

	m := NewModel()
	m = m.SetLLMConfig(cfg.LLM)
	m = m.openProviderConfig(selectPrimary)
	m.providerDraft.Provider = "deepseek"
	m.providerDraft.Model = "deepseek-chat"
	m.providerDraft.Endpoint = "https://api.deepseek.com"
	m.providerDraft.APIKey = "secret"

	model, cmd := m.persistProviderConfig()
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Nil(t, cmd)
	assert.Equal(t, screenModelSelect, updated.screen)
	assert.Equal(t, modelSelectProviders, updated.modelSelectMode)
	assert.Equal(t, selectSecondary, updated.modelSelectPhase)
}

func TestOpenProviderConfig_SecondaryKeepsOwnProviderWhenModelEmpty(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.Primary.Provider = "deepseek"
	cfg.LLM.Primary.Model = "deepseek-chat"
	cfg.LLM.Secondary.Provider = "ollama"
	cfg.LLM.Secondary.Model = ""

	m := NewModel()
	m = m.SetLLMConfig(cfg.LLM)
	m = m.openProviderConfig(selectSecondary)

	assert.Equal(t, "ollama", m.providerDraft.Provider)
}

func TestProviderModelsMsgFailureOpensAISetup(t *testing.T) {
	m := NewModel()
	m.screen = screenLoading
	m.primaryProvider = "ollama"
	m.selectedPrimary = "qwen2.5:3b"
	m.llmProvider = ollama.NewClient("http://localhost:11434", "qwen2.5:3b")

	model, cmd := m.Update(providerModelsMsg{available: true})
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Nil(t, cmd)
	assert.Equal(t, screenModelSelect, updated.screen)
	assert.Equal(t, modelSelectProviders, updated.modelSelectMode)
}

func TestGitStateWhileLoadingWaitsForProviderProbe(t *testing.T) {
	m := NewModel()
	m.screen = screenLoading
	m.primaryProvider = "ollama"
	m.selectedPrimary = "qwen2.5:3b"
	m.llmProvider = ollama.NewClient("http://localhost:11434", "qwen2.5:3b")

	model, cmd := m.Update(gitStateMsg{state: &status.GitState{}})
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Nil(t, cmd)
	assert.Equal(t, screenLoading, updated.screen)
}

func TestGitStateWhileLoadingWithoutProviderOpensAISetup(t *testing.T) {
	m := NewModel()
	m.screen = screenLoading
	m.primaryProvider = "openai"
	m.selectedPrimary = ""

	model, cmd := m.Update(gitStateMsg{state: &status.GitState{}})
	updated, ok := model.(Model)
	require.True(t, ok)

	assert.Nil(t, cmd)
	assert.Equal(t, screenModelSelect, updated.screen)
	assert.Equal(t, modelSelectProviders, updated.modelSelectMode)
}

func TestUpdateWorkflowSelectWrapsFromTailToHead(t *testing.T) {
	m := NewModel()
	m.screen = screenWorkflowSelect
	m.workflows = []workflowDefinition{
		{ID: "a", Label: "A"},
		{ID: "b", Label: "B"},
		{ID: "c", Label: "C"},
	}
	m.workflowCursor = len(m.workflows) - 1

	model, _ := m.updateWorkflowSelect(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	updated := model.(Model)

	assert.Equal(t, 0, updated.workflowCursor)
	assert.Equal(t, 0, updated.workflowScroll)
}

func TestUpdateMainTypingGoesToInlineComposer(t *testing.T) {
	m := NewModel()
	m.screen = screenMain

	model, _ := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "部"}))
	updated := model.(Model)

	assert.Equal(t, "部", updated.composerInput)
	assert.Equal(t, 1, updated.composerCursor)
}
