package components_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/components"
)

func TestNewModal(t *testing.T) {
	m := components.NewModal(makeTheme(), "Test")
	if m == nil {
		t.Fatal("NewModal() should return non-nil")
	}
	if m.IsVisible() {
		t.Error("NewModal should create hidden modal")
	}
}

func TestModal_Show_Hide(t *testing.T) {
	m := components.NewModal(makeTheme(), "Test")
	m.Show("content")
	if !m.IsVisible() {
		t.Error("Show() should make modal visible")
	}
	m.Hide()
	if m.IsVisible() {
		t.Error("Hide() should hide modal")
	}
}

func TestModal_Render_WhenHidden(t *testing.T) {
	m := components.NewModal(makeTheme(), "Test")
	out := m.Render(100, 50)
	if out != "" {
		t.Error("Render when hidden should return empty")
	}
}

func TestModal_Update_EscapeHides(t *testing.T) {
	m := components.NewModal(makeTheme(), "Test")
	m.Show("content")
	if !m.IsVisible() {
		t.Fatal("modal should be visible after Show")
	}
	_ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.IsVisible() {
		t.Error("Update(Escape) should hide modal")
	}
}
