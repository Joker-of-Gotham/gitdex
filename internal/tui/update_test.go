package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/stretchr/testify/assert"
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
	updated := model.(Model)

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
	updated := model.(Model)

	assert.Equal(t, "https://github.com/user/repo.git", updated.inputValues[0])
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
	updated := model.(Model)

	assert.NotNil(t, cmd)
	assert.Equal(t, 1, updated.suggIdx)
	assert.Equal(t, git.ExecDone, updated.suggExecState[0])
	assert.Equal(t, "Review advisory", updated.lastCommand.Title)
	assert.Equal(t, "advisory viewed", updated.lastCommand.Status)
}
