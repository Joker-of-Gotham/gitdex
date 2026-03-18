package helper

import (
	"context"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/jsonfix"
	promptv2 "github.com/Joker-of-Gotham/gitdex/internal/llm/promptv2"
)

// KnowledgeSelector uses the helper LLM to choose relevant knowledge files.
type KnowledgeSelector struct {
	LLM      llm.LLMProvider
	Store    *dotgitdex.Manager
	Language string
}

type knowledgeSelectionResponse struct {
	SelectedKnowledge []string `json:"selected_knowledge"`
}

// SelectForMaintain uses Prompt A to select knowledge files for maintenance.
func (ks *KnowledgeSelector) SelectForMaintain(ctx context.Context, gitContent, output, index string) ([]string, error) {
	sys, usr := promptv2.BuildPromptA(gitContent, output, index, ks.Language)
	return ks.callAndParse(ctx, sys, usr)
}

// SelectForGoal uses Prompt C to select knowledge files for goal completion.
func (ks *KnowledgeSelector) SelectForGoal(ctx context.Context, gitContent, output, index, goal, todoList string) ([]string, error) {
	sys, usr := promptv2.BuildPromptC(gitContent, output, index, goal, todoList, ks.Language)
	return ks.callAndParse(ctx, sys, usr)
}

func (ks *KnowledgeSelector) callAndParse(ctx context.Context, system, user string) ([]string, error) {
	resp, err := ks.LLM.Generate(ctx, llm.GenerateRequest{
		Role:   llm.RoleSecondary,
		System: system,
		Prompt: user,
	})
	if err != nil {
		return nil, err
	}

	var result knowledgeSelectionResponse
	if err := jsonfix.RepairAndUnmarshal(resp.Text, &result); err != nil {
		return nil, err
	}
	return result.SelectedKnowledge, nil
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
