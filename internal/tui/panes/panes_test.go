package panes_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/panes"
	"github.com/your-org/gitdex/internal/tui/theme"
)

func makeTheme() *theme.Theme {
	t := theme.NewTheme(true)
	return &t
}

func makeStyles() theme.Styles {
	t := theme.NewTheme(true)
	return theme.NewStyles(t)
}

func TestNewNavPane(t *testing.T) {
	p := panes.NewNavPane(makeTheme(), makeStyles(), nil)
	p.SetSize(40, 15)
	if len(p.SelectedItem().Label) == 0 && len(p.SelectedItem().Path) == 0 {
		// SelectedItem may return first item
	}
	v := p.View()
	if v == "" {
		t.Error("NavPane should render non-empty content")
	}
}

func TestNavPane_SetSize(t *testing.T) {
	p := panes.NewNavPane(makeTheme(), makeStyles(), nil)
	p.SetSize(40, 20)
	v := p.View()
	if v == "" {
		t.Error("View after SetSize should be non-empty")
	}
}

func TestNavPane_SetFocused_Focused(t *testing.T) {
	p := panes.NewNavPane(makeTheme(), makeStyles(), nil)
	p.SetFocused(true)
	if !p.Focused() {
		t.Error("Focused() should be true after SetFocused(true)")
	}
	p.SetFocused(false)
	if p.Focused() {
		t.Error("Focused() should be false after SetFocused(false)")
	}
}

func TestNavPane_SelectedItem(t *testing.T) {
	p := panes.NewNavPane(makeTheme(), makeStyles(), nil)
	item := p.SelectedItem()
	if item.Label == "" {
		t.Error("SelectedItem() initially should return a non-empty item")
	}
}

func TestNavPane_View(t *testing.T) {
	p := panes.NewNavPane(makeTheme(), makeStyles(), nil)
	p.SetSize(50, 15)
	v := p.View()
	if v == "" {
		t.Error("View() should return non-empty string")
	}
	if v == "" {
		t.Error("View() should render non-empty content")
	}
}

func TestNewStatusPane(t *testing.T) {
	p := panes.NewStatusPane(makeStyles())
	if p.Summary() != nil {
		t.Error("NewStatusPane() should start with nil summary")
	}
}

func TestStatusPane_Update_Summary(t *testing.T) {
	p := panes.NewStatusPane(makeStyles())
	summary := &repo.RepoSummary{
		Owner:        "test-owner",
		Repo:         "test-repo",
		OverallLabel: repo.Healthy,
	}
	updated, _ := p.Update(panes.StatusUpdateMsg{Summary: summary})
	p2 := updated
	if p2.Summary() == nil {
		t.Error("Summary() should return non-nil after StatusUpdateMsg")
	}
	if p2.Summary().Owner != "test-owner" || p2.Summary().Repo != "test-repo" {
		t.Errorf("Summary(): got %s/%s, want test-owner/test-repo",
			p2.Summary().Owner, p2.Summary().Repo)
	}
}

func TestStatusPane_View_NilSummary(t *testing.T) {
	p := panes.NewStatusPane(makeStyles())
	p.SetSize(50, 15)
	v := p.View()
	if !strings.Contains(v, "No repository data loaded") {
		t.Errorf("View with nil summary should show placeholder, got %q", v)
	}
}

func TestStatusPane_View_ValidSummary(t *testing.T) {
	p := panes.NewStatusPane(makeStyles())
	summary := &repo.RepoSummary{
		Owner:        "my-org",
		Repo:         "my-repo",
		OverallLabel: repo.Healthy,
	}
	updated, _ := p.Update(panes.StatusUpdateMsg{Summary: summary})
	p2 := updated
	p2.SetSize(50, 15)
	v := p2.View()
	if !strings.Contains(v, "my-org") || !strings.Contains(v, "my-repo") {
		t.Errorf("View with valid summary should show owner/repo, got %q", v)
	}
}

func TestNewRiskPane(t *testing.T) {
	p := panes.NewRiskPane(makeStyles())
	p.SetSize(40, 15)
	v := p.View()
	if v == "" {
		t.Error("RiskPane View() should return non-empty string")
	}
	// When summary is nil, shows "No data"
	if !strings.Contains(v, "No data") {
		t.Errorf("RiskPane View with nil summary should show No data, got %q", v)
	}
}

func TestRiskPane_View(t *testing.T) {
	p := panes.NewRiskPane(makeStyles())
	p.SetSize(50, 15)
	v := p.View()
	if v == "" {
		t.Error("RiskPane View() should return non-empty string")
	}
}

func TestNewInputPane(t *testing.T) {
	p := panes.NewInputPane(makeStyles())
	if p.Value() != "" {
		t.Errorf("NewInputPane Value: got %q, want empty", p.Value())
	}
}

func TestInputPane_SetFocused_Value(t *testing.T) {
	p := panes.NewInputPane(makeStyles())
	p.SetFocused(true)
	p.SetSize(50, 10)

	// Type "hi" - InputPane uses len(k)==1 for single chars
	for _, ch := range "hi" {
		updated, _ := p.Update(tea.KeyPressMsg{Code: ch})
		p = updated
	}

	if p.Value() != "hi" {
		t.Errorf("InputPane Value after typing hi: got %q", p.Value())
	}
}
