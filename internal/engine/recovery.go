package engine

import (
	"fmt"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

// RecoveryTriage suggests recovery commands based on current repository state.
func RecoveryTriage(state *status.GitState) []git.Suggestion {
	if state == nil {
		return nil
	}

	var suggestions []git.Suggestion

	workingCount := len(state.WorkingTree)
	stagedCount := len(state.StagingArea)

	if workingCount > 0 {
		suggestions = append(suggestions, git.Suggestion{
			ID:          "restore-working",
			Action:      "Discard working tree changes",
			Command:     []string{"git", "restore", "."},
			Reason:      fmt.Sprintf("The working tree has %d unstaged changes. This discards them.", workingCount),
			RiskLevel:   git.RiskDangerous,
			Interaction: git.AutoExec,
		})
		suggestions = append(suggestions, git.Suggestion{
			ID:          "stash-working",
			Action:      "Stash working tree changes",
			Command:     []string{"git", "stash", "push", "-m", "auto-stash before recovery"},
			Reason:      "Safer than discarding changes. You can recover them later with git stash pop.",
			RiskLevel:   git.RiskSafe,
			Interaction: git.AutoExec,
		})
	}

	if stagedCount > 0 {
		suggestions = append(suggestions, git.Suggestion{
			ID:          "unstage-all",
			Action:      "Unstage all files",
			Command:     []string{"git", "restore", "--staged", "."},
			Reason:      fmt.Sprintf("The index has %d staged files. This moves them back to the working tree.", stagedCount),
			RiskLevel:   git.RiskSafe,
			Interaction: git.AutoExec,
		})
	}

	if !state.IsInitial && state.CommitCount > 0 {
		hasUpstream := state.UpstreamState != nil
		isAhead := len(state.AheadCommits) > 0

		if isAhead || !hasUpstream {
			suggestions = append(suggestions, git.Suggestion{
				ID:          "reset-soft-last",
				Action:      "Undo the last commit but keep changes staged",
				Command:     []string{"git", "reset", "--soft", "HEAD~1"},
				Reason:      "Move HEAD back one commit while keeping the changes staged for a new commit.",
				RiskLevel:   git.RiskCaution,
				Interaction: git.AutoExec,
			})
		}

		if hasUpstream && !isAhead {
			suggestions = append(suggestions, git.Suggestion{
				ID:          "revert-last",
				Action:      "Revert the last published commit",
				Command:     []string{"git", "revert", "HEAD"},
				Reason:      "Create a new commit that safely undoes the last published commit.",
				RiskLevel:   git.RiskCaution,
				Interaction: git.AutoExec,
			})
		}

		suggestions = append(suggestions, git.Suggestion{
			ID:          "reflog-rescue",
			Action:      "Inspect reflog for recovery points",
			Command:     []string{"git", "reflog", "--oneline", "-20"},
			Reason:      "Reflog shows recent HEAD movements and can help you find a commit to restore.",
			RiskLevel:   git.RiskSafe,
			Interaction: git.AutoExec,
		})
	}

	return suggestions
}
