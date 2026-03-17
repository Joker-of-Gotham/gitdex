package planner

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	promptv2 "github.com/Joker-of-Gotham/gitdex/internal/llm/promptv2"
)

// MaintenancePlanner generates maintenance suggestions using Prompt B.
type MaintenancePlanner struct {
	LLM      llm.LLMProvider
	Language string
}

// Plan produces an ordered sequence of maintenance actions.
func (p *MaintenancePlanner) Plan(ctx context.Context, gitContent, output, knowledgeCtx string) ([]SuggestionItem, string, error) {
	sys, usr := promptv2.BuildPromptB(gitContent, output, knowledgeCtx, p.Language)
	resp, err := p.LLM.Generate(ctx, llm.GenerateRequest{
		Role:   llm.RolePrimary,
		System: sys,
		Prompt: usr,
	})
	if err != nil {
		return nil, "", err
	}

	text := cleanJSON(resp.Text)
	var result plannerResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, "", err
	}
	sanitizeSuggestions(result.Suggestions)
	return result.Suggestions, result.Analysis, nil
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
