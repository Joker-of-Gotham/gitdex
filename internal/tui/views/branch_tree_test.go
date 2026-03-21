package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestBranchTreeView_RenderAndCheckoutRequest(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewBranchTreeView(&th)
	v.SetSize(120, 30)
	v.SetEditable(true)
	v.SetBranches([]BranchEntry{
		{Name: "main", IsCurrent: true},
		{Name: "feature/demo", Upstream: "origin/feature/demo", SHA: "abcdef1234567890"},
	})

	v.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd := v.Update(tea.KeyPressMsg{Code: 'c'})
	if cmd == nil {
		t.Fatal("c should request branch checkout")
	}
	msg := cmd()
	req, ok := msg.(RequestBranchCheckoutMsg)
	if !ok {
		t.Fatalf("checkout request type = %T", msg)
	}
	if req.Name != "feature/demo" {
		t.Fatalf("checkout branch = %q", req.Name)
	}
}

func TestBranchTreeView_DetailAndResult(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewBranchTreeView(&th)
	v.SetSize(120, 30)
	v.SetBranches([]BranchEntry{
		{Name: "main", SHA: "abcdef1234567890", Upstream: "origin/main", IsCurrent: true},
	})

	v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !v.detail {
		t.Fatal("enter should toggle branch detail")
	}

	out := v.Render()
	if !strings.Contains(out, "Inspector detail active") {
		t.Fatalf("detail render = %q", out)
	}

	v.Update(BranchCheckoutResultMsg{Name: "main"})
	if !strings.Contains(v.statusLine, "Checked out") {
		t.Fatalf("statusLine = %q", v.statusLine)
	}
}

func TestBranchTreeView_BranchActionPrompt(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewBranchTreeView(&th)
	v.SetSize(120, 30)
	v.SetEditable(true)
	v.SetBranches([]BranchEntry{
		{Name: "main", IsCurrent: true},
		{Name: "feature/demo", Upstream: "origin/feature/demo", SHA: "abcdef1234567890"},
	})

	_, cmd := v.Update(tea.KeyPressMsg{Code: 'n'})
	if cmd == nil {
		t.Fatal("opening branch create prompt should return a focus command")
	}
	v.prompt.SetValue("feature/new main")
	_, submit := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if submit == nil {
		t.Fatal("submitting branch create prompt should return a request command")
	}
	msg := submit()
	req, ok := msg.(RequestBranchActionMsg)
	if !ok {
		t.Fatalf("submit returned %T, want RequestBranchActionMsg", msg)
	}
	if req.Kind != BranchActionCreate || req.Name != "feature/new" || req.Target != "main" {
		t.Fatalf("unexpected request: %#v", req)
	}
}
