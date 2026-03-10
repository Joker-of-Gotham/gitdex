package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateLanguageSelect_PersistsLanguageAndReturns(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	require.NoError(t, i18n.Init("en"))
	defer func() {
		_ = i18n.Init("en")
	}()
	config.Set(config.DefaultConfig())

	m := NewModel()
	m.screen = screenLanguageSelect
	m.languageReturnTo = screenMain
	m.languageCursor = m.languageCursorFor("zh")

	model, _ := m.updateLanguageSelect(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	updated := model.(Model)

	assert.Equal(t, screenMain, updated.screen)
	assert.Equal(t, "zh", i18n.Lang())

	cfgPath, err := config.GlobalConfigPath()
	require.NoError(t, err)
	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "language: zh")
}
