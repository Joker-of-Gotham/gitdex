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

	if interaction == git.PlatformExec {
		if strings.TrimSpace(item.CapabilityID) == "" {
			return git.Suggestion{}, fmt.Errorf("AI suggestion %d is platform_exec but missing capability_id", idx)
		}
		flow := strings.ToLower(strings.TrimSpace(item.Flow))
		switch flow {
		case "inspect", "mutate", "validate", "rollback":
		default:
			return git.Suggestion{}, fmt.Errorf("AI suggestion %d is platform_exec but has invalid flow %q", idx, item.Flow)
		}

		inputs := make([]git.InputField, 0, len(item.Inputs))
		for _, in := range item.Inputs {
			label := strings.TrimSpace(in.Label)
			if label == "" {
				label = "Value"
			}
			key := strings.TrimSpace(in.Key)
			if key == "" {
				return git.Suggestion{}, fmt.Errorf("AI suggestion %d platform_exec input missing key", idx)
			}
			inputs = append(inputs, git.InputField{
				Key:          key,
				Label:        label,
				Placeholder:  defaultInputPlaceholder(key, label, strings.TrimSpace(in.Placeholder)),
				ArgIndex:     -1,
				DefaultValue: in.DefaultValue,
			})
		}

		s := git.Suggestion{
			ID:          fmt.Sprintf("llm-%d", idx),
			Action:      strings.TrimSpace(item.Action),
			Reason:      strings.TrimSpace(item.Reason),
			RiskLevel:   parseRisk(item.Risk),
			Source:      git.SourceLLM,
			Confidence:  0.85,
			Interaction: git.PlatformExec,
			Inputs:      inputs,
			PlatformOp: &git.PlatformExecInfo{
				CapabilityID:    strings.TrimSpace(item.CapabilityID),
				Flow:            flow,
				Operation:       strings.TrimSpace(item.Operation),
				ResourceID:      strings.TrimSpace(item.ResourceID),
				Scope:           cloneStringMap(item.Scope),
				Query:           cloneStringMap(item.Query),
				Payload:         append([]byte(nil), item.Payload...),
				ValidatePayload: append([]byte(nil), item.ValidatePayload...),
				RollbackPayload: append([]byte(nil), item.RollbackPayload...),
			},
		}
		if s.Action == "" {
			s.Action = fmt.Sprintf("%s %s", s.PlatformOp.Flow, s.PlatformOp.CapabilityID)
		}
		if s.Reason == "" {
			s.Reason = "AI judged this platform administration flow is needed."
		}
		return s, nil
	}

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
	case "platform_exec", "platform":
		return git.PlatformExec
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

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func parseRisk(r string) git.RiskLevel {
	switch strings.ToLower(strings.TrimSpace(r)) {
	case "safe", "low", "none":
		return git.RiskSafe
	case "caution", "warning", "medium":
		return git.RiskCaution
	case "dangerous", "danger", "high":
		return git.RiskDangerous
	default:
		return git.RiskSafe
	}
}
