package engine

import (
	"context"
	"encoding/json"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/response"
)

type repairRequest struct {
	repoStateSnapshot
	Rejected []repairRejectedProposal `json:"rejected"`
}

type repairRejectedProposal struct {
	Action      string   `json:"action"`
	Command     []string `json:"command,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	Risk        string   `json:"risk,omitempty"`
	Interaction string   `json:"interaction,omitempty"`
	RejectedFor string   `json:"rejected_for"`
}

func (p *Pipeline) repairRejectedSuggestions(ctx context.Context, state *status.GitState, issues []SuggestionValidationIssue) ([]git.Suggestion, []SuggestionValidationIssue, error) {
	if p == nil || p.llmProvider == nil || len(issues) == 0 {
		return nil, nil, nil
	}

	system, user, err := buildRepairPrompt(state, issues)
	if err != nil {
		return nil, nil, err
	}
	resp, err := llm.GenerateText(ctx, p.llmProvider, llm.GenerateRequest{
		Model:       p.primaryModel,
		Role:        llm.RolePrimary,
		System:      system,
		Prompt:      user,
		Temperature: 0,
	})
	if err != nil {
		return nil, nil, err
	}

	_, cleaned := response.ExtractThinking(resp.Text)
	parsed, err := parseLLMResponse(state, cleaned)
	if err != nil {
		return nil, nil, err
	}
	valid, rejected := ValidateSuggestionsWithIssues(parsed.suggestions, state)
	return valid, rejected, nil
}

func buildRepairPrompt(state *status.GitState, issues []SuggestionValidationIssue) (system, user string, err error) {
	system = `You are a Git command repair engine.
You receive rejected Git suggestions and the current repository state.
Rewrite each rejected item into a VALID replacement suggestion that still advances the same intent.

Rules:
- Return strict JSON only.
- Output ONLY replacement suggestions; do not repeat valid suggestions that were not rejected.
- If a branch switch targets the current branch, replace it with an inspection or next-best action.
- Never create an already-existing branch.
- Never reference a missing remote, tag, or file path.
- Prefer modern Git forms such as "git switch".
- You may lower risk or change interaction if needed.

Schema:
[
  {
    "action": "short title",
    "argv": ["git","..."],
    "reason": "why this replacement is valid now",
    "risk": "safe|caution|dangerous",
    "interaction": "auto|needs_input|info|file_write"
  }
]`

	in := repairRequest{
		repoStateSnapshot: buildRepoStateSnapshot(state),
		Rejected:          make([]repairRejectedProposal, 0, len(issues)),
	}
	for _, issue := range issues {
		in.Rejected = append(in.Rejected, repairRejectedProposal{
			Action:      issue.Suggestion.Action,
			Command:     append([]string(nil), issue.Suggestion.Command...),
			Reason:      issue.Suggestion.Reason,
			Risk:        riskToString(issue.Suggestion.RiskLevel),
			Interaction: interactionLabel(issue.Suggestion.Interaction),
			RejectedFor: issue.Reason,
		})
	}

	data, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return "", "", err
	}
	user = "Repair the rejected suggestions below:\n" + string(data)
	return system, user, nil
}
