package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestIssuesView_ID(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewIssuesView(&th)
	if v.ID() != ViewIssues {
		t.Errorf("ID() = %q, want %q", v.ID(), ViewIssues)
	}
}

func TestIssuesView_SetItems(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewIssuesView(&th)
	items := []repo.IssueSummary{
		{Number: 1, Title: "Bug", Author: "alice", State: "OPEN"},
		{Number: 2, Title: "Feature", Author: "bob", State: "CLOSED"},
	}
	v.SetItems(items)
	if len(v.items) != 2 {
		t.Errorf("items = %d, want 2", len(v.items))
	}
}

func TestIssuesView_Navigation(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewIssuesView(&th)
	v.SetSize(120, 40)
	v.SetItems([]repo.IssueSummary{
		{Number: 1}, {Number: 2}, {Number: 3},
	})

	v.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if v.cursor != 1 {
		t.Errorf("cursor = %d, want 1", v.cursor)
	}
}

func TestIssuesView_DetailToggle(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewIssuesView(&th)
	v.SetSize(120, 40)
	v.SetItems([]repo.IssueSummary{
		{Number: 1, Title: "Test", Author: "dev", State: "OPEN", Comments: 3, Labels: []string{"bug"}},
	})

	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !v.detail {
		t.Error("Enter should show detail")
	}
	if cmd == nil {
		t.Fatal("Enter should request issue detail")
	}
}

func TestIssuesView_IssueDetailMsg(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewIssuesView(&th)
	v.SetSize(140, 40)
	v.Update(IssueDetailMsg{Detail: IssueDetail{
		Number:    10,
		Title:     "My Issue",
		Author:    "dev",
		State:     "open",
		Body:      "Issue detail",
		Labels:    []string{"bug"},
		Assignees: []string{"ops"},
	}})
	out := v.detailView.Render()
	if !strings.Contains(out, "#10") {
		t.Fatalf("detail render missing issue number: %q", out)
	}
}

func TestIssuesView_IssueDetailActionPrompt(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewIssuesView(&th)
	v.SetSize(140, 40)
	v.detail = true
	v.detailView.SetDetail(&IssueDetail{Number: 10, Title: "My Issue", Author: "dev", State: "open"})

	_, cmd := v.Update(tea.KeyPressMsg{Code: 'c'})
	if cmd == nil {
		t.Fatal("opening issue comment prompt should return a focus command")
	}
	if !v.detailView.PromptActive() {
		t.Fatal("issue comment prompt should be active")
	}

	v.detailView.prompt.SetValue("tracking this")
	_, submit := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if submit == nil {
		t.Fatal("submitting issue comment should return a request command")
	}
	msg := submit()
	req, ok := msg.(RequestIssueActionMsg)
	if !ok {
		t.Fatalf("submit returned %T, want RequestIssueActionMsg", msg)
	}
	if req.Kind != IssueActionComment || req.Number != 10 || req.Body != "tracking this" {
		t.Fatalf("unexpected request: %#v", req)
	}
}

func TestIssuesView_Render(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewIssuesView(&th)
	v.SetSize(120, 40)
	v.SetItems([]repo.IssueSummary{
		{Number: 10, Title: "My Issue", Author: "dev", State: "OPEN"},
	})

	output := v.Render()
	if !strings.Contains(output, "Issues") {
		t.Error("should contain title")
	}
	if !strings.Contains(output, "#10") {
		t.Error("should contain issue number")
	}
}

func TestIssuesView_RenderEmpty(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewIssuesView(&th)
	v.SetSize(120, 40)

	output := v.Render()
	if !strings.Contains(output, "No issues") {
		t.Error("should show empty message")
	}
}

func TestIssuesView_DataMsg(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewIssuesView(&th)
	v.SetSize(120, 40)

	v.Update(IssuesDataMsg{Items: []repo.IssueSummary{
		{Number: 1, State: "OPEN"},
		{Number: 2, State: "CLOSED"},
	}})
	if len(v.items) != 2 {
		t.Errorf("items = %d, want 2", len(v.items))
	}
}
