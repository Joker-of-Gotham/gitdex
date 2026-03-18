package planner

import (
	"context"

	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/jsonfix"
	promptv2 "github.com/Joker-of-Gotham/gitdex/internal/llm/promptv2"
)

// GoalPlanner generates goal-completion suggestions using Prompt D.
type GoalPlanner struct {
	LLM      llm.LLMProvider
	Language string
}

// Plan produces an ordered sequence of actions to advance the goal.
func (p *GoalPlanner) Plan(ctx context.Context, gitContent, output, knowledgeCtx, goal, todoList string) ([]SuggestionItem, string, error) {
	sys, usr := promptv2.BuildPromptD(gitContent, output, knowledgeCtx, goal, todoList, p.Language)
	resp, err := p.LLM.Generate(ctx, llm.GenerateRequest{
		Role:   llm.RolePrimary,
		System: sys,
		Prompt: usr,
	})
	if err != nil {
		return nil, "", err
	}

	var result plannerResponse
	if err := jsonfix.RepairAndUnmarshal(resp.Text, &result); err != nil {
		return nil, "", err
	}
	sanitizeSuggestions(result.Suggestions)
	return result.Suggestions, result.Analysis, nil
}
