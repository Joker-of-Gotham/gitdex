package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestCommitLogView_RenderAndDetailRequest(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewCommitLogView(&th)
	v.SetSize(140, 40)
	v.SetCommits([]CommitEntry{
		{Hash: "abcdef1234567890", Author: "alice", Date: "2026-03-21", Message: "initial"},
	})

	out := v.Render()
	if !strings.Contains(out, "Commits") {
		t.Fatalf("render missing title: %q", out)
	}

	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !v.detail {
		t.Fatal("enter should open commit detail")
	}
	if cmd == nil {
		t.Fatal("enter should request commit detail")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("detail request type = %T", msg)
	}
	if len(batch) != 2 {
		t.Fatalf("detail request batch len = %d", len(batch))
	}
	first := batch[0]()
	if _, ok := first.(CommitSelectedMsg); !ok {
		t.Fatalf("first batch msg = %T", first)
	}
	second := batch[1]()
	if _, ok := second.(RequestCommitDetailMsg); !ok {
		t.Fatalf("second batch msg = %T", second)
	}
}

func TestCommitLogView_CommitDetailMsg(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewCommitLogView(&th)
	v.SetSize(140, 40)
	v.Update(CommitDetailMsg{
		Hash:    "abcdef1234567890",
		Content: "commit abcdef1234567890\nAuthor: Alice\n\nmessage",
	})
	out := v.renderDetail(80)
	if !strings.Contains(out, "commit abcdef1234567890") {
		t.Fatalf("detail render = %q", out)
	}
}

func TestCommitLogView_CommitActionPrompt(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewCommitLogView(&th)
	v.SetSize(140, 40)
	v.SetEditable(true)
	v.SetCommits([]CommitEntry{
		{Hash: "abcdef1234567890", Author: "alice", Date: "2026-03-21", Message: "initial"},
	})

	_, cmd := v.Update(tea.KeyPressMsg{Code: 'p'})
	if cmd == nil {
		t.Fatal("opening commit prompt should return a focus command")
	}

	_, submit := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if submit == nil {
		t.Fatal("submitting commit prompt should return a request command")
	}
	msg := submit()
	req, ok := msg.(RequestCommitActionMsg)
	if !ok {
		t.Fatalf("submit returned %T, want RequestCommitActionMsg", msg)
	}
	if req.Kind != CommitActionCherryPick || req.Hash != "abcdef1234567890" {
		t.Fatalf("unexpected request: %#v", req)
	}
}
