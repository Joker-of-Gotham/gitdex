package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSuggestionsAgainstState_NormalizesCheckoutToSwitch(t *testing.T) {
	state := &status.GitState{
		LocalBranch:   git.BranchInfo{Name: "main"},
		LocalBranches: []string{"main", "dev"},
	}

	out := ValidateSuggestionsAgainstState([]git.Suggestion{{
		Action:  "Switch to dev",
		Command: []string{"git", "checkout", "dev"},
	}}, state)

	require.Len(t, out, 1)
	assert.Equal(t, []string{"git", "switch", "dev"}, out[0].Command)
}

func TestValidateSuggestionsAgainstState_FiltersInvalidBranchOperations(t *testing.T) {
	state := &status.GitState{
		LocalBranch:   git.BranchInfo{Name: "dev"},
		LocalBranches: []string{"main", "dev"},
	}

	out := ValidateSuggestionsAgainstState([]git.Suggestion{
		{Action: "Switch current branch", Command: []string{"git", "checkout", "dev"}},
		{Action: "Create duplicate branch", Command: []string{"git", "switch", "-c", "dev"}},
		{Action: "Delete current branch", Command: []string{"git", "branch", "-d", "dev"}},
	}, state)

	assert.Empty(t, out)
}

func TestValidateSuggestionsAgainstState_FiltersMissingRemoteAndPath(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		Remotes:     []string{"origin"},
		RemoteInfos: []git.RemoteInfo{{Name: "origin"}},
	}

	out := ValidateSuggestionsAgainstState([]git.Suggestion{
		{Action: "Bad push", Command: []string{"git", "push", "upstream", "main"}},
		{Action: "Bad remote prune", Command: []string{"git", "remote", "prune", "upstream"}},
		{Action: "Missing ignore target", Command: []string{"git", "check-ignore", "-v", "gitmanual.exe"}},
	}, state)

	assert.Empty(t, out)
}

func TestValidateSuggestionsAgainstState_AllowsCheckIgnoreForExistingPath(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tempDir))
	defer func() { _ = os.Chdir(oldWd) }()

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "gitdex.exe"), []byte("x"), 0o600))

	state := &status.GitState{}
	out := ValidateSuggestionsAgainstState([]git.Suggestion{{
		Action:  "Inspect ignore rule",
		Command: []string{"git", "check-ignore", "-v", "gitdex.exe"},
	}}, state)

	require.Len(t, out, 1)
	assert.Equal(t, []string{"git", "check-ignore", "-v", "gitdex.exe"}, out[0].Command)
}
