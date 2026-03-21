package views_test

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/tui/views"
)

func TestNewPlansView(t *testing.T) {
	v := views.NewPlansView(makeTheme())
	if v == nil {
		t.Fatal("NewPlansView should return non-nil")
	}
	if v.ID() != views.ViewPlans {
		t.Errorf("ID(): got %v, want ViewPlans", v.ID())
	}
}

func TestPlansView_Render_Empty(t *testing.T) {
	v := views.NewPlansView(makeTheme())
	v.SetSize(80, 20)
	out := v.Render()
	if out == "" {
		t.Error("Render() should not return empty")
	}
	if !strings.Contains(out, "没有活动的执行计划") && !strings.Contains(out, "Select a repository to derive a live stabilization plan") {
		t.Error("Render() empty state should show plans empty state text")
	}
}
