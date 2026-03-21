package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestPullsView_ID(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewPullsView(&th)
	if v.ID() != ViewPulls {
		t.Errorf("ID() = %q, want %q", v.ID(), ViewPulls)
	}
	if v.Title() != "Pull Requests" {
		t.Errorf("Title() = %q, want Pull Requests", v.Title())
	}
}

func TestPullsView_SetItems(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewPullsView(&th)
	items := []repo.PullRequestSummary{
		{Number: 1, Title: "Fix bug", Author: "alice"},
		{Number: 2, Title: "Add feature", Author: "bob"},
	}
	v.SetItems(items)
	if len(v.items) != 2 {
		t.Errorf("items count = %d, want 2", len(v.items))
	}
}

func TestPullsView_Navigation(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewPullsView(&th)
	v.SetSize(120, 40)
	v.SetItems([]repo.PullRequestSummary{
		{Number: 1, Title: "PR1", Author: "a"},
		{Number: 2, Title: "PR2", Author: "b"},
		{Number: 3, Title: "PR3", Author: "c"},
	})

	v.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if v.cursor != 1 {
		t.Errorf("cursor = %d after down, want 1", v.cursor)
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if v.cursor != 2 {
		t.Errorf("cursor = %d after 2nd down, want 2", v.cursor)
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if v.cursor != 2 {
		t.Errorf("cursor should not exceed item count, got %d", v.cursor)
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if v.cursor != 1 {
		t.Errorf("cursor = %d after up, want 1", v.cursor)
	}
}

func TestPullsView_DetailToggle(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewPullsView(&th)
	v.SetSize(120, 40)
	v.SetItems([]repo.PullRequestSummary{
		{Number: 1, Title: "Test PR", Author: "alice", Labels: []string{"bug"}, IsDraft: true, NeedsReview: false, StaleDays: 5},
	})

	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !v.detail {
		t.Error("Enter should open detail")
	}
	if cmd == nil {
		t.Fatal("Enter should request PR detail")
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v.detail {
		t.Error("Esc should close detail")
	}
}

func TestPullsView_PRDetailMsg(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewPullsView(&th)
	v.SetSize(140, 40)
	v.Update(PRDetailMsg{Detail: PRDetail{
		Number:  42,
		Title:   "Improve sync",
		Author:  "alice",
		State:   "open",
		Body:    "Longer detail",
		Labels:  []string{"infra"},
		Reviews: []PRReviewItem{{User: "reviewer", State: "APPROVED"}},
	}})
	out := v.detailView.Render()
	if !strings.Contains(out, "#42") {
		t.Fatalf("detail render missing PR number: %q", out)
	}
}

func TestPullsView_PRDetailActionPrompt(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewPullsView(&th)
	v.SetSize(140, 40)
	v.detail = true
	v.detailView.SetDetail(&PRDetail{Number: 42, Title: "Improve sync", Author: "alice", State: "open"})

	_, cmd := v.Update(tea.KeyPressMsg{Code: 'c'})
	if cmd == nil {
		t.Fatal("opening PR comment prompt should return a focus command")
	}
	if !v.detailView.PromptActive() {
		t.Fatal("comment prompt should be active")
	}

	v.detailView.prompt.SetValue("ship it")
	_, submit := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if submit == nil {
		t.Fatal("submitting PR comment should return a request command")
	}
	msg := submit()
	req, ok := msg.(RequestPRActionMsg)
	if !ok {
		t.Fatalf("submit returned %T, want RequestPRActionMsg", msg)
	}
	if req.Kind != PRActionComment || req.Number != 42 || req.Body != "ship it" {
		t.Fatalf("unexpected request: %#v", req)
	}
}

func TestPullsView_Render(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewPullsView(&th)
	v.SetSize(120, 40)
	v.SetItems([]repo.PullRequestSummary{
		{Number: 42, Title: "My PR", Author: "dev", IsDraft: true},
	})

	output := v.Render()
	if !strings.Contains(output, "Pull Requests") {
		t.Error("should contain title")
	}
	if !strings.Contains(output, "#42") {
		t.Error("should contain PR number")
	}
}

func TestPullsView_RenderEmpty(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewPullsView(&th)
	v.SetSize(120, 40)

	output := v.Render()
	if !strings.Contains(output, "No pull requests") {
		t.Error("should show empty message")
	}
}

func TestPullsView_DataMsg(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewPullsView(&th)
	v.SetSize(120, 40)

	v.Update(PullsDataMsg{Items: []repo.PullRequestSummary{
		{Number: 1, Title: "Test"},
	}})
	if len(v.items) != 1 {
		t.Errorf("items = %d after DataMsg, want 1", len(v.items))
	}
}
