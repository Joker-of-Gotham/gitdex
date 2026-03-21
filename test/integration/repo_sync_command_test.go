package integration

import (
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestRepoCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	repoCmd, _, err := root.Find([]string{"repo"})
	if err != nil {
		t.Fatalf("repo command not found: %v", err)
	}

	subs := []string{"inspect", "clone", "sync"}
	for _, name := range subs {
		found := false
		for _, c := range repoCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'repo'", name)
		}
	}
}

func TestRepoSyncRequiresFlag(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"repo", "sync"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when neither --preview nor --execute is specified")
	}
}

func TestRepoSyncHasPreviewFlag(t *testing.T) {
	root := command.NewRootCommand()
	syncCmd, _, _ := root.Find([]string{"repo", "sync"})
	if syncCmd == nil {
		t.Fatal("repo sync command not found")
	}
	if syncCmd.Flags().Lookup("preview") == nil {
		t.Error("--preview flag not found")
	}
	if syncCmd.Flags().Lookup("execute") == nil {
		t.Error("--execute flag not found")
	}
}
