package engine

import (
	"fmt"
	"strings"

	gitctx "github.com/Joker-of-Gotham/gitdex/internal/engine/context"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

func convertSuggestions(state *status.GitState, items []llmSuggestionJSON) ([]git.Suggestion, []string, error) {
	var out []git.Suggestion
	var rejected []string
	for i, item := range items {
		s, err := convertSuggestion(state, i, item)
		if err != nil {
			rejected = append(rejected, err.Error())
			continue
		}
		out = append(out, s)
	}
	return out, rejected, nil
}

func convertSuggestion(_ *status.GitState, idx int, item llmSuggestionJSON) (git.Suggestion, error) {
	argv := item.Argv
	if len(argv) == 0 && strings.TrimSpace(item.Command) != "" {
		argv = shellSplit(item.Command)
	}
	interaction := normalizeInteraction(item.Interaction, argv)

	if interaction == git.FileWrite {
		if strings.TrimSpace(item.FilePath) == "" {
			return git.Suggestion{}, fmt.Errorf("AI suggestion %d is file_write but missing file_path", idx)
		}
		operation := strings.ToLower(strings.TrimSpace(item.FileOperation))
		if operation == "" {
			// Default: if file exists, update; otherwise create
			operation = "create"
		}
		s := git.Suggestion{
			ID:          fmt.Sprintf("llm-%d", idx),
			Action:      strings.TrimSpace(item.Action),
			Reason:      strings.TrimSpace(item.Reason),
			RiskLevel:   parseRisk(item.Risk),
			Source:      git.SourceLLM,
			Confidence:  0.85,
			Interaction: git.FileWrite,
			FileOp: &git.FileWriteInfo{
				Path:      strings.TrimSpace(item.FilePath),
				Content:   item.FileContent,
				Operation: operation,
				Backup:    operation == "update" || operation == "delete", // auto-backup for destructive ops
			},
		}
		if s.Action == "" {
			switch operation {
			case "update":
				s.Action = "Update file: " + s.FileOp.Path
			case "delete":
				s.Action = "Delete file: " + s.FileOp.Path
			case "append":
				s.Action = "Append to file: " + s.FileOp.Path
			default:
				s.Action = "Create file: " + s.FileOp.Path
			}
		}
		if s.Reason == "" {
			s.Reason = "AI judged this file operation is needed."
		}
		return s, nil
	}

	if interaction == git.InfoOnly {
		s := git.Suggestion{
			ID:          fmt.Sprintf("llm-%d", idx),
			Action:      strings.TrimSpace(item.Action),
			Command:     argv,
			Reason:      strings.TrimSpace(item.Reason),
			RiskLevel:   parseRisk(item.Risk),
			Source:      git.SourceLLM,
			Confidence:  0.85,
			Interaction: git.InfoOnly,
		}
		if s.Action == "" {
			s.Action = "AI advisory"
		}
		if s.Reason == "" {
			s.Reason = "AI judged this to be the next best step."
		}
		return s, nil
	}

	if len(argv) < 2 || !strings.EqualFold(argv[0], "git") {
		return git.Suggestion{}, fmt.Errorf("AI suggestion %d is not a valid git argv array", idx)
	}

	argv = sanitizeSuggestedArgv(argv)
	argv, item.Inputs, interaction = normalizeGitSuggestion(argv, item.Inputs, interaction)

	if interaction == git.AutoExec {
		detected := detectPlaceholdersInArgv(argv)
		if len(detected) > 0 {
			interaction = git.NeedsInput
			if len(item.Inputs) == 0 {
				item.Inputs = detected
			}
		}
	}

	if interaction == git.AutoExec && len(argv) >= 2 {
		sub := strings.ToLower(argv[1])
		if info, ok := gitctx.Get().Subcommands[sub]; ok && info.RequiresMessage {
			hasMessageFlag := false
			for _, a := range argv[2:] {
				if gitctx.Get().IsMessageFlag(a) || strings.HasPrefix(a, "-m") || strings.HasPrefix(a, "--message=") {
					hasMessageFlag = true
					break
				}
				if gitctx.Get().IsSkipMessageFlag(a) {
					hasMessageFlag = true
					break
				}
			}
			if !hasMessageFlag {
				interaction = git.NeedsInput
				argv = append(argv, "-m", "<commit-message>")
				item.Inputs = append(item.Inputs, llmInputJSON{
					Key:      "commit_message",
					Label:    info.DefaultInputLabel,
					ArgIndex: len(argv) - 1,
				})
			}
		}
	}

	var inputs []git.InputField
	if interaction == git.NeedsInput {
		if len(item.Inputs) == 0 {
			return git.Suggestion{}, fmt.Errorf("AI suggestion %d needs input but did not provide inputs[]", idx)
		}
		for _, in := range item.Inputs {
			argIndex := in.ArgIndex
			if argIndex < 2 || argIndex >= len(argv) {
				if remapped, ok := inferInputArgIndex(argv, in); ok {
					argIndex = remapped
				}
			}
			if argIndex < 2 || argIndex >= len(argv) {
				return git.Suggestion{}, fmt.Errorf("AI suggestion %d has invalid input arg_index %d", idx, in.ArgIndex)
			}
			label := strings.TrimSpace(in.Label)
			if label == "" {
				label = "Value"
			}
			inputs = append(inputs, git.InputField{
				Key:          strings.TrimSpace(in.Key),
				Label:        label,
				Placeholder:  defaultInputPlaceholder(strings.TrimSpace(in.Key), label, strings.TrimSpace(in.Placeholder)),
				ArgIndex:     argIndex,
				DefaultValue: in.DefaultValue,
			})
		}
	}

	s := git.Suggestion{
		ID:          fmt.Sprintf("llm-%d", idx),
		Action:      strings.TrimSpace(item.Action),
		Command:     argv,
		Reason:      strings.TrimSpace(item.Reason),
		RiskLevel:   parseRisk(item.Risk),
		Source:      git.SourceLLM,
		Confidence:  0.85,
		Interaction: interaction,
		Inputs:      inputs,
	}
	if s.Action == "" {
		s.Action = "AI suggestion"
	}
	if s.Reason == "" {
		s.Reason = "AI judged this to be the next best step."
	}
	return s, nil
}

func normalizeInteraction(raw string, argv []string) git.InteractionMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "needs_input", "input":
		return git.NeedsInput
	case "info", "advisory":
		return git.InfoOnly
	case "file_write", "file_create", "write_file", "create_file":
		return git.FileWrite
	case "auto", "":
		if len(argv) == 0 {
			return git.InfoOnly
		}
		return git.AutoExec
	default:
		if len(argv) == 0 {
			return git.InfoOnly
		}
		return git.AutoExec
	}
}

func parseRisk(r string) git.RiskLevel {
	switch strings.ToLower(strings.TrimSpace(r)) {
	case "caution", "warning":
		return git.RiskCaution
	case "dangerous", "danger", "high":
		return git.RiskDangerous
	default:
		return git.RiskCaution
	}
}
