package engine

import (
	"os"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
)

func suppressRepeatedSuccessfulSuggestions(suggestions []git.Suggestion, recentOps []prompt.OperationRecord) []git.Suggestion {
	if len(suggestions) == 0 || len(recentOps) == 0 {
		return suggestions
	}
	successful := make(map[string]struct{}, len(recentOps))
	viewed := make(map[string]struct{}, len(recentOps))
	for _, op := range recentOps {
		if strings.EqualFold(strings.TrimSpace(op.Result), "success") && strings.TrimSpace(op.Command) != "" {
			successful[normalizeCommandIdentity(op.Command)] = struct{}{}
		}
		if strings.EqualFold(strings.TrimSpace(op.Type), "viewed") && strings.TrimSpace(op.Action) != "" {
			viewed[normalizeCommandIdentity(op.Action)] = struct{}{}
		}
	}
	if len(successful) == 0 && len(viewed) == 0 {
		return suggestions
	}

	filtered := make([]git.Suggestion, 0, len(suggestions))
	for _, s := range suggestions {
		if s.Interaction == git.InfoOnly {
			if _, ok := viewed[normalizeCommandIdentity(s.Action)]; ok {
				continue
			}
			filtered = append(filtered, s)
			continue
		}
		if len(s.Command) == 0 {
			filtered = append(filtered, s)
			continue
		}
		if _, ok := successful[normalizeCommandIdentity(strings.Join(s.Command, " "))]; ok {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

// ValidateSuggestionsAgainstState filters out suggestions that are physically
// impossible given the current repository state (e.g., deleting a file that
// doesn't exist, committing when nothing is staged).
func ValidateSuggestionsAgainstState(suggestions []git.Suggestion, state *status.GitState) []git.Suggestion {
	if len(suggestions) == 0 || state == nil {
		return suggestions
	}
	filtered := make([]git.Suggestion, 0, len(suggestions))
	for _, s := range suggestions {
		if !isSuggestionValid(s, state) {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

func isSuggestionValid(s git.Suggestion, state *status.GitState) bool {
	// Validate file_write delete: file must exist
	if s.Interaction == git.FileWrite && s.FileOp != nil {
		op := strings.ToLower(s.FileOp.Operation)
		if op == "delete" && !fileExists(s.FileOp.Path) {
			return false
		}
		return true
	}

	if len(s.Command) < 2 {
		return true
	}

	sub := strings.ToLower(s.Command[1])

	switch sub {
	case "commit":
		return len(state.StagingArea) > 0

	case "rm":
		for _, arg := range s.Command[2:] {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			if !fileExists(arg) {
				return false
			}
		}

	case "add":
		if len(state.WorkingTree) == 0 {
			return false
		}
		for _, arg := range s.Command[2:] {
			if arg == "." || arg == "-A" || arg == "--all" || strings.HasPrefix(arg, "-") {
				continue
			}
			found := false
			for _, f := range state.WorkingTree {
				if f.Path == arg {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}

	case "push":
		if len(state.RemoteInfos) == 0 {
			return false
		}
	}

	return true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func normalizeCommandIdentity(cmd string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(cmd))), " ")
}
