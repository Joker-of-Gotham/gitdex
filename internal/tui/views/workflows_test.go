package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestWorkflowsView_RenderAndDispatchPrompt(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewWorkflowsView(&th)
	v.SetSize(140, 40)
	v.SetRuns([]WorkflowRunEntry{
		{RunID: 321, WorkflowID: 123, Name: "CI", Status: "completed", Conclusion: "success", Branch: "main", Event: "push"},
	})

	out := v.Render()
	if !strings.Contains(out, "Workflows") {
		t.Fatalf("render missing title: %q", out)
	}

	_, cmd := v.Update(tea.KeyPressMsg{Code: 'r'})
	if cmd == nil {
		t.Fatal("opening workflow dispatch prompt should return a focus command")
	}

	_, submit := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if submit == nil {
		t.Fatal("submitting workflow dispatch prompt should return a request command")
	}
	msg := submit()
	req, ok := msg.(RequestWorkflowDispatchMsg)
	if !ok {
		t.Fatalf("submit returned %T, want RequestWorkflowDispatchMsg", msg)
	}
	if req.WorkflowID != 123 || req.Ref != "main" {
		t.Fatalf("unexpected request: %#v", req)
	}
}

func TestWorkflowsView_RequestRunActions(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewWorkflowsView(&th)
	v.SetSize(140, 40)
	v.SetRuns([]WorkflowRunEntry{
		{RunID: 456, WorkflowID: 654, Name: "Deploy", Status: "in_progress", Branch: "release"},
	})

	_, rerunCmd := v.Update(tea.KeyPressMsg{Code: 'R'})
	if rerunCmd == nil {
		t.Fatal("rerun should return a request command")
	}
	rerunMsg := rerunCmd()
	rerunReq, ok := rerunMsg.(RequestWorkflowActionMsg)
	if !ok {
		t.Fatalf("rerun returned %T, want RequestWorkflowActionMsg", rerunMsg)
	}
	if rerunReq.RunID != 456 || rerunReq.Kind != WorkflowActionRerun {
		t.Fatalf("unexpected rerun request: %#v", rerunReq)
	}

	_, cancelCmd := v.Update(tea.KeyPressMsg{Code: 'x'})
	if cancelCmd == nil {
		t.Fatal("cancel should return a request command")
	}
	cancelMsg := cancelCmd()
	cancelReq, ok := cancelMsg.(RequestWorkflowActionMsg)
	if !ok {
		t.Fatalf("cancel returned %T, want RequestWorkflowActionMsg", cancelMsg)
	}
	if cancelReq.RunID != 456 || cancelReq.Kind != WorkflowActionCancel {
		t.Fatalf("unexpected cancel request: %#v", cancelReq)
	}
}
