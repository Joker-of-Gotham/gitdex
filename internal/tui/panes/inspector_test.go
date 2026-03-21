package panes_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/panes"
)

func TestNewInspectorPane(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	if p == nil {
		t.Fatal("NewInspectorPane should return non-nil")
	}
}

func TestInspectorPane_Toggle(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	if !p.IsVisible() {
		t.Error("inspector should be visible by default")
	}
	p.Toggle()
	if p.IsVisible() {
		t.Error("Toggle() should flip visibility to false")
	}
	p.Toggle()
	if !p.IsVisible() {
		t.Error("Toggle() again should restore to true")
	}
}

func TestInspectorPane_SetMode(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	p.SetSize(60, 20)
	p.SetMode(panes.ModeEvidence)
	out := p.View()
	if !strings.Contains(out, "Evidence") {
		t.Error("SetMode(ModeEvidence) should change View() output")
	}
}

func TestInspectorPane_View(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	p.SetSize(60, 20)
	out := p.View()
	if out == "" {
		t.Error("View() when visible should return non-empty string")
	}
	if !strings.Contains(out, "Repository") {
		t.Error("View() should contain default mode name 'Repository'")
	}
}

func TestInspectorPane_Update_ModeSwitch(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	p.SetFocused(true)
	p.SetSize(60, 20)

	p.Update(tea.KeyPressMsg{Code: '2'})
	out := p.View()
	if !strings.Contains(out, "Risk") {
		t.Error("key '2' should switch to Risk mode")
	}

	p.Update(tea.KeyPressMsg{Code: '3'})
	out = p.View()
	if !strings.Contains(out, "Evidence") {
		t.Error("key '3' should switch to Evidence mode")
	}

	p.Update(tea.KeyPressMsg{Code: '4'})
	out = p.View()
	if !strings.Contains(out, "Audit") {
		t.Error("key '4' should switch to Audit mode")
	}

	p.Update(tea.KeyPressMsg{Code: '1'})
	out = p.View()
	if !strings.Contains(out, "Repository") {
		t.Error("key '1' should switch back to Repository mode")
	}
}

func TestInspectorPane_Update_NotFocused(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	p.SetFocused(false)
	p.SetSize(60, 20)

	cmd := p.Update(tea.KeyPressMsg{Code: '2'})
	if cmd != nil {
		t.Error("Update when not focused should return nil cmd")
	}
	out := p.View()
	if !strings.Contains(out, "Repository") {
		t.Error("mode should not change when not focused")
	}
}

func TestInspectorPane_SetStyles(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	newStyles := makeStyles()
	p.SetStyles(newStyles)
}

func TestInspectorPane_ViewHidden(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	p.Toggle()
	out := p.View()
	if out != "" {
		t.Error("View() when hidden should return empty string")
	}
}

func TestInspectorPane_SetRepoDetail(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	p.SetSize(60, 30)
	p.SetRepoDetail(panes.RepoDetailData{
		Name:        "test-org/test-repo",
		Description: "A test repository",
		Stars:       42,
		Language:    "Go",
		IsLocal:     true,
		LocalPaths:  []string{"/tmp/test-repo"},
	})
	out := p.View()
	if !strings.Contains(out, "test-org/test-repo") {
		t.Error("repo detail should show repo name")
	}
	if !strings.Contains(out, "A test repository") {
		t.Error("repo detail should show description")
	}
	plain := strings.ToLower(out)
	if !strings.Contains(plain, "local clone") && !strings.Contains(plain, "local") {
		t.Error("repo detail should show local status")
	}
}

func TestInspectorPane_EnrichRepoDetail(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	p.SetSize(60, 30)
	p.SetRepoDetail(panes.RepoDetailData{
		Name: "test-org/repo",
	})
	p.EnrichRepoDetail(panes.RepoDetailData{
		Description: "Enriched description",
		Stars:       100,
		License:     "MIT",
		Topics:      []string{"go", "tui"},
	})
	out := p.View()
	if !strings.Contains(out, "Enriched description") {
		t.Error("enriched detail should show updated description")
	}
	if !strings.Contains(out, "MIT") {
		t.Error("enriched detail should show license")
	}
}

func TestInspectorPane_Show(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	p.Toggle()
	if p.IsVisible() {
		t.Error("should be hidden after toggle")
	}
	p.Show()
	if !p.IsVisible() {
		t.Error("Show() should make inspector visible")
	}
}

func TestInspectorPane_DetailModeIgnoresGlobalTabs(t *testing.T) {
	p := panes.NewInspectorPane(makeTheme(), makeStyles())
	p.SetFocused(true)
	p.SetSize(60, 20)
	p.SetCommitDetail(panes.InspectorCommitData{Hash: "abc123", Message: "test"})

	p.Update(tea.KeyPressMsg{Code: '2'})
	out := p.View()
	if strings.Contains(out, "Risk") && !strings.Contains(out, "Context-locked detail surface") {
		t.Fatal("detail mode should not switch to global risk tabs")
	}
	if !strings.Contains(out, "Commit") {
		t.Fatal("detail mode should keep commit detail visible")
	}
}
