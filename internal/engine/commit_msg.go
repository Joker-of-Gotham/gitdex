package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
)

const commitMsgSystemPrompt = `You generate Git commit messages.

Rules:
1. Output only the commit message text.
2. Use one line when possible.
3. Keep it under 72 characters.
4. Prefer imperative mood.
5. Use Conventional Commits when the repository already follows that style.
6. Do not add quotes, bullets, markdown, or explanation.

Examples:
feat(auth): add token refresh support
fix(api): handle concurrent request race
docs(readme): update installation steps`

// GenerateCommitMessage uses the secondary (or primary) LLM to generate
// a commit message from staged diff output.
func (p *Pipeline) GenerateCommitMessage(ctx context.Context, state *status.GitState, stagedDiff string) (string, error) {
	if p.llmProvider == nil || !p.llmProvider.IsAvailable(ctx) {
		return "", fmt.Errorf("LLM not available")
	}

	diffSummary := stagedDiff
	if len(stagedDiff) > 4000 {
		diffSummary = stagedDiff[:4000] + "\n... (diff truncated)"
	}

	branchInfo := ""
	if state != nil && state.LocalBranch.Name != "" {
		branchInfo = fmt.Sprintf("Branch: %s\n", state.LocalBranch.Name)
	}

	conventionalHint := ""
	if state != nil && state.CommitSummaryInfo != nil && state.CommitSummaryInfo.UsesConventional {
		conventionalHint = "The repository history already uses Conventional Commits.\n"
	}

	userPrompt := fmt.Sprintf(
		"%s%s\nStaged diff:\n```\n%s\n```\n\nGenerate the best commit message now.",
		branchInfo,
		conventionalHint,
		diffSummary,
	)

	model := p.secondaryModel
	role := llm.RoleSecondary
	if !p.secondaryOn || strings.TrimSpace(p.secondaryModel) == "" {
		model = p.primaryModel
		role = llm.RolePrimary
	}

	resp, err := llm.GenerateText(ctx, p.llmProvider, llm.GenerateRequest{
		Model:       model,
		Role:        role,
		System:      commitMsgSystemPrompt,
		Prompt:      userPrompt,
		Temperature: 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("generate commit message: %w", err)
	}

	msg := cleanCommitMessage(resp.Text)
	if msg == "" {
		return "", fmt.Errorf("LLM returned empty commit message")
	}
	return msg, nil
}

func cleanCommitMessage(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	return strings.TrimSpace(raw)
}
