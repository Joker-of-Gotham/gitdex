package components_test

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/tui/components"
)

func TestNewSpinner(t *testing.T) {
	s := components.NewSpinner(makeTheme(), "loading")
	if s == nil {
		t.Fatal("NewSpinner should return non-nil")
	}
}

func TestSpinner_Render_WhenNotVisible(t *testing.T) {
	s := components.NewSpinner(makeTheme(), "loading")
	s.SetVisible(false)
	out := s.Render()
	if out != "" {
		t.Errorf("Render when not visible should return empty, got %q", out)
	}
}

func TestSpinner_Render_WhenVisible(t *testing.T) {
	s := components.NewSpinner(makeTheme(), "loading")
	s.SetVisible(true)
	out := s.Render()
	if out == "" {
		t.Error("Render when visible should return non-empty")
	}
	if !strings.Contains(out, "loading") {
		t.Error("Render should contain label")
	}
}
