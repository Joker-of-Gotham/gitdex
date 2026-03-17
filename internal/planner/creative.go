package planner

import (
	"context"
	"encoding/json"

	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	promptv2 "github.com/Joker-of-Gotham/gitdex/internal/llm/promptv2"
)

// CreativePlanner generates new goals using Prompt E.
type CreativePlanner struct {
	LLM      llm.LLMProvider
	Language string
}

// Generate proposes new gitdex and creative goals.
func (p *CreativePlanner) Generate(ctx context.Context, gitContent, output, index, goals, todoList, githubCtx string) (*CreativeOutput, error) {
	sys, usr := promptv2.BuildPromptE(gitContent, output, index, goals, todoList, githubCtx, p.Language)
	resp, err := p.LLM.Generate(ctx, llm.GenerateRequest{
		Role:   llm.RolePrimary,
		System: sys,
		Prompt: usr,
	})
	if err != nil {
		return nil, err
	}

	text := cleanJSON(resp.Text)
	var result CreativeOutput
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, err
	}
	return &result, nil
}
