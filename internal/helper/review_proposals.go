package helper

import (
	"context"
	"fmt"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/jsonfix"
)

// ReviewResult holds the triaged creative goals.
type ReviewResult struct {
	ApprovedGitdexGoals []string
	ApprovedCreative    []string
	Discarded           []string
}

// ProposalReviewer reviews Planner-generated goals, deduplicates, and classifies.
type ProposalReviewer struct {
	LLM      llm.LLMProvider
	Store    *dotgitdex.Manager
	Language string
}

type reviewResponse struct {
	ApprovedGitdex   []string `json:"approved_gitdex"`
	ApprovedCreative []string `json:"approved_creative"`
	Discarded        []string `json:"discarded"`
}

// ReviewProposals processes Planner output, checking against existing goals
// and discarded proposals to avoid duplicates.
func (pr *ProposalReviewer) ReviewProposals(
	ctx context.Context,
	gitdexGoals, creativeGoals []string,
	existingGoals []dotgitdex.Goal,
) (*ReviewResult, error) {

	discarded, _ := pr.Store.ReadDiscardedProposals()

	existingTitles := make([]string, 0, len(existingGoals))
	for _, g := range existingGoals {
		existingTitles = append(existingTitles, g.Title)
	}

	system := `You are a proposal reviewer. Given candidate goals, existing goals, and previously discarded goals, classify each candidate:
- approved_gitdex: actionable goals that don't duplicate existing ones
- approved_creative: strategic insights that don't duplicate existing ones
- discarded: duplicates, low-value, or infeasible proposals

OUTPUT FORMAT (strict JSON, no markdown fences):
{"approved_gitdex": ["..."], "approved_creative": ["..."], "discarded": ["..."]}`

	user := fmt.Sprintf(`## Candidate Gitdex Goals
%v

## Candidate Creative Goals
%v

## Existing Goals
%v

## Previously Discarded
%v

Classify the candidates.`,
		gitdexGoals, creativeGoals, existingTitles, discarded)

	resp, err := pr.LLM.Generate(ctx, llm.GenerateRequest{
		Role:   llm.RoleSecondary,
		System: system,
		Prompt: user,
	})
	if err != nil {
		return nil, err
	}

	var result reviewResponse
	if err := jsonfix.RepairAndUnmarshal(resp.Text, &result); err != nil {
		return nil, err
	}

	return &ReviewResult{
		ApprovedGitdexGoals: result.ApprovedGitdex,
		ApprovedCreative:    result.ApprovedCreative,
		Discarded:           result.Discarded,
	}, nil
}
