package analyzer

import (
	"fmt"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

// SuggestPullStrategy returns "rebase" or "merge" for git pull.
// Clean working tree + linear history -> rebase; diverged -> merge.
func SuggestPullStrategy(state *status.GitState) string {
	if state == nil {
		return "merge"
	}

	// Check for clean working tree (no uncommitted changes)
	hasChanges := len(state.WorkingTree) > 0 || len(state.StagingArea) > 0
	for _, f := range state.WorkingTree {
		if f.WorktreeCode != git.StatusUnmodified && f.WorktreeCode != git.StatusIgnored {
			hasChanges = true
			break
		}
	}
	for _, f := range state.StagingArea {
		if f.StagingCode != git.StatusUnmodified {
			hasChanges = true
			break
		}
	}

	// Divergence: ahead and behind
	ahead, behind := 0, 0
	if state.UpstreamState != nil {
		ahead = state.UpstreamState.Ahead
		behind = state.UpstreamState.Behind
	} else {
		ahead = state.LocalBranch.Ahead
		behind = state.LocalBranch.Behind
	}
	diverged := ahead > 0 && behind > 0

	// If diverged -> merge (safer for shared history)
	if diverged {
		return "merge"
	}
	// Uncommitted changes -> merge (rebase with changes is trickier)
	if hasChanges {
		return "merge"
	}
	// Clean working tree and linear (only behind, not ahead) -> rebase
	if behind > 0 && ahead == 0 {
		return "rebase"
	}
	// Default to merge for safety
	return "merge"
}

// SuggestMergeStrategy returns the recommended strategy ("merge", "squash", "rebase")
// and a list of pre-check warnings.
func SuggestMergeStrategy(source, target string, state *status.GitState) (string, []string) {
	if state == nil {
		return "merge", nil
	}

	var warnings []string

	// Count conflicting (unmerged) files
	conflictPaths := make(map[string]bool)
	for _, f := range state.StagingArea {
		if f.StagingCode == git.StatusUnmerged {
			conflictPaths[f.Path] = true
		}
	}
	for _, f := range state.WorkingTree {
		if f.StagingCode == git.StatusUnmerged || f.WorktreeCode == git.StatusUnmerged {
			conflictPaths[f.Path] = true
		}
	}
	conflictCount := len(conflictPaths)
	if conflictCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d conflicting files detected", conflictCount))
	}

	// Branch divergence: ahead/behind
	ahead, behind := 0, 0
	if state.UpstreamState != nil {
		ahead = state.UpstreamState.Ahead
		behind = state.UpstreamState.Behind
	} else {
		ahead = state.LocalBranch.Ahead
		behind = state.LocalBranch.Behind
	}
	if ahead > 0 && behind > 0 {
		warnings = append(warnings, fmt.Sprintf("Branch has diverged: %d ahead, %d behind", ahead, behind))
	} else if behind > 0 {
		warnings = append(warnings, fmt.Sprintf("Branch is %d commits behind target", behind))
	}

	// Choose strategy
	strategy := "merge"
	// Prefer squash for feature branches with many commits
	if ahead > 3 && conflictCount == 0 {
		strategy = "squash"
	}
	// Prefer rebase when behind and no conflicts
	if behind > 0 && conflictCount == 0 && ahead <= 5 {
		strategy = "rebase"
	}
	// If conflicts exist, merge is safest
	if conflictCount > 0 {
		strategy = "merge"
	}

	return strategy, warnings
}
