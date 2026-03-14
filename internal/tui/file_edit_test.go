package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlashEditOpensFileEditor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "README.md")

	m := NewModel()
	m.screen = screenMain
	m.lastCommand = commandTrace{
		ResultKind:    resultKindFileWrite,
		FilePath:      path,
		FileOperation: "update",
		BeforeContent: "old",
		AfterContent:  "new",
	}

	m.composerInput = "/edit"
	m.composerCursor = len([]rune(m.composerInput))

	model, cmd := m.submitInlineGoal()
	updated := model.(Model)

	require.Nil(t, cmd)
	assert.Equal(t, screenFileEdit, updated.screen)
	assert.Equal(t, path, updated.fileEditReq.Path)
	assert.Equal(t, "new", updated.fileEdit)
}

func TestUpdateFileEditSubmitsEditedContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

	m := NewModel()
	m = m.openFileEdit(&git.FileWriteInfo{
		Path:      path,
		Operation: "update",
		Content:   "new",
		Backup:    true,
	})
	m.fileEdit = "edited"
	m.fileCursor = len([]rune(m.fileEdit))

	model, cmd := m.updateFileEdit(tea.KeyPressMsg(tea.Key{Text: "ctrl+s"}))
	updated := model.(Model)

	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = updated.Update(msg)
	updated = model.(Model)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "edited", string(data))
	assert.Equal(t, screenMain, updated.screen)
	assert.Equal(t, resultKindFileWrite, updated.lastCommand.ResultKind)
	assert.Equal(t, "edited", updated.lastCommand.AfterContent)
	assert.Equal(t, "old", updated.lastCommand.BeforeContent)
}
