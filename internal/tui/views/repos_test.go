package views

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestReposView_CRequestsCloneForRemoteRepo(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewReposView(&th)
	v.SetSize(120, 30)
	v.SetItems([]RepoListItem{
		{Owner: "owner", Name: "repo", FullName: "owner/repo", IsLocal: false},
	})

	_, cmd := v.Update(tea.KeyPressMsg{Text: "c"})
	if cmd == nil {
		t.Fatal("expected clone request command")
	}
	msg := cmd()
	request, ok := msg.(CloneRepoRequestMsg)
	if !ok {
		t.Fatalf("command returned %T, want CloneRepoRequestMsg", msg)
	}
	if request.Repo.FullName != "owner/repo" {
		t.Fatalf("request repo = %q, want owner/repo", request.Repo.FullName)
	}
}
