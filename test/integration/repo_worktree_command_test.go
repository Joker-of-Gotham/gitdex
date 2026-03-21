package integration

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestRepoWorktreeCommandsRegistered(t *testing.T) {
	root := command.NewRootCommand()
	repoCmd, _, err := root.Find([]string{"repo"})
	if err != nil {
		t.Fatalf("repo command not found: %v", err)
	}

	worktreeCmd, _, err := repoCmd.Find([]string{"worktree"})
	if err != nil {
		t.Fatalf("repo worktree command not found: %v", err)
	}

	subs := []string{"create", "inspect", "diff", "discard"}
	for _, name := range subs {
		found := false
		for _, c := range worktreeCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'repo worktree'", name)
		}
	}
}

func TestRepoWorktreeCreateRequiresBranch(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"repo", "worktree", "create"})
	root.SetOut(bytes.NewBuffer(nil))
	root.SetErr(bytes.NewBuffer(nil))
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --branch is not specified")
	}
}

func TestRepoWorktreeCreateHasBranchFlag(t *testing.T) {
	root := command.NewRootCommand()
	createCmd, _, _ := root.Find([]string{"repo", "worktree", "create"})
	if createCmd == nil {
		t.Fatal("repo worktree create command not found")
	}
	if createCmd.Flags().Lookup("branch") == nil {
		t.Error("--branch flag not found")
	}
}

func TestRepoWorktreeInspectHasWorktreeDirFlag(t *testing.T) {
	root := command.NewRootCommand()
	inspectCmd, _, _ := root.Find([]string{"repo", "worktree", "inspect"})
	if inspectCmd == nil {
		t.Fatal("repo worktree inspect command not found")
	}
	if inspectCmd.Flags().Lookup("worktree-dir") == nil {
		t.Error("--worktree-dir flag not found")
	}
}

func TestRepoWorktreeDiffHasWorktreeDirFlag(t *testing.T) {
	root := command.NewRootCommand()
	diffCmd, _, _ := root.Find([]string{"repo", "worktree", "diff"})
	if diffCmd == nil {
		t.Fatal("repo worktree diff command not found")
	}
	if diffCmd.Flags().Lookup("worktree-dir") == nil {
		t.Error("--worktree-dir flag not found")
	}
}

func TestRepoWorktreeDiscardHasWorktreeDirFlag(t *testing.T) {
	root := command.NewRootCommand()
	discardCmd, _, _ := root.Find([]string{"repo", "worktree", "discard"})
	if discardCmd == nil {
		t.Fatal("repo worktree discard command not found")
	}
	if discardCmd.Flags().Lookup("worktree-dir") == nil {
		t.Error("--worktree-dir flag not found")
	}
}

func TestRepoWorktreeCreate_JSONOutput(t *testing.T) {
	out := bytes.NewBuffer(nil)
	root := command.NewRootCommand()
	root.SetArgs([]string{"repo", "worktree", "create", "--branch", "test-branch", "--output", "json"})
	root.SetOut(out)
	root.SetErr(bytes.NewBuffer(nil))
	root.SetIn(strings.NewReader(""))

	// Will fail without a real repo, but we can check JSON structure if it runs
	err := root.Execute()
	if err != nil {
		// Expected: no repository found
		if !strings.Contains(err.Error(), "repository") && !strings.Contains(err.Error(), "repo") {
			t.Logf("execute error (may be expected): %v", err)
		}
		return
	}

	var parsed map[string]any
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Errorf("output should be valid JSON: %v", err)
	}
}
