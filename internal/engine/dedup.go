package engine

import (
	"fmt"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
)

type SuggestionValidationIssue struct {
	Suggestion git.Suggestion
	Reason     string
}

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
		if s.Interaction == git.PlatformExec {
			identity := git.PlatformExecIdentity(s.PlatformOp)
			if identity != "" {
				if _, ok := successful[normalizeCommandIdentity(identity)]; ok {
					continue
				}
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
	validated, _ := ValidateSuggestionsWithIssues(suggestions, state)
	return validated
}

func ValidateSuggestionsWithIssues(suggestions []git.Suggestion, state *status.GitState) ([]git.Suggestion, []SuggestionValidationIssue) {
	if len(suggestions) == 0 || state == nil {
		return suggestions, nil
	}
	filtered := make([]git.Suggestion, 0, len(suggestions))
	issues := make([]SuggestionValidationIssue, 0)
	for _, s := range suggestions {
		next, ok := normalizeSuggestionAgainstState(s, state)
		if !ok {
			issues = append(issues, SuggestionValidationIssue{
				Suggestion: s,
				Reason:     validationIssueReason(s),
			})
			continue
		}
		filtered = append(filtered, next)
	}
	return filtered, issues
}

func normalizeCommandIdentity(cmd string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(cmd))), " ")
}

// semanticOpKey extracts a coarse operation intent from a git command,
// so "git switch -c foo" and "git switch -c bar" both map to "switch -c".
func semanticOpKey(argv []string) string {
	if len(argv) < 2 || !strings.EqualFold(argv[0], "git") {
		return ""
	}
	sub := strings.ToLower(argv[1])
	switch sub {
	case "switch", "checkout":
		for _, a := range argv[2:] {
			if a == "-c" || a == "-b" || a == "--create" {
				return sub + " -c"
			}
		}
		return sub
	case "branch":
		for _, a := range argv[2:] {
			if a == "-d" || a == "-D" || a == "--delete" {
				return "branch -d"
			}
			if a == "-m" || a == "-M" || a == "--move" {
				return "branch -m"
			}
		}
		if len(argv) > 2 && !strings.HasPrefix(argv[2], "-") {
			return "branch create"
		}
		return sub
	case "tag":
		for _, a := range argv[2:] {
			if a == "-d" || a == "--delete" {
				return "tag -d"
			}
		}
		if len(argv) > 2 && !strings.HasPrefix(argv[2], "-") {
			return "tag create"
		}
		return sub
	case "remote":
		if len(argv) > 2 {
			return "remote " + strings.ToLower(argv[2])
		}
		return sub
	case "stash":
		if len(argv) > 2 {
			return "stash " + strings.ToLower(argv[2])
		}
		return "stash push"
	default:
		return sub
	}
}

func suppressSemanticDuplicates(suggestions []git.Suggestion, recentOps []prompt.OperationRecord) []git.Suggestion {
	if len(suggestions) == 0 || len(recentOps) == 0 {
		return suggestions
	}
	recentSemanticOps := make(map[string]struct{}, len(recentOps))
	for _, op := range recentOps {
		if !strings.EqualFold(strings.TrimSpace(op.Result), "success") {
			continue
		}
		cmd := strings.TrimSpace(op.Command)
		if cmd == "" {
			continue
		}
		key := semanticOpKey(strings.Fields(cmd))
		if key != "" {
			recentSemanticOps[key] = struct{}{}
		}
	}
	if len(recentSemanticOps) == 0 {
		return suggestions
	}

	seen := make(map[string]bool)
	filtered := make([]git.Suggestion, 0, len(suggestions))
	for _, s := range suggestions {
		if s.Interaction != git.AutoExec || len(s.Command) < 2 {
			filtered = append(filtered, s)
			continue
		}
		key := semanticOpKey(s.Command)
		if key == "" {
			filtered = append(filtered, s)
			continue
		}
		if _, recentlyDone := recentSemanticOps[key]; recentlyDone && seen[key] {
			continue
		}
		seen[key] = true
		filtered = append(filtered, s)
	}
	return filtered
}

func validationIssueReason(s git.Suggestion) string {
	action := strings.TrimSpace(s.Action)
	if action == "" {
		action = strings.TrimSpace(strings.Join(s.Command, " "))
	}
	if action == "" && s.FileOp != nil {
		action = strings.TrimSpace(s.FileOp.Operation + " " + s.FileOp.Path)
	}
	if action == "" {
		action = "unknown suggestion"
	}
	return fmt.Sprintf("invalid for current repository state: %s", action)
}
