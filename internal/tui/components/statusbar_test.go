package components_test

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/tui/components"
	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestNewStatusBar(t *testing.T) {
	tk := theme.NewTheme(true)
	sb := components.NewStatusBar(&tk)
	if sb == nil {
		t.Fatal("NewStatusBar() should return non-nil")
	}
}

func TestStatusBar_SetMode_SetBranch_SetThemeName(t *testing.T) {
	tk := theme.NewTheme(true)
	sb := components.NewStatusBar(&tk)
	sb.SetMode("INSERT")
	sb.SetBranch("main")
	sb.SetThemeName("dracula")
	out := sb.Render()
	if out == "" {
		t.Error("Render() should return non-empty string")
	}
	if !strings.Contains(out, "main") {
		t.Error("Render() should contain branch name")
	}
	if !strings.Contains(out, "INSERT") {
		t.Error("Render() should contain mode")
	}
}
