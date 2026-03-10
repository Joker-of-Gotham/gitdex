package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gitctx "github.com/Joker-of-Gotham/gitdex/internal/engine/context"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
)

type verifyResponseJSON struct {
	Suggestions []verifySuggestionJSON `json:"suggestions"`
}

type verifySuggestionJSON struct {
	Index       int      `json:"index"`
	Argv        []string `json:"argv,omitempty"`
	Risk        string   `json:"risk,omitempty"`
	Interaction string   `json:"interaction,omitempty"`
	Issues      []string `json:"issues,omitempty"`
}

type verifier struct {
	provider llm.LLMProvider
	model    string
}

func newVerifier(provider llm.LLMProvider, model string) *verifier {
	return &verifier{
		provider: provider,
		model:    strings.TrimSpace(model),
	}
}

func (v *verifier) Verify(ctx context.Context, state *status.GitState, suggestions []git.Suggestion) ([]git.Suggestion, int, error) {
	if v == nil || v.provider == nil || v.model == "" || len(suggestions) == 0 {
		return suggestions, 0, nil
	}

	system, user := v.buildPrompt(state, suggestions)
	resp, err := llm.GenerateText(ctx, v.provider, llm.GenerateRequest{
		Model:       v.model,
		Role:        llm.RoleSecondary,
		System:      system,
		Prompt:      user,
		Temperature: 0.0,
	})
	if err != nil {
		return suggestions, 0, err
	}

	parsed, err := parseVerifierResponse(resp.Text)
	if err != nil {
		return suggestions, 0, err
	}

	out := append([]git.Suggestion(nil), suggestions...)
	fixes := 0
	for _, item := range parsed.Suggestions {
		if item.Index < 0 || item.Index >= len(out) {
			continue
		}
		s := out[item.Index]
		changed := false

		if len(item.Argv) > 0 && strings.EqualFold(item.Argv[0], "git") {
			next := sanitizeSuggestedArgv(item.Argv)
			if !sameStringSlice(s.Command, next) {
				s.Command = next
				changed = true
			}
		}
		if strings.TrimSpace(item.Interaction) != "" {
			next := normalizeInteraction(item.Interaction, s.Command)
			if next != s.Interaction {
				s.Interaction = next
				changed = true
			}
		}
		if strings.TrimSpace(item.Risk) != "" {
			next := parseRisk(item.Risk)
			if next != s.RiskLevel {
				s.RiskLevel = next
				changed = true
			}
		}

		s, normalized := normalizeSuggestionPostVerify(s)
		if normalized {
			changed = true
		}
		if changed {
			fixes++
		}
		out[item.Index] = s
	}

	return out, fixes, nil
}

func (v *verifier) buildPrompt(state *status.GitState, suggestions []git.Suggestion) (system, user string) {
	system = `You are a strict Git suggestion verifier.
You receive a list of already-generated suggestions.
For each suggestion, verify:
1) git argv syntax and argument order
2) placeholder-like arguments
3) risk level correctness
4) interaction type correctness

Return strict JSON only:
{
  "suggestions": [
    {
      "index": 0,
      "argv": ["git","push","-u","origin","main"],
      "risk": "safe|caution|dangerous",
      "interaction": "auto|needs_input|info|file_write",
      "issues": ["..."]
    }
  ]
}

Rules:
- Keep suggestion count/order by using index.
- If no change is needed for a suggestion, you may omit it from output.
- Never output markdown or explanatory prose.`

	type verifyItem struct {
		Index       int      `json:"index"`
		Action      string   `json:"action"`
		Argv        []string `json:"argv"`
		Reason      string   `json:"reason"`
		Risk        string   `json:"risk"`
		Interaction string   `json:"interaction"`
	}
	type verifyInput struct {
		Branch      string       `json:"branch"`
		Upstream    string       `json:"upstream,omitempty"`
		Suggestions []verifyItem `json:"suggestions"`
	}

	in := verifyInput{
		Branch:   state.LocalBranch.Name,
		Upstream: state.LocalBranch.Upstream,
	}
	for i, s := range suggestions {
		in.Suggestions = append(in.Suggestions, verifyItem{
			Index:       i,
			Action:      s.Action,
			Argv:        s.Command,
			Reason:      s.Reason,
			Risk:        riskToString(s.RiskLevel),
			Interaction: interactionLabel(s.Interaction),
		})
	}
	data, _ := json.MarshalIndent(in, "", "  ")
	user = "Verify the following suggestions JSON:\n" + string(data)
	return system, user
}

func parseVerifierResponse(text string) (verifyResponseJSON, error) {
	text = normalizeStructuredResponseText(text)
	if text == "" {
		return verifyResponseJSON{}, fmt.Errorf("empty verifier response")
	}
	firstBrace := strings.Index(text, "{")
	if firstBrace < 0 {
		return verifyResponseJSON{}, fmt.Errorf("verifier response is not valid JSON")
	}
	candidate := text[firstBrace:]
	if end := findMatchingBrace(candidate); end > 0 {
		candidate = candidate[:end+1]
	}
	var out verifyResponseJSON
	if err := json.Unmarshal([]byte(candidate), &out); err != nil {
		return verifyResponseJSON{}, err
	}
	return out, nil
}

func normalizeSuggestionPostVerify(s git.Suggestion) (git.Suggestion, bool) {
	changed := false
	if len(s.Command) == 0 || !strings.EqualFold(s.Command[0], "git") {
		return s, changed
	}
	argv := sanitizeSuggestedArgv(s.Command)
	if !sameStringSlice(argv, s.Command) {
		s.Command = argv
		changed = true
	}

	inputs := toLLMInputs(s.Inputs)
	argv, inputs, interaction := normalizeGitSuggestion(argv, inputs, s.Interaction)
	if interaction == git.AutoExec {
		detected := detectPlaceholdersInArgv(argv)
		if len(detected) > 0 {
			interaction = git.NeedsInput
			if len(inputs) == 0 {
				inputs = detected
			}
		}
	}
	nextInputs := s.Inputs
	if interaction == git.NeedsInput {
		converted, err := llmInputsToFields(argv, inputs)
		if err == nil {
			nextInputs = converted
		}
	}
	if !sameStringSlice(argv, s.Command) {
		s.Command = argv
		changed = true
	}
	if s.Interaction != interaction {
		s.Interaction = interaction
		changed = true
	}
	if !sameInputFields(s.Inputs, nextInputs) {
		s.Inputs = nextInputs
		changed = true
	}
	return s, changed
}

func toLLMInputs(in []git.InputField) []llmInputJSON {
	out := make([]llmInputJSON, 0, len(in))
	for _, item := range in {
		out = append(out, llmInputJSON{
			Key:          item.Key,
			Label:        item.Label,
			Placeholder:  item.Placeholder,
			DefaultValue: item.DefaultValue,
			ArgIndex:     item.ArgIndex,
		})
	}
	return out
}

func llmInputsToFields(argv []string, in []llmInputJSON) ([]git.InputField, error) {
	fields := make([]git.InputField, 0, len(in))
	for _, item := range in {
		argIndex := item.ArgIndex
		if argIndex < 2 || argIndex >= len(argv) {
			if remapped, ok := inferInputArgIndex(argv, item); ok {
				argIndex = remapped
			}
		}
		if argIndex < 2 || argIndex >= len(argv) {
			return nil, fmt.Errorf("invalid arg_index: %d", item.ArgIndex)
		}
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = "Value"
		}
		fields = append(fields, git.InputField{
			Key:          strings.TrimSpace(item.Key),
			Label:        label,
			Placeholder:  defaultInputPlaceholder(strings.TrimSpace(item.Key), label, strings.TrimSpace(item.Placeholder)),
			ArgIndex:     argIndex,
			DefaultValue: item.DefaultValue,
		})
	}
	if len(fields) == 0 && len(argv) >= 2 && strings.EqualFold(argv[0], "git") {
		sub := strings.ToLower(strings.TrimSpace(argv[1]))
		if info, ok := gitctx.Get().Subcommands[sub]; ok && info.RequiresMessage {
			_, hasMessage, _ := findCommitMessageArg(argv)
			if !hasMessage {
				return nil, fmt.Errorf("commit requires message input")
			}
		}
	}
	return fields, nil
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameInputFields(a, b []git.InputField) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func interactionLabel(m git.InteractionMode) string {
	switch m {
	case git.NeedsInput:
		return "needs_input"
	case git.InfoOnly:
		return "info"
	case git.FileWrite:
		return "file_write"
	default:
		return "auto"
	}
}

func riskToString(r git.RiskLevel) string {
	switch r {
	case git.RiskSafe:
		return "safe"
	case git.RiskDangerous:
		return "dangerous"
	default:
		return "caution"
	}
}
