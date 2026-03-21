package components_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/components"
)

func TestNewCmdPalette(t *testing.T) {
	cp := components.NewCmdPalette(makeTheme())
	if cp == nil {
		t.Fatal("NewCmdPalette() should return non-nil")
	}
	if cp.IsVisible() {
		t.Error("NewCmdPalette should create hidden palette")
	}
}

func TestCmdPalette_Show_Hide(t *testing.T) {
	cp := components.NewCmdPalette(makeTheme())
	cp.Show()
	if !cp.IsVisible() {
		t.Error("Show() should make palette visible")
	}
	cp.Hide()
	if cp.IsVisible() {
		t.Error("Hide() should hide palette")
	}
}

func TestCmdPalette_AddItem(t *testing.T) {
	cp := components.NewCmdPalette(makeTheme())
	cp.AddItem(components.PaletteItem{Label: "Test", Description: "Test action", Shortcut: "F1", Action: nil})
	cp.Show()
	out := cp.Render(80, 24)
	if out == "" {
		t.Error("Render with items should return non-empty")
	}
	if !cp.IsVisible() {
		t.Error("Show() should make visible")
	}
	_ = cp.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cp.IsVisible() {
		t.Error("Escape should hide palette")
	}
}
