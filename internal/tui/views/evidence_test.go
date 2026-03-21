package views_test

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/tui/views"
)

func TestNewEvidenceView(t *testing.T) {
	v := views.NewEvidenceView(makeTheme())
	if v == nil {
		t.Fatal("NewEvidenceView should return non-nil")
	}
	if v.ID() != views.ViewEvidence {
		t.Errorf("ID(): got %v, want ViewEvidence", v.ID())
	}
}

func TestEvidenceView_Render_Empty(t *testing.T) {
	v := views.NewEvidenceView(makeTheme())
	v.SetSize(80, 20)
	out := v.Render()
	if out == "" {
		t.Error("Render() should not return empty")
	}
	if !strings.Contains(out, "No execution evidence yet") {
		t.Error("Render() empty state should show evidence empty state text")
	}
}
