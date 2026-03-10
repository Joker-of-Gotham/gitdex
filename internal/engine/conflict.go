package engine

import (
	"context"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

// ConflictGuide generates suggestions for resolving merge/rebase/cherry-pick conflicts.
func ConflictGuide(state *status.GitState) []git.Suggestion {
	if state == nil {
		return nil
	}

	var suggestions []git.Suggestion

	if state.MergeInProgress {
		conflictFiles := unmergedFiles(state)
		if len(conflictFiles) > 0 {
			suggestions = append(suggestions, git.Suggestion{
				ID:          "conflict-list",
				Action:      "Review unmerged files",
				Command:     []string{"git", "diff", "--name-only", "--diff-filter=U"},
				Reason:      "A merge is in progress and these files still have unresolved conflicts: " + strings.Join(conflictFiles, ", "),
				RiskLevel:   git.RiskSafe,
				Interaction: git.AutoExec,
			})
		}
		suggestions = append(suggestions, git.Suggestion{
			ID:          "merge-continue",
			Action:      "Continue merge after resolving conflicts",
			Command:     []string{"git", "merge", "--continue"},
			Reason:      "Use this after all conflicts are resolved and the affected files are staged.",
			RiskLevel:   git.RiskCaution,
			Interaction: git.AutoExec,
		})
		suggestions = append(suggestions, git.Suggestion{
			ID:          "merge-abort",
			Action:      "Abort the current merge",
			Command:     []string{"git", "merge", "--abort"},
			Reason:      "Return to the pre-merge state and discard the in-progress merge attempt.",
			RiskLevel:   git.RiskCaution,
			Interaction: git.AutoExec,
		})
	}

	if state.RebaseInProgress {
		suggestions = append(suggestions, git.Suggestion{
			ID:          "rebase-continue",
			Action:      "Continue rebase after resolving conflicts",
			Command:     []string{"git", "rebase", "--continue"},
			Reason:      "Use this after resolving conflicts and staging the updated files.",
			RiskLevel:   git.RiskCaution,
			Interaction: git.AutoExec,
		})
		suggestions = append(suggestions, git.Suggestion{
			ID:          "rebase-skip",
			Action:      "Skip the current rebased commit",
			Command:     []string{"git", "rebase", "--skip"},
			Reason:      "Skip the current commit if you do not want to keep its changes.",
			RiskLevel:   git.RiskDangerous,
			Interaction: git.AutoExec,
		})
		suggestions = append(suggestions, git.Suggestion{
			ID:          "rebase-abort",
			Action:      "Abort the current rebase",
			Command:     []string{"git", "rebase", "--abort"},
			Reason:      "Stop the rebase and return the branch to its pre-rebase state.",
			RiskLevel:   git.RiskCaution,
			Interaction: git.AutoExec,
		})
	}

	if state.CherryInProgress {
		suggestions = append(suggestions, git.Suggestion{
			ID:          "cherry-continue",
			Action:      "Continue cherry-pick after resolving conflicts",
			Command:     []string{"git", "cherry-pick", "--continue"},
			Reason:      "Use this after resolving conflicts and staging the result.",
			RiskLevel:   git.RiskCaution,
			Interaction: git.AutoExec,
		})
		suggestions = append(suggestions, git.Suggestion{
			ID:          "cherry-abort",
			Action:      "Abort the current cherry-pick",
			Command:     []string{"git", "cherry-pick", "--abort"},
			Reason:      "Stop the cherry-pick and restore the state from before it started.",
			RiskLevel:   git.RiskCaution,
			Interaction: git.AutoExec,
		})
	}

	return suggestions
}

func unmergedFiles(state *status.GitState) []string {
	seen := make(map[string]bool)
	var files []string
	check := func(list []git.FileStatus) {
		for _, f := range list {
			if f.StagingCode == git.StatusUnmerged || f.WorktreeCode == git.StatusUnmerged {
				if !seen[f.Path] {
					files = append(files, f.Path)
					seen[f.Path] = true
				}
			}
		}
	}
	check(state.WorkingTree)
	check(state.StagingArea)
	return files
}

// IsConflictState returns true if the repo is in a merge/rebase/cherry-pick state.
func IsConflictState(state *status.GitState) bool {
	return state != nil && (state.MergeInProgress || state.RebaseInProgress || state.CherryInProgress)
}

// ConflictContext generates a conflict-specific context string for LLM analysis.
func ConflictContext(ctx context.Context, state *status.GitState) string {
	if !IsConflictState(state) {
		return ""
	}
	_ = ctx

	var parts []string
	if state.MergeInProgress {
		parts = append(parts, "MERGE IN PROGRESS")
	}
	if state.RebaseInProgress {
		parts = append(parts, "REBASE IN PROGRESS")
	}
	if state.CherryInProgress {
		parts = append(parts, "CHERRY-PICK IN PROGRESS")
	}
	if files := unmergedFiles(state); len(files) > 0 {
		parts = append(parts, "Unmerged files: "+strings.Join(files, ", "))
	}
	return strings.Join(parts, "\n")
}
