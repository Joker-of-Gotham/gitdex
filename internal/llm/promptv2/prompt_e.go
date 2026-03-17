package promptv2

import "fmt"

// BuildPromptE constructs the Planner prompt for creative goal generation.
func BuildPromptE(gitContent, output, index, goals, todoList, githubCtx, language string) (system, user string) {
	system = fmt.Sprintf(`Role: Creative goal generator for a Git repository and its GitHub ecosystem.

Task: Analyse repository state, existing goals, GitHub issues/PRs, and propose
valuable new goals in two categories:
A. Gitdex-actionable: executable via git/GitHub operations (maintenance, PR, CI, docs, etc.)
B. Creative/strategic: insights and future directions for the user (not executed)

Rules:
- Each goal must be unique — skip any that overlap with existing or recently completed goals.
- Prioritise high-impact, low-effort goals.
- If no good goals exist, return empty arrays.
- Be conservative: fewer high-quality goals are better than many low-quality ones.

OUTPUT FORMAT (strict JSON, no markdown fences):
{
  "analysis": "brief overview of opportunities spotted",
  "gitdex_goals": ["goal 1", "goal 2", ...],
  "creative_goals": ["insight 1", "insight 2", ...]
}

Respond in %s.`, languageName(language))

	user = fmt.Sprintf(`## Git Context
%s

## Recent Execution Log
%s

## Knowledge Index
%s

## Existing Goals
%s

## Current To-Do List
%s

## GitHub Context
%s

Generate creative and actionable new goals.`,
		gitContent, output, index, goals, todoList, githubCtx)

	return system, user
}
