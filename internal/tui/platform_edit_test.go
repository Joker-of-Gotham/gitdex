package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	platformruntime "github.com/Joker-of-Gotham/gitdex/internal/platform/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMain_EditPlatformShortcutOpensEditor(t *testing.T) {
	m := NewModel()
	m.screen = screenMain
	m.lastPlatformOp = &git.PlatformExecInfo{
		CapabilityID: "pages",
		Flow:         "inspect",
		Query:        map[string]string{"view": "latest_build"},
	}

	model, cmd := m.updateMain(tea.KeyPressMsg(tea.Key{Text: "e"}))
	updated := model.(Model)

	require.Nil(t, cmd)
	assert.Equal(t, screenPlatformEdit, updated.screen)
	assert.Contains(t, updated.platformEdit, `"capability_id": "pages"`)
	assert.Contains(t, updated.platformEdit, `"view": "latest_build"`)
}

func TestUpdatePlatformEditSubmitsEditedRequest(t *testing.T) {
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
	m = m.openPlatformEdit(&git.PlatformExecInfo{
		CapabilityID: "pages",
		Flow:         "inspect",
		Query:        map[string]string{"view": "latest_build"},
	})

	model, cmd := m.updatePlatformEdit(tea.KeyPressMsg(tea.Key{Text: "ctrl+s"}))
	updated := model.(Model)

	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = updated.Update(msg)
	updated = model.(Model)

	assert.Equal(t, screenMain, updated.screen)
	assert.Equal(t, resultKindPlatformAdmin, updated.lastCommand.ResultKind)
	assert.Equal(t, "pages", updated.lastCommand.PlatformCapability)
	assert.Equal(t, "inspect", updated.lastCommand.PlatformFlow)
	require.NotNil(t, updated.lastPlatformOp)
	assert.Equal(t, "pages", updated.lastPlatformOp.CapabilityID)
}
