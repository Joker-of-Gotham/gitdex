package views_test

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/views"
)

func TestNewCockpitView(t *testing.T) {
	v := views.NewCockpitView(makeTheme())
	if v == nil {
		t.Fatal("NewCockpitView should return non-nil")
	}
	if v.ID() != views.ViewCockpit {
		t.Errorf("ID(): got %v, want ViewCockpit", v.ID())
	}
	if v.Title() != "Cockpit" {
		t.Errorf("Title(): got %q, want Cockpit", v.Title())
	}
}

func TestCockpitView_Render_NilSummary(t *testing.T) {
	v := views.NewCockpitView(makeTheme())
	v.SetSize(80, 20)
	out := v.Render()
	if out == "" {
		t.Error("Render() should not return empty")
	}
	if !strings.Contains(out, "gitdex scan") {
		t.Error("Render() with nil summary should show empty state text about gitdex scan")
	}
}

func TestCockpitView_Render_WithSummary(t *testing.T) {
	v := views.NewCockpitView(makeTheme())
	v.SetSize(80, 20)
	v.SetSummary(&repo.RepoSummary{
		Owner: "test-owner",
		Repo:  "test-repo",
	})
	out := v.Render()
	if out == "" {
		t.Error("Render() should not return empty")
	}
	if !strings.Contains(out, "test-owner") || !strings.Contains(out, "test-repo") {
		t.Error("Render() with summary should show repo info")
	}
}
